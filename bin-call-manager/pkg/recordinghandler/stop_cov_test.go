package recordinghandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-flow-manager/models/activeflow"
	smfile "monorepo/bin-storage-manager/models/file"
)

func Test_stopped_cov(t *testing.T) {
	recID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	custID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")
	flowID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000003")
	afID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000004")
	newAFID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000005")

	newRec := func(flow uuid.UUID) *recording.Recording {
		return &recording.Recording{
			Identity:     commonidentity.Identity{ID: recID, CustomerID: custID},
			AsteriskID:   "ast-1",
			Filenames:    []string{"a_in.wav"},
			OnEndFlowID:  flow,
			ActiveflowID: afID,
		}
	}

	tests := []struct {
		name string
		rec  *recording.Recording

		moveErr   error
		fileErr   error
		createErr error
		varErr    error
		execErr   error

		expectCreate bool
		expectVar    bool
		expectExec   bool
		expectErr    bool
	}{
		{name: "no on end flow", rec: newRec(uuid.Nil)},
		{name: "move failure", rec: newRec(uuid.Nil), moveErr: errors.New("move"), expectErr: true},
		{name: "file create failure", rec: newRec(uuid.Nil), fileErr: errors.New("file"), expectErr: true},
		{name: "activeflow create failure", rec: newRec(flowID), expectCreate: true, createErr: errors.New("create"), expectErr: true},
		{name: "variable set failure is ignored", rec: newRec(flowID), expectCreate: true, expectVar: true, varErr: errors.New("var"), expectExec: true},
		{name: "execute failure", rec: newRec(flowID), expectCreate: true, expectVar: true, expectExec: true, execErr: errors.New("exec"), expectErr: true},
		{name: "normal with flow", rec: newRec(flowID), expectCreate: true, expectVar: true, expectExec: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &recordingHandler{reqHandler: mockReq}
			ctx := context.Background()

			mockReq.EXPECT().AstProxyRecordingFileMove(ctx, tt.rec.AsteriskID, tt.rec.Filenames, 30000).Return(tt.moveErr)
			if tt.moveErr == nil {
				mockReq.EXPECT().StorageV1FileCreate(ctx, custID, uuid.Nil, smfile.ReferenceTypeRecording, recID, smfile.TypeRecording, gomock.Any(), gomock.Any(), "a_in.wav", defaultBucketName, gomock.Any(), 30000).Return(&smfile.File{}, tt.fileErr)
			}
			if tt.expectCreate {
				mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, custID, flowID, activeflow.ReferenceTypeNone, uuid.Nil, afID, nil, "", activeflow.WebhookMethodNone).
					Return(func() *activeflow.Activeflow {
						if tt.createErr != nil {
							return nil
						}
						return &activeflow.Activeflow{Identity: commonidentity.Identity{ID: newAFID}}
					}(), tt.createErr)
			}
			if tt.expectVar {
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, newAFID, gomock.Any()).Return(tt.varErr)
			}
			if tt.expectExec {
				mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, newAFID).Return(tt.execErr)
			}

			err := h.stopped(ctx, tt.rec)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_Stop_cov(t *testing.T) {
	recID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		getRec *recording.Recording
		getErr error

		stopChannelIDs bool // expect HangingUp for call type
		astStopErr     error
		setStatusErr   error
		finalGetErr    error

		expectStatusSet bool
		expectFinalGet  bool
		expectErr       bool
	}{
		{
			name:   "get failure",
			getErr: errors.New("get"), expectErr: true,
		},
		{
			name:   "unsupported reference type",
			getRec: &recording.Recording{Identity: commonidentity.Identity{ID: recID}, ReferenceType: recording.ReferenceType("unknown")}, expectErr: true,
		},
		{
			name:   "call type normal",
			getRec: &recording.Recording{Identity: commonidentity.Identity{ID: recID}, ReferenceType: recording.ReferenceTypeCall, AsteriskID: "ast-1", ChannelIDs: []string{"c1"}},

			stopChannelIDs: true, expectStatusSet: true, expectFinalGet: true,
		},
		{
			name:   "confbridge stop failure",
			getRec: &recording.Recording{Identity: commonidentity.Identity{ID: recID}, ReferenceType: recording.ReferenceTypeConfbridge, AsteriskID: "ast-1", RecordingName: "rn"},

			astStopErr: errors.New("stop"), expectErr: true,
		},
		{
			name:   "set status failure",
			getRec: &recording.Recording{Identity: commonidentity.Identity{ID: recID}, ReferenceType: recording.ReferenceTypeConfbridge, AsteriskID: "ast-1", RecordingName: "rn"},

			setStatusErr: errors.New("status"), expectStatusSet: true, expectErr: true,
		},
		{
			name:   "final get failure",
			getRec: &recording.Recording{Identity: commonidentity.Identity{ID: recID}, ReferenceType: recording.ReferenceTypeConfbridge, AsteriskID: "ast-1", RecordingName: "rn"},

			finalGetErr: errors.New("final"), expectStatusSet: true, expectFinalGet: true, expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &recordingHandler{reqHandler: mockReq, db: mockDB, channelHandler: mockChannel}
			ctx := context.Background()

			// Get() (first call) + optional second RecordingGet for final lookup
			mockDB.EXPECT().RecordingGet(ctx, recID).Return(tt.getRec, tt.getErr)

			if tt.getRec != nil {
				switch tt.getRec.ReferenceType {
				case recording.ReferenceTypeCall:
					mockChannel.EXPECT().HangingUpWithAsteriskID(ctx, tt.getRec.AsteriskID, "c1", gomock.Any()).Return(nil)
				case recording.ReferenceTypeConfbridge:
					mockReq.EXPECT().AstRecordingStop(ctx, tt.getRec.AsteriskID, "rn_in").Return(tt.astStopErr)
				}
			}
			if tt.expectStatusSet {
				mockDB.EXPECT().RecordingSetStatus(ctx, recID, recording.StatusStopping).Return(tt.setStatusErr)
			}
			if tt.expectFinalGet {
				mockDB.EXPECT().RecordingGet(ctx, recID).Return(tt.getRec, tt.finalGetErr)
			}

			res, err := h.Stop(ctx, recID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res == nil {
				t.Errorf("Wrong match. expect: recording, got: nil")
			}
		})
	}
}

func Test_Stopped_cov(t *testing.T) {
	recID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001")
	custID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000002")

	tests := []struct {
		name string

		statusErr  error
		getErr     error
		moveErr    error
		expectGet  bool
		expectMove bool
		expectErr  bool
	}{
		{name: "normal", expectGet: true, expectMove: true},
		{name: "set status failure", statusErr: errors.New("status"), expectErr: true},
		{name: "get failure", getErr: errors.New("get"), expectGet: true, expectErr: true},
		{name: "stopped failure", moveErr: errors.New("move"), expectGet: true, expectMove: true, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &recordingHandler{reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			rec := &recording.Recording{
				Identity:      commonidentity.Identity{ID: recID, CustomerID: custID},
				ReferenceType: recording.ReferenceTypeCall,
				AsteriskID:    "ast-1",
			}

			mockDB.EXPECT().RecordingSetStatus(ctx, recID, recording.StatusEnded).Return(tt.statusErr)
			if tt.expectGet {
				var r *recording.Recording
				if tt.getErr == nil {
					r = rec
				}
				mockDB.EXPECT().RecordingGet(ctx, recID).Return(r, tt.getErr)
			}
			if tt.expectGet && tt.getErr == nil {
				mockNotify.EXPECT().PublishWebhookEvent(ctx, custID, recording.EventTypeRecordingFinished, rec)
			}
			if tt.expectMove {
				mockReq.EXPECT().AstProxyRecordingFileMove(ctx, "ast-1", gomock.Any(), 30000).Return(tt.moveErr)
			}

			res, err := h.Stopped(ctx, recID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res != rec {
				t.Errorf("Wrong match. expect: %v, got: %v", rec, res)
			}
		})
	}
}
