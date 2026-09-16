package main

import "testing"

func TestNewServerTokenIsHighEntropyAndUnique(t *testing.T) {
	first, err := newServerToken()
	if err != nil {
		t.Fatalf("newServerToken() error = %v", err)
	}
	second, err := newServerToken()
	if err != nil {
		t.Fatalf("newServerToken() second error = %v", err)
	}
	if len(first) != 64 || len(second) != 64 {
		t.Fatalf("token lengths = %d and %d, want 64", len(first), len(second))
	}
	if first == second {
		t.Fatal("newServerToken() returned the same token twice")
	}
}
