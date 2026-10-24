package transcribehandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

func Test_Stop_onEndFlow(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000001")
	customerID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000002")
	onEndFlowID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000003")
	activeflowID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000004")
	newActiveflowID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000005")

	newTranscribe := func() *transcribe.Transcribe {
		return &transcribe.Transcribe{
			Identity:      commonidentity.Identity{ID: id, CustomerID: customerID},
			ActiveflowID:  activeflowID,
			OnEndFlowID:   onEndFlowID,
			ReferenceType: transcribe.ReferenceTypeCall,
			Status:        transcribe.StatusProgressing,
			Language:      "en-US",
			Direction:     transcribe.DirectionBoth,
		}
	}

	tests := []struct {
		name string

		createErr   error
		variableErr error
		executeErr  error

		expectVariable bool
		expectExecute  bool
	}{
		{
			name:           "on end flow is created, variables set and executed",
			expectVariable: true,
			expectExecute:  true,
		},
		{
			name:      "activeflow creation failure is swallowed and nothing else runs",
			createErr: fmt.Errorf("flow unavailable"),
		},
		{
			name:           "variable failure is swallowed and the flow is still executed",
			variableErr:    fmt.Errorf("variable set failed"),
			expectVariable: true,
			expectExecute:  true,
		},
		{
			name:           "execute failure is swallowed",
			executeErr:     fmt.Errorf("execute failed"),
			expectVariable: true,
			expectExecute:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transcribeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			// no streamings: stopLive only marks the transcribe done
			done := newTranscribe()
			done.Status = transcribe.StatusDone
			gomock.InOrder(
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(newTranscribe(), nil),
				mockDB.EXPECT().TranscribeUpdate(ctx, id, map[transcribe.Field]any{transcribe.FieldStatus: transcribe.StatusDone}).Return(nil),
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(done, nil),
			)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, transcribe.EventTypeTranscribeDone, done)

			var createRes *fmactiveflow.Activeflow
			if tt.createErr == nil {
				createRes = &fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: newActiveflowID}}
			}
			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, onEndFlowID, fmactiveflow.ReferenceTypeTranscribe, id, activeflowID, nil, "", fmactiveflow.WebhookMethodNone).Return(createRes, tt.createErr)

			if tt.expectVariable {
				expectVariables := map[string]string{
					variableTranscribeID:        id.String(),
					variableTranscribeLanguage:  "en-US",
					variableTranscribeDirection: string(transcribe.DirectionBoth),
				}
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, newActiveflowID, expectVariables).Return(tt.variableErr)
			}
			if tt.expectExecute {
				mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, newActiveflowID).Return(tt.executeErr)
			}

			res, err := h.Stop(ctx, id)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != done {
				t.Errorf("Wrong match. expect the done transcribe, got: %v", res)
			}
		})
	}
}

func Test_Stop_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000011")

	tests := []struct {
		name string

		getErr    error
		updateErr error
		reGetErr  error
	}{
		{
			name:   "lookup failure",
			getErr: fmt.Errorf("select failed"),
		},
		{
			name:   "lookup of a missing transcribe",
			getErr: dbhandler.ErrNotFound,
		},
		{
			name:      "status update failure",
			updateErr: fmt.Errorf("update failed"),
		},
		{
			name:     "lookup failure after the status update",
			reGetErr: fmt.Errorf("select failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &transcribeHandler{db: mockDB}
			ctx := context.Background()

			tr := &transcribe.Transcribe{
				Identity:      commonidentity.Identity{ID: id},
				ReferenceType: transcribe.ReferenceTypeConfbridge,
				Status:        transcribe.StatusProgressing,
			}

			if tt.getErr != nil {
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, tt.getErr)
			} else {
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil)
				mockDB.EXPECT().TranscribeUpdate(ctx, id, map[transcribe.Field]any{transcribe.FieldStatus: transcribe.StatusDone}).Return(tt.updateErr)
				if tt.updateErr == nil {
					mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, tt.reGetErr)
				}
			}

			res, err := h.Stop(ctx, id)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}
