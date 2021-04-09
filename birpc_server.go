// Copyright © 2021 Cenk Altı

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// The original code repository can be found here: https://github.com/cenkalti/rpc2

package birpc

import (
	"io"
	"net"

	"github.com/cenkalti/hub"
)

const (
	clientConnected hub.Kind = iota
	clientDisconnected
)

// BirpcServer responds to RPC requests made by Client.
type BirpcServer struct {
	*basicServer
	eventHub *hub.Hub
}

type connectionEvent struct {
	Client ClientConnector
}

type disconnectionEvent struct {
	Client ClientConnector
}

func (connectionEvent) Kind() hub.Kind    { return clientConnected }
func (disconnectionEvent) Kind() hub.Kind { return clientDisconnected }

// NewBirpcServer returns a new BirpcServer.
func NewBirpcServer() *BirpcServer {
	return &BirpcServer{
		basicServer: newBasicServer(),
		eventHub:    &hub.Hub{},
	}
}

// OnConnect registers a function to run when a client connects.
func (s *BirpcServer) OnConnect(f func(ClientConnector)) {
	s.eventHub.Subscribe(clientConnected, func(e hub.Event) {
		go f(e.(connectionEvent).Client)
	})
}

// OnDisconnect registers a function to run when a client disconnects.
func (s *BirpcServer) OnDisconnect(f func(ClientConnector)) {
	s.eventHub.Subscribe(clientDisconnected, func(e hub.Event) {
		go f(e.(disconnectionEvent).Client)
	})
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.  Accept blocks; the caller typically
// invokes it in a go statement.
func (s *BirpcServer) Accept(lis net.Listener) error {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		go s.ServeConn(conn)
	}
}

// ServeConn runs the server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn in a go statement.
// ServeConn uses the gob wire format (see package gob) on the
// connection.  To use an alternate codec, use ServeCodec.
func (s *BirpcServer) ServeConn(conn io.ReadWriteCloser) {
	s.ServeCodec(NewGobBirpcCodec(conn))
}

// ServeCodec is like ServeConn but uses the specified codec to
// decode requests and encode responses.
func (s *BirpcServer) ServeCodec(codec BirpcCodec) {
	defer codec.Close()

	// Client also handles the incoming connections.
	c := &BirpcClient{
		codec:       codec,
		basicServer: s.basicServer,
		basicClient: newBasicClient(codec),

		server:     true,
		disconnect: make(chan struct{}),
	}

	s.eventHub.Publish(connectionEvent{c})
	c.input()
	s.eventHub.Publish(disconnectionEvent{c})
}
