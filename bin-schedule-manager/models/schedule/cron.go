package schedule

import (
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

// cronParser parses standard 5-field cron expressions (minute hour dom month dow).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ValidateCron validates the given 5-field cron expression.
// It rejects unparseable specs and specs that never match any time
// (e.g. "0 0 30 2 *" — February 30th).
func ValidateCron(spec string) error {
	s, err := cronParser.Parse(spec)
	if err != nil {
		return errors.Wrapf(err, "could not parse the cron spec. spec: %s", spec)
	}

	if s.Next(time.Now().UTC()).IsZero() {
		return errors.Errorf("the cron spec never matches. spec: %s", spec)
	}

	return nil
}

// NextRun returns the next run time of the given cron expression after the given time.
// The computation is always done in UTC.
func NextRun(spec string, from time.Time) (time.Time, error) {
	s, err := cronParser.Parse(spec)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "could not parse the cron spec. spec: %s", spec)
	}

	res := s.Next(from.UTC())
	if res.IsZero() {
		return time.Time{}, errors.Errorf("the cron spec never matches. spec: %s", spec)
	}

	return res, nil
}
