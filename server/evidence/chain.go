// Package evidence provides the hash-chained evidence-plane interface.
// P6 replaces the Noop stub with a MongoDB-backed Ed25519-signed chain.
package evidence

import "context"

// Entry is one evidence plane record.
type Entry struct {
	ActorEmail string
	Action     string
	Payload    map[string]any
}

// Chain appends tamper-evident entries. Nil-safe: Append on a nil Chain is
// always a no-op error-free, so callers don't need nil-guards.
type Chain interface {
	Append(ctx context.Context, e Entry) error
}

// Noop discards every entry. P6 replaces this with the hash-chained writer.
type Noop struct{}

func (n *Noop) Append(_ context.Context, _ Entry) error { return nil }
