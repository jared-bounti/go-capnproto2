package rpc

import (
	"context"
	"errors"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/exc"
	"capnproto.org/go/capnp/v3/internal/rc"
	"capnproto.org/go/capnp/v3/internal/syncutil"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
)

// An answerID is an index into the answers table.
type answerID uint32

// answer is an entry in a Conn's answer table.
type answer struct {
	// c and id must be set before any answer methods are called.
	c  *Conn
	id answerID

	// cancel cancels the Context used in the received method call.
	// May be nil.
	cancel context.CancelFunc

	// ret is the outgoing Return struct.  ret is valid iff there was no
	// error creating the message.  If ret is invalid, then this answer
	// entry is a placeholder until the remote vat cancels the call.
	ret rpccp.Return

	// sendMsg sends the return message.  The caller MUST hold ans.c.lk.
	sendMsg func()

	// msgReleaser releases the return message when its refcount hits zero.
	// The caller MUST NOT hold ans.c.lk.
	msgReleaser *rc.Releaser

	// results is the memoized answer to ret.Results().
	// Set by AllocResults and setBootstrap, but contents can only be read
	// if flags has resultsReady but not finishReceived set.
	results rpccp.Payload

	// All fields below are protected by s.c.mu.

	// flags is a bitmask of events that have occurred in an answer's
	// lifetime.
	flags answerFlags

	// exportRefs is the number of references to exports placed in the
	// results.
	exportRefs map[exportID]uint32

	// pcall is the PipelineCaller returned by RecvCall.  It will be set
	// to nil once results are ready.
	pcall capnp.PipelineCaller
	// promise is a promise wrapping pcall. It will be resolved, and
	// subsequently set to nil, once the results are ready.
	promise *capnp.Promise

	// pcalls is added to for every pending RecvCall and subtracted from
	// for every RecvCall return (delivery acknowledgement).  This is used
	// to satisfy the Returner.Return contract.
	pcalls sync.WaitGroup

	// err is the error passed to (*answer).sendException or from creating
	// the Return message.  Can only be read after resultsReady is set in
	// flags.
	err error
}

type answerFlags uint8

const (
	returnSent answerFlags = 1 << iota
	finishReceived
	resultsReady
	releaseResultCapsFlag
)

// flags.Contains(flag) Returns true iff flags contains flag, which must
// be a single flag.
func (flags answerFlags) Contains(flag answerFlags) bool {
	return flags&flag != 0
}

// errorAnswer returns a placeholder answer with an error result already set.
func errorAnswer(c *Conn, id answerID, err error) *answer {
	return &answer{
		c:     c,
		id:    id,
		err:   err,
		flags: resultsReady | returnSent,
	}
}

// newReturn creates a new Return message. The returned Releaser will release the message when
// all references to it are dropped; the caller is responsible for one reference. This will not
// happen before the message is sent, as the returned send function retains a reference.
func (c *Conn) newReturn() (_ rpccp.Return, sendMsg func(), _ *rc.Releaser, _ error) {
	outMsg, err := c.transport.NewMessage()
	if err != nil {
		return rpccp.Return{}, nil, nil, rpcerr.WrapFailed("create return", err)
	}
	ret, err := outMsg.Message.NewReturn()
	if err != nil {
		outMsg.Release()
		return rpccp.Return{}, nil, nil, rpcerr.WrapFailed("create return", err)
	}

	// Before releasing the message, we need to wait both until it is sent and
	// until the local vat is done with it.  We therefore implement a simple
	// ref-counting mechanism such that 'release' must be called twice before
	// 'releaseMsg' is called.
	releaser := rc.NewReleaser(2, outMsg.Release)

	return ret, func() {
		c.lk.sendTx.Send(asyncSend{
			send:    outMsg.Send,
			release: releaser.Decr,
			onSent: func(err error) {
				if err != nil {
					c.er.ReportError(exc.WrapError("send return", err))
				}
			},
		})
	}, releaser, nil
}

// setPipelineCaller sets ans.pcall to pcall if the answer has not
// already returned.  The caller MUST NOT hold ans.c.lk.
//
// This also sets ans.promise to a new promise, wrapping pcall.
func (ans *answer) setPipelineCaller(m capnp.Method, pcall capnp.PipelineCaller) {
	syncutil.With(&ans.c.lk, func() {
		if !ans.flags.Contains(resultsReady) {
			ans.pcall = pcall
			ans.promise = capnp.NewPromise(m, pcall)
		}
	})
}

// AllocResults allocates the results struct.
func (ans *answer) AllocResults(sz capnp.ObjectSize) (capnp.Struct, error) {
	var err error
	ans.results, err = ans.ret.NewResults()
	if err != nil {
		return capnp.Struct{}, rpcerr.WrapFailed("alloc results", err)
	}
	s, err := capnp.NewStruct(ans.results.Segment(), sz)
	if err != nil {
		return capnp.Struct{}, rpcerr.WrapFailed("alloc results", err)
	}
	if err := ans.results.SetContent(s.ToPtr()); err != nil {
		return capnp.Struct{}, rpcerr.WrapFailed("alloc results", err)
	}
	ans.msgReleaser.Incr()
	return s, nil
}

// setBootstrap sets the results to an interface pointer, stealing the
// reference.
func (ans *answer) setBootstrap(c capnp.Client) error {
	if ans.ret.HasResults() || len(ans.ret.Message().CapTable) > 0 {
		panic("setBootstrap called after creating results")
	}
	// Add the capability to the table early to avoid leaks if setBootstrap fails.
	ans.ret.Message().CapTable = []capnp.Client{c}

	var err error
	ans.results, err = ans.ret.NewResults()
	if err != nil {
		return rpcerr.WrapFailed("alloc bootstrap results", err)
	}
	iface := capnp.NewInterface(ans.results.Segment(), 0)
	if err := ans.results.SetContent(iface.ToPtr()); err != nil {
		return rpcerr.WrapFailed("alloc bootstrap results", err)
	}
	return nil
}

// PrepareReturn implements capnp.Returner.PrepareReturn
func (ans *answer) PrepareReturn(e error) {
	rl := &releaseList{}
	defer rl.Release()

	ans.c.withLocked(func(c *lockedConn) {
		if e == nil {
			ans.prepareSendReturn(c, rl)
		} else {
			ans.prepareSendException(c, rl, e)
		}
	})
}

// Return implements capnp.Returner.Return
func (ans *answer) Return() {
	rl := &releaseList{}
	defer rl.Release()
	defer ans.pcalls.Wait()

	var err error
	ans.c.withLocked(func(c *lockedConn) {
		if ans.err == nil {
			err = ans.completeSendReturn(c, rl)
		} else {
			ans.completeSendException(c, rl)
		}
	})
	ans.c.tasks.Done() // added by handleCall

	if err == nil {
		return
	}

	if err = ans.c.shutdown(err); err != nil {
		ans.c.er.ReportError(err)
	}
}

func (ans *answer) ReleaseResults() {
	if ans.results.IsValid() {
		ans.msgReleaser.Decr()
	}
}

// sendReturn sends the return message with results allocated by a
// previous call to AllocResults.  If the answer already received a
// Finish with releaseResultCaps set to true, then sendReturn returns
// the number of references to be subtracted from each export.
//
// The caller MUST be holding onto ans.c.lk, and the lockedConn parameter
// must be equal to ans.c (it exists to make it hard to forget to acquire
// the lock, per usual).
//
// sendReturn MUST NOT be called if sendException was previously called.
func (ans *answer) sendReturn(c *lockedConn, rl *releaseList) error {
	ans.prepareSendReturn(c, rl)
	return ans.completeSendReturn(c, rl)
}

func (ans *answer) prepareSendReturn(c *lockedConn, rl *releaseList) {
	c.assertIs(ans.c)

	var err error
	ans.exportRefs, err = c.fillPayloadCapTable(ans.results)
	if err != nil {
		ans.c.er.ReportError(rpcerr.Annotate(err, "send return"))
	}
	// Continue.  Don't fail to send return if cap table isn't fully filled.

	select {
	case <-ans.c.bgctx.Done():
		// We're not going to send the message after all, so don't forget to release it.
		ans.msgReleaser.Decr()
		ans.sendMsg = nil
	default:
	}
}

func (ans *answer) completeSendReturn(c *lockedConn, rl *releaseList) error {
	c.assertIs(ans.c)

	ans.pcall = nil
	ans.flags |= resultsReady

	fin := ans.flags.Contains(finishReceived)
	if ans.sendMsg != nil {
		if ans.promise != nil {
			if fin {
				// Can't use ans.result after a finish, but it's
				// ok to return an error if the finish comes in
				// before the return. Possible enhancement: use
				// the cancel variant of return.
				ans.promise.Reject(rpcerr.Failed(errors.New("received finish before return")))
			} else {
				ans.promise.Resolve(ans.results.Content())
			}
			ans.promise = nil
		}
		ans.sendMsg()
	}

	ans.flags |= returnSent
	if fin {
		return ans.destroy(c, rl)
	}
	return nil
}

// sendException sends an exception on the answer's return message.
//
// The caller MUST be holding onto ans.c.lk. sendException MUST NOT
// be called if sendReturn was previously called.
func (ans *answer) sendException(c *lockedConn, rl *releaseList, ex error) {
	ans.prepareSendException(c, rl, ex)
	ans.completeSendException(c, rl)
}

func (ans *answer) prepareSendException(c *lockedConn, rl *releaseList, ex error) {
	c.assertIs(ans.c)
	ans.err = ex

	select {
	case <-ans.c.bgctx.Done():
	default:
		// Send exception.
		if e, err := ans.ret.NewException(); err != nil {
			ans.c.er.ReportError(exc.WrapError("send exception", err))
			ans.sendMsg = nil
		} else {
			e.SetType(rpccp.Exception_Type(exc.TypeOf(ex)))
			if err := e.SetReason(ex.Error()); err != nil {
				ans.c.er.ReportError(exc.WrapError("send exception", err))
				ans.sendMsg = nil
			}
		}
	}
}

func (ans *answer) completeSendException(c *lockedConn, rl *releaseList) {
	c.assertIs(ans.c)

	ex := ans.err
	ans.pcall = nil
	ans.flags |= resultsReady

	if ans.promise != nil {
		ans.promise.Reject(ex)
		ans.promise = nil
	}
	if ans.sendMsg != nil {
		ans.sendMsg()
	}
	ans.flags |= returnSent
	if ans.flags.Contains(finishReceived) {
		// destroy will never return an error because sendException does
		// create any exports.
		_ = ans.destroy(c, rl)
	}
}

// destroy removes the answer from the table and returns ReleaseFuncs to
// run. The answer must have sent a return and received a finish.
// The caller must be holding onto ans.c.lk.
//
// shutdown has its own strategy for cleaning up an answer.
func (ans *answer) destroy(c *lockedConn, rl *releaseList) error {
	c.assertIs(ans.c)

	rl.Add(ans.msgReleaser.Decr)
	delete(c.lk.answers, ans.id)
	if !ans.flags.Contains(releaseResultCapsFlag) || len(ans.exportRefs) == 0 {
		return nil

	}
	return c.releaseExportRefs(rl, ans.exportRefs)
}
