package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/cgrates/rpc"
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
	client := rpc.ClientValueFromContext(ctx)
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

	srv := rpc.NewBirpcServer()
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

	clt := rpc.NewBirpcClientWithCodec(NewJSONBirpcCodec(conn))
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
