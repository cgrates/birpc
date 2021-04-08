package svc

import (
	"sync"

	"context"
)

// ClientConnector is the connection used in RpcClient, as interface so we can combine the rpc.RpcClient with http one or websocket
type ClientConnector interface {
	Call(ctx context.Context, serviceMethod string, args, reply interface{}) error
}

// Pending manages a map of all pending requests to a rpc.Service for a
// connection (an rpc.ServerCodec).
type Pending struct {
	mu     sync.Mutex
	m      map[uint64]context.CancelFunc // seq -> cancel
	parent context.Context
}

func NewPending(parent context.Context) *Pending {
	return &Pending{
		m:      make(map[uint64]context.CancelFunc),
		parent: parent,
	}
}

func (s *Pending) Start(seq uint64) context.Context {
	ctx, cancel := context.WithCancel(s.parent)
	s.mu.Lock()
	// we assume seq is not already in map. If not, the client is broken.
	s.m[seq] = cancel
	s.mu.Unlock()
	return ctx
}

func (s *Pending) Cancel(seq uint64) {
	s.mu.Lock()
	cancel, ok := s.m[seq]
	if ok {
		delete(s.m, seq)
	}
	s.mu.Unlock()
	if ok {
		cancel()
	}
}

type CancelArgs struct {
	// Seq is the sequence number for the rpc.Call to cancel.
	Seq uint64

	// pending is the DS used by rpc.Server to track the ongoing calls for
	// this connection. It should not be set by the client, the Service will
	// set it.
	pending *Pending
}

// SetPending sets the pending map for the server to use. Do not use on the
// client.
func (a *CancelArgs) SetPending(p *Pending) {
	a.pending = p
}

// GoRPC is an internal service used by rpc.
type GoRPC struct{}

func (*GoRPC) Cancel2(_ context.Context, _ ClientConnector, args *CancelArgs, _ *bool) error {
	args.pending.Cancel(args.Seq)
	return nil
}
