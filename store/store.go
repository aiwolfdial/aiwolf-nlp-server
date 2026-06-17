// Package store abstracts persistence for orchestration state so the default
// file-based behaviour can later be swapped (e.g. for SQLite) without touching
// the callers.
package store

// MatchOptimizerStore persists the match-optimizer's arena scheduling state.
// The default implementation reads and writes the same JSON the optimizer has
// always produced.
type MatchOptimizerStore interface {
	// Load returns the raw persisted bytes, or an error (e.g. os.ErrNotExist)
	// when nothing has been saved yet.
	Load() ([]byte, error)
	// Save persists the raw bytes.
	Save(data []byte) error
}
