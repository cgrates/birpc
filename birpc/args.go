package birpc

import (
	"context"
	"errors"
	"fmt"
)

type Args3 struct{ A, B int }
type Reply3 int
type Airth3 struct{}

func (*Airth3) Add(ctx context.Context, client ClientConnector, args *Args3, reply *Reply3) error {
	*reply = Reply3(args.A + args.B)

	var rep Reply3
	if client == nil {
		return errors.New("expected client not nil")
	}
	err := client.Call(ctx, "Airth3.Mult", &Args3{2, 3}, &rep)
	if err != nil {
		return err
	}

	if rep != 6 {
		return fmt.Errorf("not expected: %d", rep)
	}

	return nil
}

func (*Airth3) Mult(_ context.Context, client ClientConnector, args *Args3, reply *Reply3) error {
	*reply = Reply3(args.A * args.B)
	return nil
}
