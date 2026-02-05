package testhelper

import (
	"time"
)

// TimePtr parses a time string and returns a pointer to time.Time.
// It supports multiple time formats commonly used in tests.
// Returns nil if parsing fails.
func TimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}

	layouts := []string{
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}

	return nil
}

// TimeMust parses a time string and returns a pointer to time.Time.
// Panics if parsing fails - use only in tests where the time string is known to be valid.
func TimeMust(s string) *time.Time {
	t := TimePtr(s)
	if t == nil {
		panic("failed to parse time: " + s)
	}
	return t
}
