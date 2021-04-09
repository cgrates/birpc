// Copyright © 2021 Cenk Altı

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// The original code repository can be found here: https://github.com/cenkalti/rpc2

package rpc

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/cgrates/rpc/context"
)

const (
	network = "tcp4"
	addr    = "127.0.0.1:5000"
)

type Args2 struct{ A, B int }
type Reply2 int
type Airth2 struct{}

func (*Airth2) Add(ctx context.Context, args *Args2, reply *Reply2) error {
	*reply = Reply2(args.A + args.B)

	var rep Reply2
	client := ctx.Client
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

func (*Airth2) Mult(client context.Context, args *Args2, reply *Reply2) error {
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
