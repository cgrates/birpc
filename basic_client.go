// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package birpc

import (
	"log"
	"sync"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/birpc/internal/svc"
)

type writeClientCodec interface {
	WriteRequest(*Request, any) error
	Close() error
}

// NewClientWithCodec is like NewClient but uses the specified
// codec to encode requests and decode responses.
func newBasicClient(c writeClientCodec) *basicClient {
	return &basicClient{
		wc:      c,
		pending: make(map[uint64]*Call),
	}
}

// basicClient represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type basicClient struct {
	wc       writeClientCodec
	reqMutex sync.Mutex // protects following
	request  Request

	mutex    sync.Mutex // protects following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

func (client *basicClient) send(call *Call) {
	client.reqMutex.Lock()
	defer client.reqMutex.Unlock()

	// Register this call.
	client.mutex.Lock()
	if client.shutdown || client.closing {
		client.mutex.Unlock()
		call.Error = ErrShutdown
		call.done()
		return
	}
	if call.seq != 0 {
		// It has already been canceled, don't bother sending
		call.Error = context.Canceled
		client.mutex.Unlock()
		call.done()
		return
	}
	client.seq++
	seq := client.seq
	call.seq = seq
	client.pending[seq] = call
	client.mutex.Unlock()

	// Encode and send the request.
	client.request.Seq = seq
	client.request.ServiceMethod = call.ServiceMethod
	err := client.wc.WriteRequest(&client.request, call.Args)
	if err != nil {
		client.mutex.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Close calls the underlying codec's Close method. If the connection is already
// shutting down, ErrShutdown is returned.
func (client *basicClient) Close() error {
	client.mutex.Lock()
	if client.closing {
		client.mutex.Unlock()
		return ErrShutdown
	}
	client.closing = true
	client.mutex.Unlock()
	return client.wc.Close()
}

// Go invokes the function asynchronously. It returns the Call structure representing
// the invocation. The done channel will signal when the call is complete by returning
// the same Call object. If done is nil, Go will allocate a new channel.
// If non-nil, done must be buffered or Go will deliberately crash.
func (client *basicClient) Go(serviceMethod string, args any, reply any, done chan *Call) *Call {
	call := new(Call)
	call.ServiceMethod = serviceMethod
	call.Args = args
	call.Reply = reply
	if done == nil {
		done = make(chan *Call, 10) // buffered.
	} else {
		// If caller passes done != nil, it must arrange that
		// done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		if cap(done) == 0 {
			log.Panic("rpc: done channel is unbuffered")
		}
	}
	call.Done = done
	client.send(call)
	return call
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (client *basicClient) Call(ctx *context.Context, serviceMethod string, args any, reply any) error {
	ch := make(chan *Call, 2) // 2 for this call and cancel
	call := client.Go(serviceMethod, args, reply, ch)
	select {
	case <-call.Done:
		return call.Error
	case <-ctx.Done():
		// Cancel the pending request on the client
		client.mutex.Lock()
		seq := call.seq
		_, ok := client.pending[seq]
		delete(client.pending, seq)
		if seq == 0 {
			// hasn't been sent yet, non-zero will prevent send
			call.seq = 1
		}
		client.mutex.Unlock()

		// Cancel running request on the server
		if seq != 0 && ok {
			client.Go("_goRPC_.Cancel", &svc.CancelArgs{Seq: seq}, nil, ch)
		}
		return ctx.Err()
	}
}
