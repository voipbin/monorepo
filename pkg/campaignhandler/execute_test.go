package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/campaigncallhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/outplanhandler"
)

func Test_ExecuteWithTypeFlow(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response                *campaign.Campaign
		responseOutplan         *outplan.Outplan
		responseOmoutdialtarget []omoutdialtarget.OutdialTarget
		responseCampaigncall    *campaigncall.Campaigncall
		responseActiveflow      *activeflow.Activeflow

		expectDestination      *cmaddress.Address
		expectDestinationIndex int
		expectTryCount         int
	}{
		{
			"normal",

			uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),

			&campaign.Campaign{
				ID:        uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),
				OutdialID: uuid.FromStringOrNil("bcad478e-c3fe-11ec-be28-a73254d6e0fc"),
				OutplanID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				QueueID:   uuid.Nil,
				FlowID:    uuid.FromStringOrNil("9574a6d6-c402-11ec-829f-33bd6e27d95f"),
				Status:    campaign.StatusRun,
				Type:      campaign.TypeFlow,
			},
			&outplan.Outplan{
				ID:           uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				MaxTryCount0: 4,
			},
			[]omoutdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
					Destination0: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					TryCount0: 0,
				},
			},
			&campaigncall.Campaigncall{
				ID:           uuid.FromStringOrNil("400524dc-c402-11ec-9e8f-2fefadc4fc39"),
				ActiveflowID: uuid.FromStringOrNil("6bab615a-c402-11ec-931f-df3080b6bcef"),
			},
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("6bab615a-c402-11ec-931f-df3080b6bcef"),
			},

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			0,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
				outplanHandler:      mockOutplan,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockOutplan.EXPECT().Get(ctx, tt.response.OutplanID).Return(tt.responseOutplan, nil)

			// get destination
			mockReq.EXPECT().OMV1OutdialtargetGetsAvailable(
				ctx,
				tt.response.OutdialID,
				tt.responseOutplan.MaxTryCount0,
				tt.responseOutplan.MaxTryCount1,
				tt.responseOutplan.MaxTryCount2,
				tt.responseOutplan.MaxTryCount3,
				tt.responseOutplan.MaxTryCount4,
				tt.responseOutplan.TryInterval,
				1,
			).Return(tt.responseOmoutdialtarget, nil)

			// executeFlow
			mockCampaigncall.EXPECT().Create(
				ctx,
				tt.response.CustomerID,
				tt.response.ID,
				tt.response.OutplanID,
				tt.response.OutdialID,
				tt.responseOmoutdialtarget[0].ID,
				tt.response.QueueID,

				gomock.Any(),
				tt.response.FlowID,

				campaigncall.ReferenceTypeFlow,
				uuid.Nil,

				tt.responseOutplan.Source,
				tt.expectDestination,
				tt.expectDestinationIndex,
				tt.expectTryCount,
			).Return(tt.responseCampaigncall, nil)
			mockCampaigncall.EXPECT().Progressing(ctx, tt.responseCampaigncall.ID).Return(tt.responseCampaigncall, nil)
			mockReq.EXPECT().FMV1ActiveflowCreate(ctx, tt.responseCampaigncall.ActiveflowID, tt.responseCampaigncall.FlowID, activeflow.ReferenceTypeNone, uuid.Nil).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FMV1ActiveflowExecute(ctx, tt.responseActiveflow.ID).Return(nil)

			mockReq.EXPECT().CAV1CampaignExecute(ctx, tt.id, 500).Return(nil)

			h.Execute(ctx, tt.id)
		})
	}
}

func Test_getDestination(t *testing.T) {

	tests := []struct {
		name string

		c *campaign.Campaign
		p *outplan.Outplan

		responseOutdialtarget []omoutdialtarget.OutdialTarget

		expectResOutdialTarget    *omoutdialtarget.OutdialTarget
		expectResDestination      *cmaddress.Address
		expectResDestinationIndex int
		expectResTryCount         int
	}{
		{
			"normal",

			&campaign.Campaign{
				ID:        uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),
				OutdialID: uuid.FromStringOrNil("bcad478e-c3fe-11ec-be28-a73254d6e0fc"),
				OutplanID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				QueueID:   uuid.Nil,
				FlowID:    uuid.FromStringOrNil("9574a6d6-c402-11ec-829f-33bd6e27d95f"),
				Status:    campaign.StatusRun,
				Type:      campaign.TypeFlow,
			},
			&outplan.Outplan{
				ID:           uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				MaxTryCount0: 4,
			},

			[]omoutdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
					Destination0: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					TryCount0: 0,
				},
			},

			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
				Destination0: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 0,
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			0,
			1,
		},
		{
			"trycount 2",

			&campaign.Campaign{
				ID:        uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),
				OutdialID: uuid.FromStringOrNil("bcad478e-c3fe-11ec-be28-a73254d6e0fc"),
				OutplanID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				QueueID:   uuid.Nil,
				FlowID:    uuid.FromStringOrNil("9574a6d6-c402-11ec-829f-33bd6e27d95f"),
				Status:    campaign.StatusRun,
				Type:      campaign.TypeFlow,
			},
			&outplan.Outplan{
				ID:           uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				MaxTryCount0: 4,
			},

			[]omoutdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
					Destination0: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					TryCount0: 1,
				},
			},

			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
				Destination0: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 1,
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			0,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
				outplanHandler:      mockOutplan,
			}

			ctx := context.Background()

			mockReq.EXPECT().OMV1OutdialtargetGetsAvailable(
				ctx,
				tt.c.OutdialID,
				tt.p.MaxTryCount0,
				tt.p.MaxTryCount1,
				tt.p.MaxTryCount2,
				tt.p.MaxTryCount3,
				tt.p.MaxTryCount4,
				tt.p.TryInterval,
				1,
			).Return(tt.responseOutdialtarget, nil)

			resTarget, resDestination, resDestinationIndex, resTrycount, err := h.getDestination(ctx, tt.c, tt.p)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(resTarget, tt.expectResOutdialTarget) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResOutdialTarget, resTarget)
			}
			if reflect.DeepEqual(resDestination, tt.expectResDestination) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResDestination, resDestination)
			}
			if reflect.DeepEqual(resDestinationIndex, tt.expectResDestinationIndex) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResDestinationIndex, resDestinationIndex)
			}
			if reflect.DeepEqual(resTrycount, tt.expectResTryCount) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResTryCount, resTrycount)
			}

		})
	}
}

func Test_isDialable(t *testing.T) {

	tests := []struct {
		name string

		campaignID   uuid.UUID
		queueID      uuid.UUID
		serviceLevel int

		responseAgents        []amagent.Agent
		responseCampaingcalls []*campaigncall.Campaigncall

		expectRes bool
	}{
		{
			"normal",

			uuid.FromStringOrNil("54bbbdce-c406-11ec-b272-0b23f3d17d40"),
			uuid.FromStringOrNil("54f6574a-c406-11ec-b9a9-3b91cde12b94"),
			100,

			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("c1ec0278-c406-11ec-8804-c738ff121aa6"),
				},
			},
			[]*campaigncall.Campaigncall{},

			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
				outplanHandler:      mockOutplan,
			}

			ctx := context.Background()

			mockReq.EXPECT().QMV1QueueGetAgents(ctx, tt.queueID, amagent.StatusAvailable).Return(tt.responseAgents, nil)
			mockCampaigncall.EXPECT().GetsByCampaignIDAndStatus(ctx, tt.campaignID, campaigncall.StatusDialing, gomock.Any(), uint64(100)).Return(tt.responseCampaingcalls, nil)

			res := h.isDialable(ctx, tt.campaignID, tt.queueID, tt.serviceLevel)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
