// Copyright © 2021 Cenk Altı

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// The original code repository can be found here: https://github.com/cenkalti/rpc2

package birpc

import (
	"errors"
	"io"
	"sync"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/birpc/internal/svc"
)

// ClientConnector is the connection used in RpcClient, as interface so we can combine the rpc.RpcClient with http one or websocket
type ClientConnector = context.ClientConnector

// BirpcClient represents an RPC BirpcClient.
// There may be multiple outstanding Calls associated
// with a single BirpcClient, and a BirpcClient may be used by
// multiple goroutines simultaneously.
type BirpcClient struct {
	*basicServer
	*basicClient
	codec      BirpcCodec
	server     bool
	disconnect chan struct{}
}

// NewBirpcClient returns a new BirpcClient to handle requests to the
// set of services at the other end of the connection.
// It adds a buffer to the write side of the connection so
// the header and payload are sent as a unit.
func NewBirpcClient(conn io.ReadWriteCloser) *BirpcClient {
	return NewBirpcClientWithCodec(NewGobBirpcCodec(conn))
}

// NewBirpcClientWithCodec is like NewBirpcClient but uses the specified
// codec to encode requests and decode responses.
func NewBirpcClientWithCodec(codec BirpcCodec) *BirpcClient {
	c := &BirpcClient{
		codec:       codec,
		basicServer: newBasicServer(),
		basicClient: newBasicClient(codec),

		disconnect: make(chan struct{}),
	}
	go c.input()
	return c
}

// DisconnectNotify returns a channel that is closed
// when the client connection has gone away.
func (c *BirpcClient) DisconnectNotify() chan struct{} {
	return c.disconnect
}

// input reads messages from codec.
// It reads a reqeust or a response to the previous request.
// If the message is request, calls the handler function.
// If the message is response, sends the reply to the associated call.
func (c *BirpcClient) input() {
	var err error
	var resp Response
	sending := &c.reqMutex
	ctx, cancel := context.WithCancel(context.Background())
	ctx.Client = c
	defer cancel()
	pending := svc.NewPending(ctx)
	wg := new(sync.WaitGroup)
	for err == nil {
		req := c.getRequest()
		resp = Response{}
		if err = c.codec.ReadHeader(req, &resp); err != nil {
			break
		}

		if req.ServiceMethod != "" {
			// request comes to server
			if err := c.readRequest(req, sending, pending, wg); err != nil {
				debugln("birpc: error reading request:", err.Error())
				c.sendResponse(sending, req, invalidRequest, c.codec, err.Error())
				c.freeRequest(req)
			}
		} else {
			c.freeRequest(req)
			// response comes to client
			if err = c.readResponse(&resp); err != nil {
				debugln("birpc: error reading response:", err.Error())
			}
		}
	}
	// Terminate pending calls.
	sending.Lock()
	c.mutex.Lock()
	c.shutdown = true
	closing := c.closing
	if err == io.EOF {
		if closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
	c.mutex.Unlock()
	sending.Unlock()
	if err != io.EOF && !closing && !c.server {
		debugln("birpc: client protocol error:", err)
	}
	wg.Wait()
	close(c.disconnect)
	if !closing {
		c.codec.Close()
	}
}

func (c *BirpcClient) readRequest(req *Request, sending *sync.Mutex, pending *svc.Pending, wg *sync.WaitGroup) error {
	svc, mtype, err := c.getService(req)
	if err != nil {
		return errors.New("birpc: can't find method " + req.ServiceMethod)
	}

	// Decode the argument value.
	argv, argIsValue := getArgv(mtype) // if true, need to indirect before calling.
	// argv guaranteed to be a pointer now.
	if err := c.codec.ReadRequestBody(argv.Interface()); err != nil {
		return err
	}
	if argIsValue {
		argv = argv.Elem()
	}
	replyv := getReplyv(mtype)
	wg.Add(1)
	go svc.call(c.basicServer, sending, pending, wg, mtype, req, argv, replyv, c.codec)

	return nil
}

func (c *BirpcClient) readResponse(resp *Response) error {
	seq := resp.Seq
	c.mutex.Lock()
	call := c.pending[seq]
	delete(c.pending, seq)
	c.mutex.Unlock()

	var err error
	switch {
	case call == nil:
		// We've got no pending call. That usually means that
		// WriteRequest partially failed, and call was already
		// removed; response is a server telling us about an
		// error reading request body. We should still attempt
		// to read error body, but there's no one to give it to.
		err = c.codec.ReadResponseBody(nil)
		if err != nil {
			err = errors.New("reading error body: " + err.Error())
		}
	case resp.Error != "":
		// We've got an error response. Give this to the request;
		// any subsequent requests will get the ReadResponseBody
		// error if there is one.
		call.Error = ServerError(resp.Error)
		err = c.codec.ReadResponseBody(nil)
		if err != nil {
			err = errors.New("reading error body: " + err.Error())
		}
		call.done()
	default:
		err = c.codec.ReadResponseBody(call.Reply)
		if err != nil {
			call.Error = errors.New("reading body " + err.Error())
		}
		call.done()
	}

	return err
}
