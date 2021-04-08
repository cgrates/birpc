package rpcc

import (
	"context"
	"net"
	"testing"
)

const (
	network = "tcp4"
	addr    = "127.0.0.1:5000"
)

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
