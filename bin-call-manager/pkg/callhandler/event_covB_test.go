package callhandler

import (
	"context"
	"testing"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_EventFMActiveflowUpdated_covB(t *testing.T) {
	callID := uuid.FromStringOrNil(covBCallID)

	tests := []struct {
		name         string
		activeflow   *fmactiveflow.Activeflow
		expectLookup bool
		lookupErr    error
		expectErr    bool
	}{
		{
			name:       "not ended is ignored",
			activeflow: &fmactiveflow.Activeflow{Status: fmactiveflow.StatusRunning, ReferenceType: fmactiveflow.ReferenceTypeCall, ReferenceID: callID},
		},
		{
			name:       "ended but not call reference is ignored",
			activeflow: &fmactiveflow.Activeflow{Status: fmactiveflow.StatusEnded, ReferenceType: fmactiveflow.ReferenceTypeNone, ReferenceID: callID},
		},
		{
			name:         "ended call, hangup error",
			activeflow:   &fmactiveflow.Activeflow{Status: fmactiveflow.StatusEnded, ReferenceType: fmactiveflow.ReferenceTypeCall, ReferenceID: callID},
			expectLookup: true,
			lookupErr:    dbhandler.ErrNotFound,
			expectErr:    true,
		},
		{
			name:         "ended call, already terminating",
			activeflow:   &fmactiveflow.Activeflow{Status: fmactiveflow.StatusEnded, ReferenceType: fmactiveflow.ReferenceTypeCall, ReferenceID: callID},
			expectLookup: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			if tt.expectLookup {
				if tt.lookupErr != nil {
					m.db.EXPECT().CallGet(ctx, callID).Return(nil, tt.lookupErr)
				} else {
					m.db.EXPECT().CallGet(ctx, callID).Return(covBCall(covBCallID, call.StatusTerminating), nil)
				}
			}

			err := h.EventFMActiveflowUpdated(ctx, tt.activeflow)
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}
