package birpc_test

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/cgrates/birpc"
	"github.com/cgrates/birpc/context"
	"github.com/cgrates/birpc/jsonrpc"
)

type Echo struct{}

func (*Echo) Echo(_ *context.Context, args int, reply *int) error {
	*reply = args
	return nil
}

func TestUnknownMethodKeepsConnection(t *testing.T) {
	for _, codec := range []struct {
		name string
		new  func(io.ReadWriteCloser) birpc.BirpcCodec
	}{
		{"gob", birpc.NewGobBirpcCodec},
		{"json", jsonrpc.NewJSONBirpcCodec},
	} {
		t.Run(codec.name, func(t *testing.T) {
			serverConn, clientConn := net.Pipe()
			defer serverConn.Close()
			defer clientConn.Close()
			deadline := time.Now().Add(5 * time.Second)
			if err := serverConn.SetDeadline(deadline); err != nil {
				t.Fatal(err)
			}
			if err := clientConn.SetDeadline(deadline); err != nil {
				t.Fatal(err)
			}

			server := birpc.NewBirpcServer()
			server.Register(new(Echo))
			serverDone := make(chan struct{})
			go func() {
				server.ServeCodec(codec.new(serverConn))
				close(serverDone)
			}()
			client := birpc.NewBirpcClientWithCodec(codec.new(clientConn))
			defer func() {
				client.Close()
				serverConn.Close()
				<-serverDone
				<-client.DisconnectNotify()
			}()

			for i := 1; i <= 3; i++ {
				var reply int
				err := client.Call(context.TODO(), "Echo.unknown", i, &reply)
				if err != birpc.ServerError("birpc: can't find method Echo.unknown") {
					t.Fatalf("unknown call: got %v", err)
				}

				if err := client.Call(context.TODO(), "Echo.Echo", i, &reply); err != nil {
					t.Fatalf("call after unknown method: %v", err)
				}
				if reply != i {
					t.Fatalf("echo reply: got %d, want %d", reply, i)
				}
			}
		})
	}
}
