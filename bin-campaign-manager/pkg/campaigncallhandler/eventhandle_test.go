package campaigncallhandler

import (
	"context"
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
