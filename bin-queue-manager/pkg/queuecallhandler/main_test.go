package queuecallhandler

import (
	"context"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"
)

func TestParseTime(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	tests := []struct {
		name string

		targetTime string

		expectRes time.Time
	}{
		{
			"normal",

			"2021-04-18T03:22:17.994000Z",

			time.Date(2021, time.April, 18, 3, 22, 17, 994000000, time.UTC),
		},
		{
			"longer",

			"2023-02-15T08:00:19.951052Z128",
			time.Date(2023, time.February, 15, 8, 0, 19, 951052000, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parseTime(tt.targetTime)
			if err != nil {
				t.Errorf("Wrong match. exepct: %v, got: %v", tt.expectRes, res)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	tests := []struct {
		name string

		startTime string
		endTime   string

		expectRes time.Duration
	}{
		{
			"normal",

			"2021-04-18T03:22:17.994000Z",
			"2021-04-18T03:52:17.994000Z",

			time.Minute * 30,
		},

		{
			"start is future",

			"2021-04-18T03:52:17.994000Z",
			"2021-04-18T03:22:17.994000Z",

			time.Minute * -30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			res := getDuration(ctx, tt.startTime, tt.endTime)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}
