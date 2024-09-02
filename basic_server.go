// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package birpc

import (
	"errors"
	"strings"
	"sync"

	"github.com/cgrates/birpc/internal/svc"
)

type writeServerCodec interface {
	WriteResponse(*Response, any) error
}

func newBasicServer() (bs *basicServer) {
	bs = new(basicServer)
	bs.RegisterName("_goRPC_", &svc.GoRPC{})
	return
}

// Server represents an RPC Server.
type basicServer struct {
	serviceMap sync.Map   // map[string]*service
	reqLock    sync.Mutex // protects freeReq
	freeReq    *Request
	respLock   sync.Mutex // protects freeResp
	freeResp   *Response
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//   - exported method of exported type
//   - two arguments, both of exported type
//   - the second argument is a pointer
//   - one return value, of type error
//
// It returns an error if the receiver is not an exported type or has
// no suitable methods. It also logs the error using package log.
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (server *basicServer) Register(rcvr any) error {
	return server.register(rcvr, "", false)
}

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func (server *basicServer) RegisterName(name string, rcvr any) error {
	return server.register(rcvr, name, true)
}

func (server *basicServer) register(rcvr any, name string, useName bool) (err error) {
	var srv *Service
	var isService bool
	if srv, isService = rcvr.(*Service); !isService { // is already defined as a service
		if srv, err = NewService(rcvr, name, useName); err != nil {
			return
		}
	}
	if _, dup := server.serviceMap.LoadOrStore(srv.Name, srv); dup {
		return errors.New("rpc: service already defined: " + srv.Name)
	}
	return
}

// UnregisterName remove the service from the server
// It returns an error if the service was not registered
// This can be used to update the server dynamically
// by bringing the service down or replacing it without
// replacing the server
func (server *basicServer) UnregisterName(name string) error {
	if _, loaded := server.serviceMap.LoadAndDelete(name); loaded {
		return errors.New("rpc: service not defined: " + name)
	}
	return nil
}

func (server *basicServer) getRequest() *Request {
	server.reqLock.Lock()
	req := server.freeReq
	if req == nil {
		req = new(Request)
	} else {
		server.freeReq = req.next
		*req = Request{}
	}
	server.reqLock.Unlock()
	return req
}

func (server *basicServer) freeRequest(req *Request) {
	server.reqLock.Lock()
	req.next = server.freeReq
	server.freeReq = req
	server.reqLock.Unlock()
}

func (server *basicServer) getResponse() *Response {
	server.respLock.Lock()
	resp := server.freeResp
	if resp == nil {
		resp = new(Response)
	} else {
		server.freeResp = resp.next
		*resp = Response{}
	}
	server.respLock.Unlock()
	return resp
}

func (server *basicServer) freeResponse(resp *Response) {
	server.respLock.Lock()
	resp.next = server.freeResp
	server.freeResp = resp
	server.respLock.Unlock()
}

func (server *basicServer) getService(req *Request) (svc *Service, mtype *MethodType, err error) {
	dot := strings.LastIndex(req.ServiceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
		return
	}
	serviceName := req.ServiceMethod[:dot]
	methodName := req.ServiceMethod[dot+1:]

	// Look up the request.
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc: can't find service " + req.ServiceMethod)
		return
	}
	svc = svci.(*Service)
	mtype = svc.Methods[methodName]
	if mtype == nil {
		err = errors.New("rpc: can't find method " + req.ServiceMethod)
	}
	return
}

// A value sent as a placeholder for the server's response value when the server
// receives an invalid request. It is never decoded by the client since the Response
// contains an error when it is used.
var invalidRequest = struct{}{}

func (server *basicServer) sendResponse(sending *sync.Mutex, req *Request, reply any, codec writeServerCodec, errmsg string) {
	resp := server.getResponse()
	// Encode the response header
	if errmsg != "" {
		resp.Error = errmsg
		reply = invalidRequest
	}
	resp.Seq = req.Seq
	sending.Lock()
	err := codec.WriteResponse(resp, reply)
	if err != nil {
		debugln("rpc: writing response:", err)
	}
	sending.Unlock()
	server.freeResponse(resp)
}
