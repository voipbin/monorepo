package campaigncallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
)

func Test_Done(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		result campaigncall.Result

		responseCampaigncall *campaigncall.Campaigncall
		expectStatus         omoutdialtarget.Status

		expectRes *campaigncall.Campaigncall
	}{
		{
			"result success",

			uuid.FromStringOrNil("f8eb1445-d31d-4adb-909f-e0284792ef8d"),
			campaigncall.ResultSuccess,

			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("f8eb1445-d31d-4adb-909f-e0284792ef8d"),
			},
			omoutdialtarget.StatusDone,

			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("f8eb1445-d31d-4adb-909f-e0284792ef8d"),
			},
		},
		{
			"result fail",

			uuid.FromStringOrNil("7093a2e7-9e1a-4955-90cf-2b5d7b544a64"),
			campaigncall.ResultFail,

			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("7093a2e7-9e1a-4955-90cf-2b5d7b544a64"),
			},
			omoutdialtarget.StatusIdle,

			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("7093a2e7-9e1a-4955-90cf-2b5d7b544a64"),
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

			mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, tt.id, campaigncall.StatusDone, tt.result).Return(nil)
			mockDB.EXPECT().CampaigncallGet(ctx, tt.id).Return(tt.responseCampaigncall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaigncall.CustomerID, campaigncall.EventTypeCampaigncallUpdated, tt.responseCampaigncall)

			mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatus(ctx, tt.responseCampaigncall.OutdialTargetID, tt.expectStatus).Return(&omoutdialtarget.OutdialTarget{}, nil)

			res, err := h.Done(ctx, tt.id, tt.result)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Progressing(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCampaigncall *campaigncall.Campaigncall
		expectRes            *campaigncall.Campaigncall
	}{
		{
			"normal",

			uuid.FromStringOrNil("f8eb1445-d31d-4adb-909f-e0284792ef8d"),

			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("f8eb1445-d31d-4adb-909f-e0284792ef8d"),
			},
			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("f8eb1445-d31d-4adb-909f-e0284792ef8d"),
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

			mockDB.EXPECT().CampaigncallUpdateStatus(ctx, tt.id, campaigncall.StatusProgressing).Return(nil)
			mockDB.EXPECT().CampaigncallGet(ctx, tt.id).Return(tt.responseCampaigncall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaigncall.CustomerID, campaigncall.EventTypeCampaigncallUpdated, tt.responseCampaigncall)

			res, err := h.Progressing(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
