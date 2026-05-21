package storage

// NoopIndex is the default placeholder implementation until a vendored SQLite backend is added.
type NoopIndex struct{}

func (n *NoopIndex) Initialize() error { return nil }

func (n *NoopIndex) WriteEvent(EventRecord) error { return nil }

func (n *NoopIndex) Query(Query) ([]EventRecord, error) { return nil, nil }

func (n *NoopIndex) Close() error { return nil }
