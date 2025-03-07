// Package server provides runtime support for implementing Cap'n Proto
// interfaces locally.
package server // import "capnproto.org/go/capnp/v3/server"

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/exc"
	"capnproto.org/go/capnp/v3/exp/mpsc"
)

// A Method describes a single capability method on a server object.
type Method struct {
	capnp.Method
	Impl func(context.Context, *Call) error
}

// Call holds the state of an ongoing capability method call.
// A Call cannot be used after the server method returns.
type Call struct {
	ctx    context.Context
	method *Method
	recv   capnp.Recv
	aq     *answerQueue
	srv    *Server

	alloced bool
	results capnp.Struct

	acked bool
}

// Args returns the call's arguments.  Args is not safe to
// reference after a method implementation returns.  Args is safe to
// call and read from multiple goroutines.
func (c *Call) Args() capnp.Struct {
	return c.recv.Args
}

// AllocResults allocates the results struct.  It is an error to call
// AllocResults more than once.
func (c *Call) AllocResults(sz capnp.ObjectSize) (capnp.Struct, error) {
	if c.alloced {
		return capnp.Struct{}, newError("multiple calls to AllocResults")
	}
	var err error
	c.alloced = true
	c.results, err = c.recv.Returner.AllocResults(sz)
	return c.results, err
}

// Go is a function that is called to unblock future calls; by default
// a server only accepts one method call at a time, waiting until
// the method returns before servicing the next method in the queue.
// calling Go spawns another goroutine to service additional Calls
// in the queue, allowing the current goroutine to block, do expensive
// computation, etc. without holding up other calls. If Go is called,
// the calling Goroutine exits when the method returns, so that there
// is never more than one goroutine pulling things from the queue.
//
// Go need not be the first call in a function nor is it required.
// short functions can return without calling Go.
func (c *Call) Go() {
	if c.acked {
		return
	}
	c.acked = true
	go c.srv.handleCalls(c.srv.handleCallsCtx)
}

// Shutdowner is the interface that wraps the Shutdown method.
type Shutdowner interface {
	Shutdown()
}

// A Server is a locally implemented interface.  It implements the
// capnp.ClientHook interface.
type Server struct {
	methods  sortedMethods
	brand    any
	shutdown Shutdowner

	// Cancels handleCallsCtx
	cancelHandleCalls context.CancelFunc

	// Context used by the goroutine running handleCalls(). Note
	// the calls themselves will have different contexts, which
	// are not children of this context, but are supplied by
	// start().
	handleCallsCtx context.Context

	// wg is incremented each time a method is queued, and
	// decremented after it is handled.
	wg sync.WaitGroup

	// Calls are inserted into this queue, to be handled
	// by a goroutine running handleCalls()
	callQueue *mpsc.Queue[*Call]

	// Handler for custom behavior of unknown methods
	HandleUnknownMethod func(m capnp.Method) *Method
}

// New returns a client hook that makes calls to a set of methods.
// If shutdown is nil then the server's shutdown is a no-op.  The server
// guarantees message delivery order by blocking each call on the
// return of the previous call or a call to Call.Go.
func New(methods []Method, brand any, shutdown Shutdowner) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	srv := &Server{
		methods:           make(sortedMethods, len(methods)),
		brand:             brand,
		shutdown:          shutdown,
		callQueue:         mpsc.New[*Call](),
		cancelHandleCalls: cancel,
		handleCallsCtx:    ctx,
	}
	copy(srv.methods, methods)
	sort.Sort(srv.methods)
	go srv.handleCalls(ctx)
	return srv
}

// Send starts a method call.
func (srv *Server) Send(ctx context.Context, s capnp.Send) (*capnp.Answer, capnp.ReleaseFunc) {
	mm := srv.methods.find(s.Method)
	if mm == nil && srv.HandleUnknownMethod != nil {
		mm = srv.HandleUnknownMethod(s.Method)
	}
	if mm == nil {
		return capnp.ErrorAnswer(s.Method, capnp.Unimplemented("unimplemented")), func() {}
	}
	args, err := sendArgsToStruct(s)
	if err != nil {
		return capnp.ErrorAnswer(mm.Method, err), func() {}
	}
	ret := new(structReturner)
	pcaller := srv.start(ctx, mm, capnp.Recv{
		Method: mm.Method, // pick up names from server method
		Args:   args,
		ReleaseArgs: func() {
			if msg := args.Message(); msg != nil {
				msg.Reset(nil)
				args = capnp.Struct{}
			}
		},
		Returner: ret,
	})
	return ret.answer(mm.Method, pcaller)
}

// Recv starts a method call.
func (srv *Server) Recv(ctx context.Context, r capnp.Recv) capnp.PipelineCaller {
	mm := srv.methods.find(r.Method)
	if mm == nil && srv.HandleUnknownMethod != nil {
		mm = srv.HandleUnknownMethod(r.Method)
	}
	if mm == nil {
		r.Reject(capnp.Unimplemented("unimplemented"))
		return nil
	}
	return srv.start(ctx, mm, r)
}

func (srv *Server) handleCalls(ctx context.Context) {
	for {
		call, err := srv.callQueue.Recv(ctx)
		if err != nil {
			// Context has been canceled; drain the rest of the queue,
			// invoking handleCall() with the cancelled context to
			// trigger cleanup.
			var ok bool
			call, ok = srv.callQueue.TryRecv()
			if !ok {
				return
			}
		}

		// The context for the individual call is not necessarily
		// related to the context managing the server's lifetime
		// (ctx); we need to monitor both and pass the call a
		// context that will be canceled if *either* context is
		// cancelled.
		callCtx, cancelCall := context.WithCancel(call.ctx)
		go func() {
			defer cancelCall()
			select {
			case <-callCtx.Done():
			case <-ctx.Done():
			}
		}()
		func() {
			defer cancelCall()
			srv.handleCall(callCtx, call)
		}()

		if call.acked {
			// Another goroutine has taken over; time
			// to retire.
			return
		}
	}
}

func (srv *Server) handleCall(ctx context.Context, c *Call) {
	defer srv.wg.Done()

	err := c.method.Impl(ctx, c)

	c.recv.ReleaseArgs()
	c.recv.Returner.PrepareReturn(err)
	if err == nil {
		c.aq.fulfill(c.results)
	} else {
		c.aq.reject(err)
	}
	c.recv.Returner.Return()
	c.recv.Returner.ReleaseResults()
}

func (srv *Server) start(ctx context.Context, m *Method, r capnp.Recv) capnp.PipelineCaller {
	srv.wg.Add(1)

	aq := newAnswerQueue(r.Method)
	srv.callQueue.Send(&Call{
		ctx:    ctx,
		method: m,
		recv:   r,
		aq:     aq,
		srv:    srv,
	})
	return aq
}

// Brand returns a value that will match IsServer.
func (srv *Server) Brand() capnp.Brand {
	return capnp.Brand{Value: serverBrand{srv.brand}}
}

// Shutdown waits for ongoing calls to finish and calls Shutdown on the
// Shutdowner passed into NewServer.  Shutdown must not be called more
// than once.
func (srv *Server) Shutdown() {
	srv.cancelHandleCalls()
	srv.wg.Wait()
	if srv.shutdown != nil {
		srv.shutdown.Shutdown()
	}
}

// IsServer reports whether a brand returned by capnp.Client.Brand
// originated from Server.Brand, and returns the brand argument passed
// to New.
func IsServer(brand capnp.Brand) (_ any, ok bool) {
	sb, ok := brand.Value.(serverBrand)
	return sb.x, ok
}

type serverBrand struct {
	x any
}

func sendArgsToStruct(s capnp.Send) (capnp.Struct, error) {
	if s.PlaceArgs == nil {
		return capnp.Struct{}, nil
	}
	st, err := newBlankStruct(s.ArgsSize)
	if err != nil {
		return capnp.Struct{}, err
	}
	if err := s.PlaceArgs(st); err != nil {
		st.Message().Reset(nil)
		// Using fmt.Errorf to ensure sendArgsToStruct returns a generic error.
		return capnp.Struct{}, fmt.Errorf("place args: %w", err)
	}
	return st, nil
}

func newBlankStruct(sz capnp.ObjectSize) (capnp.Struct, error) {
	_, seg, err := capnp.NewMessage(capnp.MultiSegment(nil))
	if err != nil {
		return capnp.Struct{}, err
	}
	st, err := capnp.NewRootStruct(seg, sz)
	if err != nil {
		return capnp.Struct{}, err
	}
	return st, nil
}

type sortedMethods []Method

// find returns the method with the given ID or nil.
func (sm sortedMethods) find(id capnp.Method) *Method {
	i := sort.Search(len(sm), func(i int) bool {
		m := &sm[i]
		if m.InterfaceID != id.InterfaceID {
			return m.InterfaceID >= id.InterfaceID
		}
		return m.MethodID >= id.MethodID
	})
	if i == len(sm) {
		return nil
	}
	m := &sm[i]
	if m.InterfaceID != id.InterfaceID || m.MethodID != id.MethodID {
		return nil
	}
	return m
}

func (sm sortedMethods) Len() int {
	return len(sm)
}

func (sm sortedMethods) Less(i, j int) bool {
	if id1, id2 := sm[i].InterfaceID, sm[j].InterfaceID; id1 != id2 {
		return id1 < id2
	}
	return sm[i].MethodID < sm[j].MethodID
}

func (sm sortedMethods) Swap(i, j int) {
	sm[i], sm[j] = sm[j], sm[i]
}

type resultsAllocer interface {
	AllocResults(capnp.ObjectSize) (capnp.Struct, error)
}

func newError(msg string) error {
	return exc.New(exc.Failed, "capnp server", msg)
}
