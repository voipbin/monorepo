package analysishandler

import "errors"

var (
	// ErrActiveflowNotEnded is returned by Start when the activeflow is not in ended state.
	ErrActiveflowNotEnded = errors.New("activeflow is not ended")
	// ErrReanalyzeCooldown is returned by Start when a reanalyze is requested
	// within the cooldown window of the last update.
	ErrReanalyzeCooldown = errors.New("reanalyze is on cooldown")
	// ErrNotFound is the masked not-found returned for absent OR cross-customer records.
	ErrNotFound = errors.New("not found")
)
