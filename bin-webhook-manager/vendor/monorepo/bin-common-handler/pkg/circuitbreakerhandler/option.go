package circuitbreakerhandler

import (
	"errors"
	"time"
)

const (
	defaultFailureThreshold = 5
	defaultOpenDuration     = 30 * time.Second
)

var ErrCircuitOpen = errors.New("circuit breaker is open")
