package birpc

import (
	"io"
	"log"
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
func (s *BirpcServer) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print("rpc.Serve: accept:", err.Error())
			return
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
