package birpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/cgrates/birpc"
)

type contextClient int

func (*contextClient) Call(context.Context, string, any, any) error {
	return nil
}

func TestClientFromContext(t *testing.T) {
	first, second := new(contextClient), new(contextClient)
	parent := birpc.WithClient(context.Background(), first)
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want birpc.ClientConnector
	}{
		{name: "absent", ctx: context.Background()},
		{name: "attached", ctx: parent, want: first},
		{name: "replaced", ctx: birpc.WithClient(parent, second), want: second},
		{name: "cleared", ctx: birpc.WithClient(parent, nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := birpc.ClientFromContext(tc.ctx); got != tc.want {
				t.Fatalf("got client %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClientFromContextWithTimeout(t *testing.T) {
	type valueKey struct{}
	client := new(contextClient)
	ctx := birpc.WithClient(context.Background(), client)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	ctx = context.WithValue(ctx, valueKey{}, 42)
	if got := birpc.ClientFromContext(ctx); got != client {
		t.Fatalf("got client %v, want %v", got, client)
	}
}

type ContextService struct {
	ctx context.Context
}

func (s *ContextService) Check(ctx context.Context, _ *struct{}, _ *bool) error {
	s.ctx = ctx
	return nil
}

func TestServiceCallContext(t *testing.T) {
	handler := new(ContextService)
	service, err := birpc.NewService(handler, "", false)
	if err != nil {
		t.Fatal(err)
	}
	ctx := birpc.WithClient(context.Background(), service)
	var reply bool
	if err := service.Call(ctx, "ContextService.Check", &struct{}{}, &reply); err != nil {
		t.Fatal(err)
	}
	if handler.ctx != ctx {
		t.Fatal("internal dispatch changed the context")
	}
}
