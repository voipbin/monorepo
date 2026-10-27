package recordinghandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_recordingReferenceTypeCall_cov2_errors(t *testing.T) {
	callID := uuid.FromStringOrNil("852def0e-f24a-11ed-845f-e32a849e7338")
	customerID := uuid.FromStringOrNil("5e1ab9a6-f24a-11ed-8c5b-3b1f0ab46f8d")
	callActiveflowID := uuid.FromStringOrNil("a1b75e8c-0727-11f0-9222-77e37a9b9643")

	tests := []struct {
		name string

		callGetErr    error
		channelGetErr error
		callStatus    call.Status
		balanceErr    error
		balanceValid  bool
		snoopErr      error
		dbCreateErr   error

		expectChannelGet bool
		expectBalance    bool
		expectSnoopCount int
		expectDBCreate   bool
		expectErr        string
	}{
		{name: "call get error", callGetErr: errors.New("x"), expectErr: "could not get call info: x"},
		{name: "channel get error", channelGetErr: errors.New("x"), expectChannelGet: true, expectErr: "could not get channel info: x"},
		{name: "call not progressing", callStatus: call.StatusDialing, expectChannelGet: true, expectErr: "invalid status. call_id: " + callID.String() + ", status: dialing"},
		{name: "balance check error", callStatus: call.StatusProgressing, balanceErr: errors.New("x"), expectChannelGet: true, expectBalance: true, expectErr: "could not check balance for recording: x"},
		{name: "insufficient balance", callStatus: call.StatusProgressing, expectChannelGet: true, expectBalance: true, expectErr: "insufficient balance for recording. customer_id: " + customerID.String()},
		{name: "snoop error", callStatus: call.StatusProgressing, balanceValid: true, snoopErr: errors.New("x"), expectChannelGet: true, expectBalance: true, expectSnoopCount: 1, expectErr: "could not create a snoop channel for recording: x"},
		{name: "db create error", callStatus: call.StatusProgressing, balanceValid: true, dbCreateErr: errors.New("x"), expectChannelGet: true, expectBalance: true, expectSnoopCount: 2, expectDBCreate: true, expectErr: "could not create the record"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &recordingHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, channelHandler: mockChannel}
			ctx := context.Background()

			c := &call.Call{
				Identity:     commonidentity.Identity{ID: callID, CustomerID: customerID},
				ChannelID:    "ch-1",
				ActiveflowID: callActiveflowID,
				Status:       tt.callStatus,
			}

			if tt.callGetErr != nil {
				mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(nil, tt.callGetErr)
			} else {
				mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(c, nil)
			}
			if tt.expectChannelGet {
				if tt.channelGetErr != nil {
					mockChannel.EXPECT().Get(ctx, "ch-1").Return(nil, tt.channelGetErr)
				} else {
					mockChannel.EXPECT().Get(ctx, "ch-1").Return(&channel.Channel{ID: "ch-1", AsteriskID: "ast-1"}, nil)
				}
			}
			if tt.expectBalance {
				mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeRecording, "", 1).Return(tt.balanceValid, tt.balanceErr)
			}
			if tt.expectSnoopCount > 0 {
				mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937"))
				mockUtil.EXPECT().TimeGetCurTimeRFC3339().Return("2020-04-18T03:22:17.995000Z")
				for i := 0; i < tt.expectSnoopCount; i++ {
					mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4()))
					if tt.snoopErr != nil {
						mockChannel.EXPECT().StartSnoop(ctx, "ch-1", gomock.Any(), gomock.Any(), gomock.Any(), channel.SnoopDirectionNone).Return(nil, tt.snoopErr)
					} else {
						mockChannel.EXPECT().StartSnoop(ctx, "ch-1", gomock.Any(), gomock.Any(), gomock.Any(), channel.SnoopDirectionNone).Return(&channel.Channel{ID: "snoop"}, nil)
					}
				}
			}
			if tt.expectDBCreate {
				mockDB.EXPECT().RecordingCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, r *recording.Recording) error {
					if r.ActiveflowID != callActiveflowID {
						t.Errorf("Wrong activeflow id. expect: %s, got: %s", callActiveflowID, r.ActiveflowID)
					}
					return tt.dbCreateErr
				})
			}

			// nil activeflow id must fall back to the call's activeflow id.
			res, err := h.recordingReferenceTypeCall(ctx, uuid.Nil, callID, recording.FormatWAV, 0, "", 0, uuid.Nil)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_recordingReferenceTypeConfbridge_cov2_errors(t *testing.T) {
	cbID := uuid.FromStringOrNil("2d6bb9e4-f24b-11ed-93c4-a3f4f0d0b2a1")
	customerID := uuid.FromStringOrNil("5e1ab9a6-f24a-11ed-8c5b-3b1f0ab46f8d")
	cbActiveflowID := uuid.FromStringOrNil("a1b75e8c-0727-11f0-9222-77e37a9b9643")
	deleted := testTimeNow()

	tests := []struct {
		name string

		cbGetErr     error
		cbDeleted    bool
		balanceErr   error
		balanceValid bool
		bridgeErr    error
		dbCreateErr  error
		recordErr    error

		expectBalance  bool
		expectBridge   bool
		expectDBCreate bool
		expectRecord   bool
		expectErr      string
	}{
		{name: "confbridge get error", cbGetErr: errors.New("x"), expectErr: "could not get confbridge info: x"},
		{name: "confbridge deleted", cbDeleted: true, expectErr: "invalid confbridge. confbridge_id: " + cbID.String()},
		{name: "balance error", balanceErr: errors.New("x"), expectBalance: true, expectErr: "could not check balance for recording: x"},
		{name: "insufficient balance", expectBalance: true, expectErr: "insufficient balance for recording. customer_id: " + customerID.String()},
		{name: "bridge get error", balanceValid: true, bridgeErr: errors.New("x"), expectBalance: true, expectBridge: true, expectErr: "could not get bridge info: x"},
		{name: "db create error", balanceValid: true, dbCreateErr: errors.New("x"), expectBalance: true, expectBridge: true, expectDBCreate: true, expectErr: "could not create the record"},
		{name: "record request error", balanceValid: true, recordErr: errors.New("x"), expectBalance: true, expectBridge: true, expectDBCreate: true, expectRecord: true, expectErr: "could not send the recording request: x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			h := &recordingHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, bridgeHandler: mockBridge}
			ctx := context.Background()

			cb := &confbridge.Confbridge{
				Identity:     commonidentity.Identity{ID: cbID, CustomerID: customerID},
				ActiveflowID: cbActiveflowID,
				BridgeID:     "br-1",
			}
			if tt.cbDeleted {
				cb.TMDelete = deleted
			}

			if tt.cbGetErr != nil {
				mockReq.EXPECT().CallV1ConfbridgeGet(ctx, cbID).Return(nil, tt.cbGetErr)
			} else {
				mockReq.EXPECT().CallV1ConfbridgeGet(ctx, cbID).Return(cb, nil)
			}
			if tt.expectBalance {
				mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeRecording, "", 1).Return(tt.balanceValid, tt.balanceErr)
			}
			if tt.expectBridge {
				if tt.bridgeErr != nil {
					mockBridge.EXPECT().Get(ctx, "br-1").Return(nil, tt.bridgeErr)
				} else {
					mockBridge.EXPECT().Get(ctx, "br-1").Return(&bridge.Bridge{ID: "br-1", AsteriskID: "ast-1"}, nil)
					mockUtil.EXPECT().TimeGetCurTimeRFC3339().Return("2020-04-18T03:22:17.995000Z")
					mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937"))
				}
			}
			if tt.expectDBCreate {
				mockDB.EXPECT().RecordingCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, r *recording.Recording) error {
					if r.ActiveflowID != cbActiveflowID || r.AsteriskID != "ast-1" || !strings.HasSuffix(r.Filenames[0], "_in.wav") {
						t.Errorf("Wrong recording. got: %v", r)
					}
					return tt.dbCreateErr
				})
				if tt.dbCreateErr == nil {
					mockDB.EXPECT().RecordingGet(ctx, uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937")).Return(&recording.Recording{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937")}}, nil)
				}
			}
			if tt.expectRecord {
				mockReq.EXPECT().AstBridgeRecord(ctx, "ast-1", "br-1", gomock.Any(), "wav", 0, 0, false, "", "fail").Return(tt.recordErr)
			}

			res, err := h.recordingReferenceTypeConfbridge(ctx, uuid.Nil, cbID, recording.FormatWAV, 0, "", 0, uuid.Nil)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func testTimeNow() *time.Time {
	t := time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)
	return &t
}
