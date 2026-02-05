package server

import (
	"time"
)

// timePtr parses a timestamp string and returns *time.Time for use in test fixtures.
// It supports both ISO8601 (6 decimal places) and RFC3339Nano formats.
// Panics if the string cannot be parsed, which is acceptable in tests.
func timePtr(s string) *time.Time {
	layouts := []string{
		"2006-01-02T15:04:05.000000Z",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	panic("timePtr: cannot parse timestamp: " + s)
}
