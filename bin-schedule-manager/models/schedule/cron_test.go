package schedule

import (
	"testing"
	"time"
)

func TestValidateCron(t *testing.T) {
	tests := []struct {
		name string

		spec string

		wantErr bool
	}{
		{
			name: "valid_every_five_minutes",

			spec: "*/5 * * * *",

			wantErr: false,
		},
		{
			name: "valid_daily_at_two",

			spec: "0 2 * * *",

			wantErr: false,
		},
		{
			name: "valid_weekly_sunday_midnight",

			spec: "0 0 * * 0",

			wantErr: false,
		},
		{
			name: "invalid_empty_spec",

			spec: "",

			wantErr: true,
		},
		{
			name: "invalid_not_a_cron",

			spec: "invalid",

			wantErr: true,
		},
		{
			name: "invalid_four_fields",

			spec: "* * * *",

			wantErr: true,
		},
		{
			name: "invalid_minute_out_of_range",

			spec: "61 * * * *",

			wantErr: true,
		},
		{
			name: "invalid_never_matching_february_30th",

			spec: "0 0 30 2 *",

			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCron(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCron() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNextRun(t *testing.T) {
	tests := []struct {
		name string

		spec string
		from time.Time

		expectRes time.Time
		wantErr   bool
	}{
		{
			name: "daily_at_two_from_midnight_utc",

			spec: "0 2 * * *",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),

			expectRes: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name: "daily_at_two_after_slot_rolls_to_next_day",

			spec: "0 2 * * *",
			from: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),

			expectRes: time.Date(2026, 1, 2, 2, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name: "non_utc_location_is_converted_to_utc",

			spec: "0 2 * * *",
			from: time.Date(2026, 1, 1, 5, 0, 0, 0, time.FixedZone("KST", 9*60*60)), // 2025-12-31 20:00 UTC

			expectRes: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name: "every_fifteen_minutes",

			spec: "*/15 * * * *",
			from: time.Date(2026, 1, 1, 10, 7, 0, 0, time.UTC),

			expectRes: time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name: "invalid_spec_returns_error",

			spec: "invalid",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),

			wantErr: true,
		},
		{
			name: "never_matching_spec_returns_error",

			spec: "0 0 30 2 *",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),

			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NextRun(tt.spec, tt.from)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if !res.Equal(tt.expectRes) {
				t.Errorf("Wrong next run. expect: %s, got: %s", tt.expectRes, res)
			}
			if res.Location() != time.UTC {
				t.Errorf("Wrong location. expect: UTC, got: %s", res.Location())
			}
		})
	}
}
