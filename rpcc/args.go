package rpcc

import (
	"context"
	"errors"
	"fmt"
)

type Args2 struct{ A, B int }
type Reply2 int
type Airth2 struct{}

func (*Airth2) Add(ctx context.Context, args *Args2, reply *Reply2) error {
	*reply = Reply2(args.A + args.B)

	var rep Reply2
	client := ClientValueFromContext(ctx)
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
