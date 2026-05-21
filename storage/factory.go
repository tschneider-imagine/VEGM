package storage

import "fmt"

func NewIndex(backend, sqlitePath string) (Index, error) {
	switch backend {
	case "", "noop":
		return &NoopIndex{}, nil
	case "sqlite":
		return NewSQLiteIndex(sqlitePath), nil
	default:
		return nil, fmt.Errorf("unsupported storage backend %q", backend)
	}
}
