package queuecallhandler

import (
	"context"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestGetDuration(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	tests := []struct {
		name string

		startTime *time.Time
		endTime   *time.Time

		expectRes time.Duration
	}{
		{
			"normal",

			timePtr(time.Date(2021, time.April, 18, 3, 22, 17, 994000000, time.UTC)),
			timePtr(time.Date(2021, time.April, 18, 3, 52, 17, 994000000, time.UTC)),

			time.Minute * 30,
		},

		{
			"start is future",

			timePtr(time.Date(2021, time.April, 18, 3, 52, 17, 994000000, time.UTC)),
			timePtr(time.Date(2021, time.April, 18, 3, 22, 17, 994000000, time.UTC)),

			time.Minute * -30,
		},

		{
			"nil start",

			nil,
			timePtr(time.Date(2021, time.April, 18, 3, 22, 17, 994000000, time.UTC)),

			0,
		},

		{
			"nil end",

			timePtr(time.Date(2021, time.April, 18, 3, 22, 17, 994000000, time.UTC)),
			nil,

			0,
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
