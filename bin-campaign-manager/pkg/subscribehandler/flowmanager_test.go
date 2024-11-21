package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-common-handler/models/sock"
)

func Test_processEventFMActiveflowDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		callID uuid.UUID

		responseCampaigncall *campaigncall.Campaigncall
	}{
		{
			"normal",
			&sock.Event{
				Publisher: "flow-manager",
				Type:      "activeflow_deleted",
				DataType:  "application/json",
				Data:      []byte(`{"id":"1f2da650-c473-11ec-871d-fbc80a740724"}`),
			},

			uuid.FromStringOrNil("1f2da650-c473-11ec-871d-fbc80a740724"),

			&campaigncall.Campaigncall{
				ID:            uuid.FromStringOrNil("e6cca6a4-c46c-11ec-8175-3fd04df5a0dc"),
				CampaignID:    uuid.FromStringOrNil("f4f81330-c46c-11ec-845b-634ec638de76"),
				ReferenceType: campaigncall.ReferenceTypeFlow,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := subscribeHandler{
				campaignHandler:     mockCampaign,
				campaigncallHandler: mockCampaigncall,
			}

			mockCampaigncall.EXPECT().GetByActiveflowID(gomock.Any(), tt.callID).Return(tt.responseCampaigncall, nil)
			mockCampaigncall.EXPECT().EventHandleActiveflowDeleted(gomock.Any(), tt.responseCampaigncall).Return(tt.responseCampaigncall, nil)
			mockCampaign.EXPECT().EventHandleActiveflowDeleted(gomock.Any(), tt.responseCampaigncall.CampaignID).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
