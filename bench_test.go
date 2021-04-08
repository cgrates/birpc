package rpc

import (
	contextg "context"
	"log"
	"net"
	"testing"

	"github.com/cgrates/rpc/birpc"
	"github.com/cgrates/rpc/context"
)

func BenchmarkBirpcInArgs(b *testing.B) {
	newServer := birpc.NewBirpcServer()
	newServer.Register(new(birpc.Airth3))
	l, addr := listenTCP()
	log.Println("NewServer test RPC server listening on", newServerAddr)
	go newServer.Accept(l)

	c, err := net.Dial("tcp", addr)
	if err != nil {
		b.Fatal(err)
	}
	client := birpc.NewBirpcClient(c)
	defer client.Close()
	client.Register(new(birpc.Airth3))
	ctx := contextg.Background()

	// Synchronous calls
	args := &birpc.Args3{7, 8}
	reply := new(birpc.Reply3)
	b.ResetTimer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.Call(ctx, "Airth3.Add", args, reply); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBirpcInContext(b *testing.B) {
	newServer := NewBirpcServer()
	newServer.Register(new(Airth2))
	l, addr := listenTCP()
	log.Println("NewServer test RPC server listening on", newServerAddr)
	go newServer.Accept(l)

	c, err := net.Dial("tcp", addr)
	if err != nil {
		b.Fatal(err)
	}
	client := NewBirpcClient(c)
	defer client.Close()
	client.Register(new(Airth2))
	ctx := context.Background()

	// Synchronous calls
	args := &Args2{7, 8}
	reply := new(Reply2)
	b.ResetTimer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.Call(ctx, "Airth2.Add", args, reply); err != nil {
			b.Fatal(err)
		}
	}
}
