package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/konglong87/go-e2e/internal/computerbridge"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

func TestDesktopBridgeBindsApprovedConversationAndCleansSocket(t *testing.T) {
	// Keep the fixture below the repo, short enough for Darwin sun_path.
	relative, err := os.MkdirTemp("..", ".cb-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(relative) })
	root, err := filepath.Abs(relative)
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := testComputerManager()
	ctx := context.Background()
	t.Cleanup(func() { _ = m.close(ctx) })
	listener, err := startDesktopComputerBridge(ctx, root, m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	client, err := computerbridge.NewClient(listener.Config())
	if err != nil {
		t.Fatal(err)
	}
	owner := cu.SessionOwner{TenantID: 7, UserID: 11, SessionID: 93}
	if _, err := client.Lookup(ctx, owner); err == nil {
		t.Fatal("implicit approval")
	}
	approved, err := m.startOwnedWithLifetime(ctx, ctx, ComputerSessionStartInput{Approved: true}, owner)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := client.Lookup(ctx, owner); err != nil || got != approved.ID {
		t.Fatal(got, err)
	}
	if _, err := m.control(ctx, approved.ID, cu.ActionStop); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Lookup(ctx, owner); err == nil {
		t.Fatal("native Stop did not revoke model lookup")
	}
	path := listener.Config().SocketPath
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("socket not cleaned", err)
	}
	if _, err := client.Lookup(ctx, owner); err == nil {
		t.Fatal("closed bridge accepted request")
	}
}
