package task

import "testing"

func TestNewID(t *testing.T) {
	id := NewID()
	if id == "" {
		t.Fatal("NewID returned empty string")
	}
	if len(id) != 26 {
		t.Fatalf("ULID should be 26 chars, got %d: %s", len(id), id)
	}

	// IDs should be unique
	id2 := NewID()
	if id == id2 {
		t.Fatal("two consecutive NewID calls returned the same value")
	}
}
