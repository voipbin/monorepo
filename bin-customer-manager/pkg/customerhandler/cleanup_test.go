package customerhandler

import (
	"testing"
	"time"
)

func TestCleanupConstants(t *testing.T) {
	if cleanupInterval != 15*time.Minute {
		t.Errorf("cleanupInterval = %v, expected %v", cleanupInterval, 15*time.Minute)
	}
	if unverifiedMaxAge != time.Hour {
		t.Errorf("unverifiedMaxAge = %v, expected %v", unverifiedMaxAge, time.Hour)
	}
}
