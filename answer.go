package capnp

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"capnproto.org/go/capnp/v3/exc"
	"capnproto.org/go/capnp/v3/internal/str"
	"capnproto.org/go/capnp/v3/internal/syncutil"
)

// A Promise holds the result of an RPC call.  Only one of Fulfill
// or Reject can be called on a Promise.  Before the result is
// written, calls can be queued up using the Answer methods — this is
// promise pipelining.
//
// Promise is most useful for implementing ClientHook.
// Most applications will use Answer, since that what is returned by
// a Client.
type Promise struct {
	method   Method
	resolved <-chan struct{}
	ans      Answer

	// The promise can be in one of the following states:
	//
	//	- Unresolved (initial state).  Can transition to any other state.
	//	- Pending resolution.  Fulfill or Reject has been called on the
	//	  Promise, but Fulfill or Reject is waiting on fulfilling the
	//	  clients and acquiring answers for any ongoing calls to caller.
	//	  All new pipelined calls will block until the Promise is resolved.
	//	  Next state is resolved.
	//	- Resolved.  Fulfill or Reject has finished.

	// mu protects the fields below.  When acquiring multiple Promise.mu
	// mutexes, they must be acquired in traversal order (i.e. p, then
	// p.next, then p.next.next).
	mu sync.Mutex

	// signals is a list of callbacks to invoke on resolution. Has at least
	// one element if the promise is unresolved or pending, nil if resolved.
	signals []func()

	// caller is the hook to make pipelined calls with.  Set to nil once
	// the promise leaves the unresolved state.
	caller PipelineCaller

	// ongoingCalls counts the number of calls to caller that have not
	// yielded an Answer yet (but not necessarily finished).
	ongoingCalls int
	// If callsStopped is non-nil, then the promise has entered into
	// the pending state and is waiting for ongoingCalls to drop to zero.
	// After decrementing ongoingCalls, callsStopped should be closed if
	// ongoingCalls is zero to wake up the goroutine.
	//
	// Only Fulfill or Reject will set callsStopped.
	callsStopped chan struct{}

	// clients is a table of promised clients created to proxy the eventual
	// result's clients.  Even after resolution, this table may still have
	// entries until the clients are released. Cannot be read or written
	// in the pending state.
	clients map[clientPath]*clientAndPromise

	// releasedClients is true after ReleaseClients has been called on this
	// promise.  Only the receiver of ReleaseClients should set this to true.
	releasedClients bool

	// result and err are the values from Fulfill or Reject respectively
	// in the resolved state.
	result Ptr
	err    error
}

type clientAndPromise struct {
	client  Client
	promise *ClientPromise
}

// NewPromise creates a new unresolved promise.  The PipelineCaller will
// be used to make pipelined calls before the promise resolves.
func NewPromise(m Method, pc PipelineCaller) *Promise {
	if pc == nil {
		panic("NewPromise(nil)")
	}
	resolved := make(chan struct{})
	p := &Promise{
		method:   m,
		resolved: resolved,
		signals:  []func(){func() { close(resolved) }},
		caller:   pc,
	}
	p.ans.f.promise = p
	p.ans.metadata = *NewMetadata()
	return p
}

// isUnresolved reports whether p is in the unresolved state.
// The caller must be holding onto p.mu.
func (p *Promise) isUnresolved() bool {
	return p.caller != nil
}

// isPendingResolution reports whether p is in the pending resolution
// state.  The caller must be holding onto p.mu.
func (p *Promise) isPendingResolution() bool {
	return p.caller == nil && len(p.signals) > 0
}

// isResolved reports whether p is in the resolved state.
// The caller must be holding onto p.mu.
func (p *Promise) isResolved() bool {
	return len(p.signals) == 0
}

// resolution returns p's resolution.  The return value is invalid
// unless p is in the resolved state.  The caller must be holding onto
// p.mu.
func (p *Promise) resolution() resolution {
	return resolution{p.method, p.result, p.err}
}

// Fulfill resolves the promise with a successful result.
//
// Fulfill will wait for any outstanding calls to the underlying
// PipelineCaller to yield Answers and any pipelined clients to be
// fulfilled.
func (p *Promise) Fulfill(result Ptr) {
	p.Resolve(result, nil)
}

// Reject resolves the promise with a failure.
//
// Reject will wait for any outstanding calls to the underlying
// PipelineCaller to yield Answers and any pipelined clients to be
// fulfilled.
func (p *Promise) Reject(e error) {
	if e == nil {
		panic("Promise.Reject(nil)")
	}
	p.Resolve(Ptr{}, e)
}

// Resolve resolves the promise.
//
// If e != nil, then this is equivalent to p.Reject(e).
// Otherwise, it is equivalent to p.Fulfill(r).
func (p *Promise) Resolve(r Ptr, e error) {
	var shutdownPromises []*ClientPromise
	syncutil.With(&p.mu, func() {
		if e != nil {
			p.requireUnresolved("Reject")
		} else {
			p.requireUnresolved("Fulfill")
		}
		p.caller = nil

		if p.ongoingCalls > 0 {
			p.callsStopped = make(chan struct{})
		}
	})

	if len(p.clients) > 0 || p.ongoingCalls > 0 {
		// Pending resolution state: wait for clients to be fulfilled
		// and calls to have answers.  p.clients cannot be touched in the
		// pending resolution state, so we have exclusive access to the
		// variable.
		res := resolution{p.method, r, e}
		for path, cp := range p.clients {
			t := path.transform()
			cp.promise.fulfill(res.client(t))
			shutdownPromises = append(shutdownPromises, cp.promise)
			cp.promise = nil
		}
		if p.callsStopped != nil {
			<-p.callsStopped
		}
	}

	syncutil.With(&p.mu, func() {
		// Move p into resolved state.
		p.callsStopped = nil
		p.result, p.err = r, e
		for _, f := range p.signals {
			f()
		}
		p.signals = nil
	})
	for _, promise := range shutdownPromises {
		promise.shutdown()
	}
}

// requireUnresolved is a helper method for checking for duplicate
// calls to Fulfill() or Reject(); panics if the promise is not in
// the unresolved state.
//
// The callerMethod argument should be the name of the method which
// is invoking requireUnresolved. The panic message will report this
// value as well as the method that originally resolved the promise,
// and which method (Fulfill or Reject) was used to resolve it.
func (p *Promise) requireUnresolved(callerMethod string) {
	if !p.isUnresolved() {
		var prevMethod string
		if p.err == nil {
			prevMethod = "Fulfill"
		} else {
			prevMethod = "Reject (error = " + strconv.Quote(p.err.Error()) + ")"
		}

		panic("Promise." + callerMethod +
			" called after previous call to " + prevMethod)
	}
}

// Answer returns a read-only view of the promise.
// Answer may be called concurrently by multiple goroutines.
func (p *Promise) Answer() *Answer {
	return &p.ans
}

// ReleaseClients waits until p is resolved and then closes any proxy
// clients created by the promise's answer.  Failure to call this method
// will result in capability leaks.  After the first call, subsequent
// calls to ReleaseClients do nothing.  It is safe to call
// ReleaseClients concurrently from multiple goroutines.
//
// This method is typically used in a ReleaseFunc.
func (p *Promise) ReleaseClients() {
	<-p.resolved
	p.mu.Lock()
	if p.releasedClients {
		p.mu.Unlock()
		return
	}
	p.releasedClients = true // must happen before traversing pointers
	clients := p.clients
	p.clients = nil
	p.mu.Unlock()
	for _, cp := range clients {
		cp.client.Release()
	}
}

// A PipelineCaller implements promise pipelining.
//
// See the counterpart methods in ClientHook for a description.
type PipelineCaller interface {
	PipelineSend(ctx context.Context, transform []PipelineOp, s Send) (*Answer, ReleaseFunc)
	PipelineRecv(ctx context.Context, transform []PipelineOp, r Recv) PipelineCaller
}

// An Answer is a deferred result of a client call.  Conceptually, this is a
// future.  It is safe to use from multiple goroutines.
type Answer struct {
	f        Future
	metadata Metadata
}

// ErrorAnswer returns a Answer that always returns error e.
func ErrorAnswer(m Method, e error) *Answer {
	p := &Promise{
		method:   m,
		resolved: closedSignal,
		err:      e,
	}
	p.ans.f.promise = p
	return &p.ans
}

// ImmediateAnswer returns an Answer that accesses s.
func ImmediateAnswer(m Method, s Struct) *Answer {
	p := &Promise{
		method:   m,
		resolved: closedSignal,
		result:   s.ToPtr(),
	}
	p.ans.f.promise = p
	p.ans.metadata = *NewMetadata()
	return &p.ans
}

// Future returns a future that is equivalent to ans.
func (ans *Answer) Future() *Future {
	return &ans.f
}

// Metadata returns a metadata map where callers can store information
// about the answer
func (ans *Answer) Metadata() *Metadata {
	return &ans.metadata
}

// Done returns a channel that is closed when the answer's call is finished.
func (ans *Answer) Done() <-chan struct{} {
	return ans.f.Done()
}

// Struct waits until the answer is resolved and returns the struct
// this answer represents.
func (ans *Answer) Struct() (Struct, error) {
	return ans.f.Struct()
}

// List waits until the answer is resolved and returns the list
// this answer represents.
func (ans *Answer) List() (List, error) {
	return ans.f.List()
}

// Client returns the answer as a client.  If the answer's originating
// call has not completed, then calls will be queued until the original
// call's completion.  The client reference is borrowed: the caller
// should not call Close.
func (ans *Answer) Client() Client {
	return ans.f.Client()
}

// Field returns a derived future which yields the pointer field given,
// defaulting to the value given.
func (ans *Answer) Field(off uint16, def []byte) *Future {
	return ans.f.Field(off, def)
}

// PipelineSend starts a pipelined call.
func (ans *Answer) PipelineSend(ctx context.Context, transform []PipelineOp, s Send) (*Answer, ReleaseFunc) {
	p := ans.f.promise
	p.mu.Lock()
	switch {
	case p.isUnresolved():
		p.ongoingCalls++
		caller := p.caller
		p.mu.Unlock()
		ans, release := caller.PipelineSend(ctx, transform, s)
		syncutil.With(&p.mu, func() {
			p.ongoingCalls--
			if p.ongoingCalls == 0 && p.callsStopped != nil {
				close(p.callsStopped)
			}
		})
		return ans, release
	case p.isPendingResolution():
		// Block new calls until resolved.
		p.mu.Unlock()
		select {
		case <-p.resolved:
		case <-ctx.Done():
			return ErrorAnswer(s.Method, ctx.Err()), func() {}
		}
		p.mu.Lock()
		fallthrough
	case p.isResolved():
		r := p.resolution()
		p.mu.Unlock()
		return r.client(transform).SendCall(ctx, s)
	default:
		panic("unreachable")
	}
}

// PipelineRecv starts a pipelined call.
func (ans *Answer) PipelineRecv(ctx context.Context, transform []PipelineOp, r Recv) PipelineCaller {
	p := ans.f.promise
	p.mu.Lock()
	switch {
	case p.isUnresolved():
		p.ongoingCalls++
		caller := p.caller
		p.mu.Unlock()
		pcall := caller.PipelineRecv(ctx, transform, r)
		syncutil.With(&p.mu, func() {
			p.ongoingCalls--
			if p.ongoingCalls == 0 && p.callsStopped != nil {
				close(p.callsStopped)
			}
		})
		return pcall
	case p.isPendingResolution():
		// Block new calls until resolved.
		p.mu.Unlock()
		select {
		case <-p.resolved:
		case <-ctx.Done():
			r.Reject(ctx.Err())
			return nil
		}
		p.mu.Lock()
		fallthrough
	case p.isResolved():
		res := p.resolution()
		p.mu.Unlock()
		return res.client(transform).RecvCall(ctx, r)
	default:
		panic("unreachable")
	}
}

// A Future accesses a portion of an Answer.  It is safe to use from
// multiple goroutines.
type Future struct {
	promise *Promise
	parent  *Future // nil if root future
	op      PipelineOp
}

// transform returns the operations needed to transform the root answer
// into the value f represents.
func (f *Future) transform() []PipelineOp {
	if f.parent == nil {
		return nil
	}
	n := 0
	for g := f; g.parent != nil; g = g.parent {
		n++
	}
	xform := make([]PipelineOp, n)
	for i, g := n-1, f; g.parent != nil; i, g = i-1, g.parent {
		xform[i] = g.op
	}
	return xform
}

// Done returns a channel that is closed when the answer's call is finished.
func (f *Future) Done() <-chan struct{} {
	return f.promise.resolved
}

// Ptr waits until the answer is resolved and returns the pointer
// this future represents.
func (f *Future) Ptr() (Ptr, error) {
	p := f.promise
	<-p.resolved
	p.mu.Lock()
	r := p.resolution()
	p.mu.Unlock()
	return r.ptr(f.transform())
}

// Struct waits until the answer is resolved and returns the struct
// this answer represents.
func (f *Future) Struct() (Struct, error) {
	p, err := f.Ptr()
	return p.Struct(), err
}

// List waits until the answer is resolved and returns the list
// this answer represents.
func (f *Future) List() (List, error) {
	p, err := f.Ptr()
	return p.List(), err
}

// Client returns the future as a client.  If the answer's originating
// call has not completed, then calls will be queued until the original
// call's completion.  The client reference is borrowed: the caller
// should not call Release.
func (f *Future) Client() Client {
	p := f.promise
	p.mu.Lock()
	switch {
	case p.isUnresolved():
		ft := f.transform()
		cpath := clientPathFromTransform(ft)
		if cp := p.clients[cpath]; cp != nil {
			return cp.client
		}
		c, pr := NewPromisedClient(PipelineClient{
			p:         p,
			transform: ft,
		})
		if p.clients == nil {
			p.clients = make(map[clientPath]*clientAndPromise)
		}
		p.clients[cpath] = &clientAndPromise{c, pr}
		p.mu.Unlock()
		return c
	case p.isPendingResolution():
		syncutil.Without(&p.mu, func() {
			<-p.resolved
		})
		fallthrough
	case p.isResolved():
		r := p.resolution()
		p.mu.Unlock()
		return r.client(f.transform())
	default:
		panic("unreachable")
	}
}

// Field returns a derived future which yields the pointer field given,
// defaulting to the value given.
func (f *Future) Field(off uint16, def []byte) *Future {
	return &Future{
		promise: f.promise,
		parent:  f,
		op: PipelineOp{
			Field:        off,
			DefaultValue: def,
		},
	}
}

// PipelineClient implements ClientHook by calling to the pipeline's answer.
type PipelineClient struct {
	p         *Promise
	transform []PipelineOp
}

func (pc PipelineClient) Answer() *Answer {
	return pc.p.Answer()
}

func (pc PipelineClient) Transform() []PipelineOp {
	return pc.transform
}

func (pc PipelineClient) Send(ctx context.Context, s Send) (*Answer, ReleaseFunc) {
	return pc.p.ans.PipelineSend(ctx, pc.transform, s)
}

func (pc PipelineClient) Recv(ctx context.Context, r Recv) PipelineCaller {
	return pc.p.ans.PipelineRecv(ctx, pc.transform, r)
}

func (pc PipelineClient) Brand() Brand {
	select {
	case <-pc.p.resolved:
		pc.p.mu.Lock()
		r := pc.p.resolution()
		pc.p.mu.Unlock()
		return r.client(pc.transform).State().Brand
	default:
		return Brand{Value: pc}
	}
}

func (pc PipelineClient) Shutdown() {
}

// A PipelineOp describes a step in transforming a pipeline.
// It maps closely with the PromisedAnswer.Op struct in rpc.capnp.
type PipelineOp struct {
	Field        uint16
	DefaultValue []byte
}

// String returns a human-readable description of op.
func (op PipelineOp) String() string {
	s := make([]byte, 0, 32)
	s = append(s, "get field "...)
	s = strconv.AppendInt(s, int64(op.Field), 10)
	if op.DefaultValue == nil {
		return string(s)
	}
	s = append(s, " with default"...)
	return string(s)
}

// Transform applies a sequence of pipeline operations to a pointer
// and returns the result.
func Transform(p Ptr, transform []PipelineOp) (Ptr, error) {
	n := len(transform)
	if n == 0 {
		return p, nil
	}
	s := p.Struct()
	for i, op := range transform[:n-1] {
		field, err := s.Ptr(op.Field)
		if err != nil {
			return Ptr{}, newTransformError(i, op.Field, err, false)
		}
		s, err = field.StructDefault(op.DefaultValue)
		if err != nil {
			return Ptr{}, newTransformError(i, op.Field, err, true)
		}
	}
	op := transform[n-1]
	p, err := s.Ptr(op.Field)
	if err != nil {
		return Ptr{}, newTransformError(n-1, op.Field, err, false)
	}
	if op.DefaultValue != nil {
		p, err = p.Default(op.DefaultValue)
		if err != nil {
			return Ptr{}, newTransformError(n-1, op.Field, err, true)
		}
		return p, nil
	}
	return p, nil
}

func newTransformError(index int, field uint16, err error, withDefault bool) error {
	msg := "transform: op " + str.Itod(index) + ": pointer field " + str.Utod(field)
	if withDefault {
		msg += " with default"
	}
	return exc.WrapError(msg, err)
}

// A resolution is the outcome of a future.
type resolution struct {
	method Method
	result Ptr
	err    error
}

// ptr obtains a Ptr by applying a transform.
func (r resolution) ptr(transform []PipelineOp) (Ptr, error) {
	if r.err != nil {
		return Ptr{}, exc.Annotate("", r.method.String(), r.err)
	}
	p, err := Transform(r.result, transform)
	if err != nil {
		return Ptr{}, exc.Annotate("", r.method.String(), err)
	}
	return p, nil
}

// client obtains a Client by applying a transform.
func (r resolution) client(transform []PipelineOp) Client {
	p, err := r.ptr(transform)
	if err != nil {
		return ErrorClient(err)
	}
	iface := p.Interface()
	if p.IsValid() && !iface.IsValid() {
		return ErrorClient(errors.New("not a capability"))
	}
	return iface.Client()
}

// clientPath is an encoded version of a list of pipeline operations.
// It is suitable as a map key.
//
// It specifically ignores default values, because a capability can't have a
// default value other than null.
type clientPath string

func clientPathFromTransform(ops []PipelineOp) clientPath {
	buf := make([]byte, 0, len(ops)*2)
	for i := range ops {
		f := ops[i].Field
		buf = append(buf, byte(f&0x00ff), byte(f&0xff00>>8))
	}
	return clientPath(buf)
}

func (cp clientPath) transform() []PipelineOp {
	ops := make([]PipelineOp, len(cp)/2)
	for i := range ops {
		ops[i].Field = uint16(cp[i*2]) | uint16(cp[i*2+1])<<8
	}
	return ops
}
