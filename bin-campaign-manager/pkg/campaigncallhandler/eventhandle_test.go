package campaigncallhandler

import (
	"context"
	"reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func Test_EventHandleActiveflowDeleted(t *testing.T) {

	tests := []struct {
		name string

		campaigncall *campaigncall.Campaigncall
		response     *campaigncall.Campaigncall
	}{
		{
			"normal",

			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3fe521fa-1c8e-412d-a57f-24f9a7d255be"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3fe521fa-1c8e-412d-a57f-24f9a7d255be"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, tt.campaigncall.ID, campaigncall.StatusDone, campaigncall.ResultSuccess).Return(nil)
			mockDB.EXPECT().CampaigncallGet(ctx, tt.campaigncall.ID).Return(tt.response, nil)
			mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatus(ctx, tt.response.OutdialTargetID, omoutdialtarget.StatusDone).Return(&omoutdialtarget.OutdialTarget{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaigncall.EventTypeCampaigncallUpdated, tt.response)

			_, err := h.EventHandleActiveflowDeleted(ctx, tt.campaigncall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandleReferenceCallHungup(t *testing.T) {

	tests := []struct {
		name string

		call         *cmcall.Call
		campaigncall *campaigncall.Campaigncall
		response     *campaigncall.Campaigncall

		expectResult campaigncall.Result
		expectStatus omoutdialtarget.Status
	}{
		{
			"hangup reason normal",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonNormal,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultSuccess,
			omoutdialtarget.StatusDone,
		},
		{
			"hangup reason amd",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonAMD,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultFail,
			omoutdialtarget.StatusIdle,
		},
		{
			"hangup reason busy",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonBusy,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultFail,
			omoutdialtarget.StatusIdle,
		},
		{
			"hangup reason canceled",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonCanceled,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultFail,
			omoutdialtarget.StatusIdle,
		},
		{
			"hangup reason dialout",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonDialout,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultFail,
			omoutdialtarget.StatusIdle,
		},
		{
			"hangup reason failed",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonFailed,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultFail,
			omoutdialtarget.StatusIdle,
		},
		{
			"hangup reason noanswer",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonNoanswer,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultFail,
			omoutdialtarget.StatusIdle,
		},
		{
			"hangup reason timeout",

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: cmcall.HangupReasonTimeout,
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1"),
				},
			},

			campaigncall.ResultFail,
			omoutdialtarget.StatusIdle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, tt.campaigncall.ID, campaigncall.StatusDone, tt.expectResult).Return(nil)
			mockDB.EXPECT().CampaigncallGet(ctx, tt.campaigncall.ID).Return(tt.response, nil)
			mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatus(ctx, tt.response.OutdialTargetID, tt.expectStatus).Return(&omoutdialtarget.OutdialTarget{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaigncall.EventTypeCampaigncallUpdated, tt.response)

			_, err := h.EventHandleReferenceCallHungup(ctx, tt.call, tt.campaigncall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandleReferenceCallHungup_alreadyDone(t *testing.T) {

	campaigncallID := uuid.FromStringOrNil("bdbed625-6203-4ab5-9c1f-4854089552e1")

	tests := []struct {
		name string

		storedStatus campaigncall.Status
		storedResult campaigncall.Result
		hangupReason cmcall.HangupReason

		// when expectDone is true, Done runs with expectResult and the outdial target gets expectTargetStatus.
		expectDone         bool
		expectResult       campaigncall.Result
		expectTargetStatus omoutdialtarget.Status
		expectErr          bool
	}{
		{
			name:               "stored dialing, normal hangup completes the campaigncall",
			storedStatus:       campaigncall.StatusDialing,
			hangupReason:       cmcall.HangupReasonNormal,
			expectDone:         true,
			expectResult:       campaigncall.ResultSuccess,
			expectTargetStatus: omoutdialtarget.StatusDone,
		},
		{
			name:               "stored progressing, normal hangup completes the campaigncall",
			storedStatus:       campaigncall.StatusProgressing,
			hangupReason:       cmcall.HangupReasonNormal,
			expectDone:         true,
			expectResult:       campaigncall.ResultSuccess,
			expectTargetStatus: omoutdialtarget.StatusDone,
		},
		{
			name:               "stored dialing, failed hangup completes the campaigncall",
			storedStatus:       campaigncall.StatusDialing,
			hangupReason:       cmcall.HangupReasonFailed,
			expectDone:         true,
			expectResult:       campaigncall.ResultFail,
			expectTargetStatus: omoutdialtarget.StatusIdle,
		},
		{
			name:         "done with fail, failed hangup is skipped",
			storedStatus: campaigncall.StatusDone,
			storedResult: campaigncall.ResultFail,
			hangupReason: cmcall.HangupReasonFailed,
		},
		{
			name:         "done with fail, busy hangup is skipped",
			storedStatus: campaigncall.StatusDone,
			storedResult: campaigncall.ResultFail,
			hangupReason: cmcall.HangupReasonBusy,
		},
		{
			name:         "done with success, normal hangup is skipped",
			storedStatus: campaigncall.StatusDone,
			storedResult: campaigncall.ResultSuccess,
			hangupReason: cmcall.HangupReasonNormal,
		},
		{
			name:         "done with success, failed hangup is skipped",
			storedStatus: campaigncall.StatusDone,
			storedResult: campaigncall.ResultSuccess,
			hangupReason: cmcall.HangupReasonFailed,
		},
		{
			name:               "done with fail, late normal hangup corrects the result",
			storedStatus:       campaigncall.StatusDone,
			storedResult:       campaigncall.ResultFail,
			hangupReason:       cmcall.HangupReasonNormal,
			expectDone:         true,
			expectResult:       campaigncall.ResultSuccess,
			expectTargetStatus: omoutdialtarget.StatusDone,
		},
		{
			name:         "done without a result, normal hangup is skipped",
			storedStatus: campaigncall.StatusDone,
			hangupReason: cmcall.HangupReasonNormal,
		},
		{
			name:         "done without a result, failed hangup is skipped",
			storedStatus: campaigncall.StatusDone,
			hangupReason: cmcall.HangupReasonFailed,
		},
		{
			name:         "done with fail, unknown hangup reason still returns the mapping error",
			storedStatus: campaigncall.StatusDone,
			storedResult: campaigncall.ResultFail,
			hangupReason: cmcall.HangupReason("unknown"),
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaigncallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			stored := &campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: campaigncallID,
				},
				Status: tt.storedStatus,
				Result: tt.storedResult,
			}
			call := &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6b87eb3-f79f-4b3d-a970-7ac4bc39fa31"),
				},
				HangupReason: tt.hangupReason,
			}

			// the gomock expectations are strict: a skipped campaigncall must not reach the database, the webhook or the outdial request.
			var expectRes *campaigncall.Campaigncall
			if tt.expectDone {
				expectRes = &campaigncall.Campaigncall{
					Identity: commonidentity.Identity{
						ID: campaigncallID,
					},
					Status: campaigncall.StatusDone,
					Result: tt.expectResult,
				}
				mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, campaigncallID, campaigncall.StatusDone, tt.expectResult).Return(nil)
				mockDB.EXPECT().CampaigncallGet(ctx, campaigncallID).Return(expectRes, nil)
				mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatus(ctx, expectRes.OutdialTargetID, tt.expectTargetStatus).Return(&omoutdialtarget.OutdialTarget{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, expectRes.CustomerID, campaigncall.EventTypeCampaigncallUpdated, expectRes)
			}

			res, err := h.EventHandleReferenceCallHungup(ctx, call, stored)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			if tt.expectDone {
				if !reflect.DeepEqual(res, expectRes) {
					t.Errorf("Wrong match. expect: %v, got: %v", expectRes, res)
				}
				return
			}

			// a skipped campaigncall returns the stored one.
			if res != stored {
				t.Errorf("Wrong match. expect: the stored campaigncall, got: %v", res)
			}
		})
	}
}
