package main

import (
	"context"
	"testing"
)

func TestComputerRuntimeOutlivesWindowContext(t *testing.T) {
	windowCtx, cancelWindow := context.WithCancel(context.Background())
	runtime := newComputerRuntime(context.Background())
	defer func() { _ = runtime.Stop(context.Background()) }()

	cancelWindow()
	select {
	case <-runtime.Context().Done():
		t.Fatal("computer runtime was coupled to the window context")
	default:
	}
	if windowCtx.Err() == nil {
		t.Fatal("window context did not cancel")
	}
	if runtime.Manager() == nil || runtime.Manager().runtimeContext() != runtime.Context() {
		t.Fatal("manager is not bound to computerRuntimeCtx")
	}
}
