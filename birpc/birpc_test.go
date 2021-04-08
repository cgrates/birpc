package birpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
)

const (
	network = "tcp4"
	addr    = "127.0.0.1:5000"
)

type Args2 struct{ A, B int }
type Reply2 int
type Airth2 struct{}

func (*Airth2) Add(ctx context.Context, client ClientConnector, args *Args2, reply *Reply2) error {
	*reply = Reply2(args.A + args.B)

	var rep Reply2
	if client == nil {
		return errors.New("expected client not nil")
	}
	err := client.Call(ctx, "Airth2.Mult", &Args2{2, 3}, &rep)
	if err != nil {
		return err
	}

	if rep != 6 {
		return fmt.Errorf("not expected: %d", rep)
	}

	return nil
}

func (*Airth2) Mult(_ context.Context, client ClientConnector, args *Args2, reply *Reply2) error {
	*reply = Reply2(args.A * args.B)
	return nil
}
func TestTCPGOB(t *testing.T) {
	lis, err := net.Listen(network, addr)
	if err != nil {
		t.Fatal(err)
	}

	srv := NewBirpcServer()
	srv.Register(new(Airth2))
	go srv.Accept(lis)

	conn, err := net.Dial(network, addr)
	if err != nil {
		t.Fatal(err)
	}

	clt := NewBirpcClient(conn)
	clt.Register(new(Airth2))
	defer clt.Close()

	// Test Call.
	DebugLog = true
	var rep Reply2
	err = clt.Call(context.TODO(), "Airth2.Add", &Args2{1, 2}, &rep)
	if err != nil {
		t.Fatal(err)
	}
	if rep != 3 {
		t.Fatalf("not expected: %d", rep)
	}
	// Test undefined method.
	err = clt.Call(context.TODO(), "Airth2.Foo", 1, &rep)
	if err.Error() != "birpc: can't find method Airth2.Foo" {
		t.Fatal(err)
	}
}
