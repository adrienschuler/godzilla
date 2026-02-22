package main

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/adrienschuler/godzilla/gen/presence"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func startTestServer(t *testing.T) pb.PresenceServiceClient {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	pb.RegisterPresenceServiceServer(srv, &server{store: newStore()})
	go srv.Serve(lis)
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return pb.NewPresenceServiceClient(conn)
}

func TestConnectDisconnect(t *testing.T) {
	client := startTestServer(t)
	ctx := context.Background()

	// Connect alice
	resp, err := client.UserConnected(ctx, &pb.UserRequest{Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Usernames) != 1 || resp.Usernames[0] != "alice" {
		t.Fatalf("expected [alice], got %v", resp.Usernames)
	}

	// Connect bob
	resp, err = client.UserConnected(ctx, &pb.UserRequest{Username: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Usernames) != 2 {
		t.Fatalf("expected 2 online users, got %d", len(resp.Usernames))
	}

	// Disconnect alice
	_, err = client.UserDisconnected(ctx, &pb.UserRequest{Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	// Verify only bob remains
	online, err := client.GetOnlineUsers(ctx, &pb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(online.Usernames) != 1 || online.Usernames[0] != "bob" {
		t.Fatalf("expected [bob], got %v", online.Usernames)
	}
}

func TestMultipleConnections(t *testing.T) {
	client := startTestServer(t)
	ctx := context.Background()

	// Connect alice twice (multiple tabs)
	client.UserConnected(ctx, &pb.UserRequest{Username: "alice"})
	client.UserConnected(ctx, &pb.UserRequest{Username: "alice"})

	// Disconnect once — should still be online
	client.UserDisconnected(ctx, &pb.UserRequest{Username: "alice"})
	online, _ := client.GetOnlineUsers(ctx, &pb.Empty{})
	if len(online.Usernames) != 1 {
		t.Fatalf("expected alice still online, got %v", online.Usernames)
	}

	// Disconnect again — now offline
	client.UserDisconnected(ctx, &pb.UserRequest{Username: "alice"})
	online, _ = client.GetOnlineUsers(ctx, &pb.Empty{})
	if len(online.Usernames) != 0 {
		t.Fatalf("expected no online users, got %v", online.Usernames)
	}
}

func TestTyping(t *testing.T) {
	client := startTestServer(t)
	ctx := context.Background()

	client.UserConnected(ctx, &pb.UserRequest{Username: "alice"})

	// Set typing
	_, err := client.SetTyping(ctx, &pb.SetTypingRequest{Username: "alice", IsTyping: true})
	if err != nil {
		t.Fatal(err)
	}

	typing, _ := client.GetTypingUsers(ctx, &pb.Empty{})
	if len(typing.Usernames) != 1 || typing.Usernames[0] != "alice" {
		t.Fatalf("expected [alice] typing, got %v", typing.Usernames)
	}

	// Stop typing
	client.SetTyping(ctx, &pb.SetTypingRequest{Username: "alice", IsTyping: false})
	typing, _ = client.GetTypingUsers(ctx, &pb.Empty{})
	if len(typing.Usernames) != 0 {
		t.Fatalf("expected no typing users, got %v", typing.Usernames)
	}
}

func TestTypingExpiry(t *testing.T) {
	s := newStore()
	s.connect("alice")
	s.setTyping("alice", true)

	if len(s.typingUsers()) != 1 {
		t.Fatal("expected alice typing")
	}

	// Manually backdate the typing timestamp
	s.mu.Lock()
	s.typing["alice"] = time.Now().Add(-9 * time.Second)
	s.mu.Unlock()

	// Wait for cleanup tick
	time.Sleep(1500 * time.Millisecond)

	if len(s.typingUsers()) != 0 {
		t.Fatal("expected typing to have expired")
	}
}
