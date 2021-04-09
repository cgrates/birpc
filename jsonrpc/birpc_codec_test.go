// Copyright © 2021 Cenk Altı

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// The original code repository can be found here: https://github.com/cenkalti/rpc2
package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/cgrates/birpc"
	"github.com/cgrates/birpc/context"
)

const (
	network = "tcp4"
	addr    = "127.0.0.1:5000"
)

type Args2 struct{ A, B int }
type Reply2 int
type Airth2 struct {
	number chan int
}

func (*Airth2) Add(ctx context.Context, args *Args2, reply *Reply2) error {
	*reply = Reply2(args.A + args.B)

	var rep Reply2
	client := ctx.Client
	if client == nil {
		return errors.New("expected client not nil")
	}
	err := client.Call(ctx, "Airth2.Mult", Args2{2, 3}, &rep)
	if err != nil {
		return err
	}

	if rep != 6 {
		return fmt.Errorf("not expected: %d", rep)
	}

	return nil
}

func (a *Airth2) Set(client context.Context, i int, _ *struct{}) error {
	a.number <- i
	return nil
}

func (*Airth2) Mult(client context.Context, args *Args2, reply *Reply2) error {
	*reply = Reply2(args.A * args.B)
	return nil
}

func (*Airth2) AddPos(ctx context.Context, args []interface{}, result *float64) error {
	*result = args[0].(float64) + args[1].(float64)
	return nil
}
func (*Airth2) RawArgs(ctx context.Context, args []json.RawMessage, reply *[]string) error {
	for _, p := range args {
		var str string
		json.Unmarshal(p, &str)
		*reply = append(*reply, str)
	}
	return nil
}
func (*Airth2) TypedArgs(ctx context.Context, args []int, reply *[]string) error {
	for _, p := range args {
		*reply = append(*reply, fmt.Sprintf("%d", p))
	}
	return nil
}

func TestJSONRPC(t *testing.T) {

	lis, err := net.Listen(network, addr)
	if err != nil {
		t.Fatal(err)
	}

	srv := birpc.NewBirpcServer()
	number := make(chan int, 1)
	srv.Register(&Airth2{number: number})

	go func() {
		conn, err := lis.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		srv.ServeCodec(NewJSONBirpcCodec(conn))
	}()

	conn, err := net.Dial(network, addr)
	if err != nil {
		t.Fatal(err)
	}

	clt := birpc.NewBirpcClientWithCodec(NewJSONBirpcCodec(conn))
	clt.Register(&Airth2{number: number})

	// Test Call.
	var rep Reply2
	err = clt.Call(context.TODO(), "Airth2.Add", Args{1, 2}, &rep)
	if err != nil {
		t.Fatal(err)
	}
	if rep != 3 {
		t.Fatalf("not expected: %d", rep)
	}
	// Test undefined method.
	err = clt.Call(context.TODO(), "Airth2.foo", 1, &rep)
	if err.Error() != "birpc: can't find method Airth2.foo" {
		t.Fatal(err)
	}

	// Test Positional arguments.
	var result float64
	err = clt.Call(context.TODO(), "Airth2.AddPos", []interface{}{1, 2}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result != 3 {
		t.Fatalf("not expected: %f", result)
	}
	testArgs := func(expected, reply []string) error {
		if len(reply) != len(expected) {
			return fmt.Errorf("incorrect reply length: %d", len(reply))
		}
		for i := range expected {
			if reply[i] != expected[i] {
				return fmt.Errorf("not expected reply[%d]: %s", i, reply[i])
			}
		}
		return nil
	}

	// Test raw arguments (partial unmarshal)
	var reply []string
	var expected []string = []string{"arg1", "arg2"}
	rawArgs := json.RawMessage(`["arg1", "arg2"]`)
	err = clt.Call(context.TODO(), "Airth2.RawArgs", rawArgs, &reply)
	if err != nil {
		t.Fatal(err)
	}

	if err = testArgs(expected, reply); err != nil {
		t.Fatal(err)
	}

	// Test typed arguments
	reply = []string{}
	expected = []string{"1", "2"}
	typedArgs := []int{1, 2}
	err = clt.Call(context.TODO(), "Airth2.TypedArgs", typedArgs, &reply)
	if err != nil {
		t.Fatal(err)
	}
	if err = testArgs(expected, reply); err != nil {
		t.Fatal(err)
	}
}
