package context

import (
	"context"
	"time"
)

var (
	Canceled         = context.Canceled
	DeadlineExceeded = context.DeadlineExceeded
	background       = Context{Context: context.Background()}
	todo             = Context{Context: context.TODO()}
)

func Background() Context {
	return background
}

func TODO() Context {
	return todo
}

// ClientConnector is the connection used in RpcClient, as interface so we can combine the rpc.RpcClient with http one or websocket
type ClientConnector interface {
	Call(ctx Context, serviceMethod string, args, reply interface{}) error
}

type CancelFunc = context.CancelFunc

type Context struct {
	context.Context

	Client ClientConnector
}

func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	ctx.Client = parent.Client
	ctx.Context, cancel = context.WithCancel(parent.Context)
	return
}
func WithDeadline(parent Context, d time.Time) (ctx Context, cancel CancelFunc) {
	ctx.Client = parent.Client
	ctx.Context, cancel = context.WithDeadline(parent.Context, d)
	return
}
func WithTimeout(parent Context, timeout time.Duration) (ctx Context, cancel CancelFunc) {
	ctx.Client = parent.Client
	ctx.Context, cancel = context.WithTimeout(parent.Context, timeout)
	return
}

func WithValue(parent Context, key, val interface{}) (ctx Context) {
	ctx.Client = parent.Client
	ctx.Context = context.WithValue(parent.Context, key, val)
	return
}
