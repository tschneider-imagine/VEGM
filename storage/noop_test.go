package storage

import "testing"

func TestNoopIndex(t *testing.T) {
	idx := &NoopIndex{}
	if err := idx.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := idx.WriteEvent(EventRecord{InstanceID: "vegm-001", MessageType: "keepAlive"}); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}
	results, err := idx.Query(Query{MessageType: "keepAlive", Limit: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if results != nil && len(results) != 0 {
		t.Fatalf("expected nil or empty results from noop index, got %#v", results)
	}
	if err := idx.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
