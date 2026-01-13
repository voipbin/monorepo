package subscribehandler

import (
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
)

func Test_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		callID uuid.UUID

		responseCampaigncall *campaigncall.Campaigncall
	}{
		{
			"normal",
			&sock.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"62a54c96-c46c-11ec-aff0-ebddfa5d9bc4"}`),
			},

			uuid.FromStringOrNil("62a54c96-c46c-11ec-aff0-ebddfa5d9bc4"),

			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e6cca6a4-c46c-11ec-8175-3fd04df5a0dc"),
				},
				CampaignID: uuid.FromStringOrNil("f4f81330-c46c-11ec-845b-634ec638de76"),
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

			mockCampaigncall.EXPECT().GetByReferenceID(gomock.Any(), tt.callID.Return(tt.responseCampaigncall, nil)
			mockCampaigncall.EXPECT().EventHandleReferenceCallHungup(gomock.Any(), gomock.Any(), tt.responseCampaigncall.Return(tt.responseCampaigncall, nil)
			mockCampaign.EXPECT().EventHandleReferenceCallHungup(gomock.Any(), tt.responseCampaigncall.CampaignID.Return(nil)

			h.processEvent(tt.event)
		})
	}
}
