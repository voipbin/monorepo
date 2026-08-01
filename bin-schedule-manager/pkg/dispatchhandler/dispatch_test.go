package dispatchhandler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

type sendResult struct {
	response *sock.Response
	err      error
}

func Test_dispatch(t *testing.T) {
	now := time.Date(2026, 8, 1, 1, 0, 5, 0, time.UTC)

	tests := []struct {
		name string

		retryMax    int
		sendResults []sendResult

		responseCompleted bool

		expectStatus     execution.Status
		expectStatusCode int
		expectErrStr     string
		expectResult     string
		expectAttempts   int
		expectEventType  string
		expectCompleted  bool
	}{
		{
			name: "success first attempt",

			retryMax: 0,
			sendResults: []sendResult{
				{response: &sock.Response{StatusCode: 200, Data: []byte(`{"ok":true}`)}},
			},

			responseCompleted: true,

			expectStatus:     execution.StatusSuccess,
			expectStatusCode: 200,
			expectErrStr:     "",
			expectResult:     `{"ok":true}`,
			expectAttempts:   1,
			expectEventType:  schedule.EventTypeExecutionSucceeded,
			expectCompleted:  true,
		},
		{
			name: "retry then success",

			retryMax: 2,
			sendResults: []sendResult{
				{err: stderrors.New("network down")},
				{response: &sock.Response{StatusCode: 200, Data: []byte(`{"ok":true}`)}},
			},

			responseCompleted: true,

			expectStatus:     execution.StatusSuccess,
			expectStatusCode: 200,
			expectErrStr:     "",
			expectResult:     `{"ok":true}`,
			expectAttempts:   2,
			expectEventType:  schedule.EventTypeExecutionSucceeded,
			expectCompleted:  true,
		},
		{
			name: "retries exhausted",

			retryMax: 1,
			sendResults: []sendResult{
				{response: &sock.Response{StatusCode: 500}},
				{response: &sock.Response{StatusCode: 500}},
			},

			responseCompleted: true,

			expectStatus:     execution.StatusFailed,
			expectStatusCode: 500,
			expectErrStr:     "dispatch failed with status code 500",
			expectResult:     "",
			expectAttempts:   2,
			expectEventType:  schedule.EventTypeExecutionFailed,
			expectCompleted:  true,
		},
		{
			name: "transport error",

			retryMax: 0,
			sendResults: []sendResult{
				{err: stderrors.New("network down")},
			},

			responseCompleted: true,

			expectStatus:     execution.StatusFailed,
			expectStatusCode: 0,
			expectErrStr:     "network down",
			expectResult:     "",
			expectAttempts:   1,
			expectEventType:  schedule.EventTypeExecutionFailed,
			expectCompleted:  true,
		},
		{
			name: "result truncated to 60000 bytes",

			retryMax: 0,
			sendResults: []sendResult{
				{response: &sock.Response{StatusCode: 200, Data: []byte(strings.Repeat("a", 70000))}},
			},

			responseCompleted: true,

			expectStatus:     execution.StatusSuccess,
			expectStatusCode: 200,
			expectErrStr:     "",
			expectResult:     strings.Repeat("a", 60000),
			expectAttempts:   1,
			expectEventType:  schedule.EventTypeExecutionSucceeded,
			expectCompleted:  true,
		},
		{
			name: "completion lost to reaper",

			retryMax: 0,
			sendResults: []sendResult{
				{response: &sock.Response{StatusCode: 200, Data: []byte(`{"ok":true}`)}},
			},

			responseCompleted: false,

			expectStatus:     execution.StatusSuccess,
			expectStatusCode: 200,
			expectErrStr:     "",
			expectResult:     `{"ok":true}`,
			expectAttempts:   1,
			expectCompleted:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &dispatchHandler{
				utilHandler:     mockUtil,
				reqHandler:      mockReq,
				db:              mockDB,
				notifyHandler:   mockNotify,
				retryBackoff:    time.Millisecond,
				lastOverlapSkip: map[uuid.UUID]time.Time{},
			}

			ctx := context.Background()

			s := &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5f607182-1a11-11ef-9be0-6f2a2b7f9d3e"),
				},
				Name: "test-dispatch",

				TargetQueue:    "bin-manager.number-manager.request",
				TargetURI:      "/v1/numbers/renew",
				TargetMethod:   "POST",
				TargetDataType: "application/json",
				TargetData:     []byte(`{"days":28}`),

				TimeoutMS: 30000,
				RetryMax:  tt.retryMax,
			}

			exec := &execution.Execution{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("60718293-1a11-11ef-a3c3-6f2a2b7f9d3e"),
				},
				ScheduleID: s.ID,
				Status:     execution.StatusRunning,
			}

			mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()

			for _, r := range tt.sendResults {
				mockReq.EXPECT().SendRequest(
					ctx,
					commonoutline.QueueName(s.TargetQueue),
					s.TargetURI,
					sock.RequestMethod(s.TargetMethod),
					s.TimeoutMS,
					0,
					s.TargetDataType,
					gomock.Any(),
				).Return(r.response, r.err)
			}

			mockDB.EXPECT().ExecutionComplete(ctx, exec.ID, tt.expectStatus, tt.expectStatusCode, tt.expectErrStr, tt.expectResult, tt.expectAttempts, 0, now).Return(tt.responseCompleted, nil)

			if tt.expectCompleted {
				mockDB.EXPECT().ExecutionGet(ctx, exec.ID).Return(exec, nil)
				mockNotify.EXPECT().PublishEvent(ctx, tt.expectEventType, exec)
			}

			before := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, string(tt.expectStatus)))

			h.dispatch(ctx, s, exec)

			after := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, string(tt.expectStatus)))

			expectDelta := float64(0)
			if tt.expectCompleted {
				expectDelta = 1
			}
			if after-before != expectDelta {
				t.Errorf("Wrong match. expect: %v, got: %v", expectDelta, after-before)
			}
		})
	}
}

func Test_truncateResult(t *testing.T) {
	tests := []struct {
		name string

		input  string
		expect string
	}{
		{
			name: "short passthrough",

			input:  "short",
			expect: "short",
		},
		{
			name: "exactly at bound",

			input:  strings.Repeat("b", resultMaxBytes),
			expect: strings.Repeat("b", resultMaxBytes),
		},
		{
			name: "over bound truncated",

			input:  strings.Repeat("c", resultMaxBytes+1),
			expect: strings.Repeat("c", resultMaxBytes),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := truncateResult(tt.input)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %d bytes, got: %d bytes", len(tt.expect), len(res))
			}
		})
	}
}
