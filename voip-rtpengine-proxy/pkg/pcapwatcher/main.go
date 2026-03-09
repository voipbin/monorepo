package pcapwatcher

import "context"

//go:generate mockgen -source=main.go -destination=mock_main.go -package=pcapwatcher

// Watcher watches for completed RTPEngine recordings and processes them.
type Watcher interface {
	// Run starts watching. Blocks until context is cancelled.
	Run(ctx context.Context) error
}
