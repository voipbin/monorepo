package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-flow-manager/models/activeflow"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

func Test_ExecuteWithTypeFlow(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCampaign        *campaign.Campaign
		responseOutplan         *outplan.Outplan
		responseOmoutdialtarget []omoutdialtarget.OutdialTarget
		responseUUID            uuid.UUID
		responseCampaigncall    *campaigncall.Campaigncall
		responseActiveflow      *activeflow.Activeflow

		expectDestination      *commonaddress.Address
		expectDestinationIndex int
		expectTryCount         int
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),

			responseCampaign: &campaign.Campaign{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),
				},
				OutdialID: uuid.FromStringOrNil("bcad478e-c3fe-11ec-be28-a73254d6e0fc"),
				OutplanID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				QueueID:   uuid.Nil,
				FlowID:    uuid.FromStringOrNil("9574a6d6-c402-11ec-829f-33bd6e27d95f"),
				Status:    campaign.StatusRun,
				Type:      campaign.TypeFlow,
			},
			responseOutplan: &outplan.Outplan{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				},
				MaxTryCount0: 4,
			},
			responseOmoutdialtarget: []omoutdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
					Destination0: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					TryCount0: 0,
				},
			},
			responseUUID: uuid.FromStringOrNil("400524dc-c402-11ec-9e8f-2fefadc4fc39"),
			responseCampaigncall: &campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("400524dc-c402-11ec-9e8f-2fefadc4fc39"),
				},
				ActiveflowID: uuid.FromStringOrNil("6bab615a-c402-11ec-931f-df3080b6bcef"),
			},
			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6bab615a-c402-11ec-931f-df3080b6bcef"),
				},
			},

			expectDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDestinationIndex: 0,
			expectTryCount:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				util:                mockUtil,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
				outplanHandler:      mockOutplan,
			}
			ctx := context.Background()

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)
			mockOutplan.EXPECT().Get(ctx, tt.responseCampaign.OutplanID).Return(tt.responseOutplan, nil)

			// get destination
			mockReq.EXPECT().OutdialV1OutdialtargetGetsAvailable(
				ctx,
				tt.responseCampaign.OutdialID,
				tt.responseOutplan.MaxTryCount0,
				tt.responseOutplan.MaxTryCount1,
				tt.responseOutplan.MaxTryCount2,
				tt.responseOutplan.MaxTryCount3,
				tt.responseOutplan.MaxTryCount4,
				1,
			).Return(tt.responseOmoutdialtarget, nil)

			// executeFlow
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockCampaigncall.EXPECT().Create(
				ctx,
				tt.responseCampaign.CustomerID,
				tt.responseCampaign.ID,
				tt.responseCampaign.OutplanID,
				tt.responseCampaign.OutdialID,
				tt.responseOmoutdialtarget[0].ID,
				tt.responseCampaign.QueueID,

				gomock.Any(),
				tt.responseCampaign.FlowID,

				campaigncall.ReferenceTypeFlow,
				uuid.Nil,

				tt.responseOutplan.Source,
				tt.expectDestination,
				tt.expectDestinationIndex,
				tt.expectTryCount,
			).Return(tt.responseCampaigncall, nil)
			mockCampaigncall.EXPECT().Progressing(ctx, tt.responseCampaigncall.ID).Return(tt.responseCampaigncall, nil)
			mockReq.EXPECT().FlowV1ActiveflowCreate(
				ctx,
				tt.responseCampaigncall.ActiveflowID,
				tt.responseCampaigncall.CustomerID,
				tt.responseCampaigncall.FlowID,
				activeflow.ReferenceTypeCampaign,
				tt.responseCampaigncall.ID,
				uuid.Nil,
			).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, tt.responseActiveflow.ID).Return(nil)

			mockReq.EXPECT().CampaignV1CampaignExecute(ctx, tt.id, 500).Return(nil)

			h.Execute(ctx, tt.id)
		})
	}
}

func Test_getTarget(t *testing.T) {

	tests := []struct {
		name string

		c *campaign.Campaign
		p *outplan.Outplan

		responseOutdialtarget []omoutdialtarget.OutdialTarget

		expectRes *omoutdialtarget.OutdialTarget
	}{
		{
			"normal",

			&campaign.Campaign{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),
				},
				OutdialID: uuid.FromStringOrNil("bcad478e-c3fe-11ec-be28-a73254d6e0fc"),
				OutplanID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				QueueID:   uuid.Nil,
				FlowID:    uuid.FromStringOrNil("9574a6d6-c402-11ec-829f-33bd6e27d95f"),
				Status:    campaign.StatusRun,
				Type:      campaign.TypeFlow,
			},
			&outplan.Outplan{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				},
				MaxTryCount0: 4,
			},

			[]omoutdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
					Destination0: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					TryCount0: 0,
				},
			},

			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 0,
			},
		},
		{
			"return empty",

			&campaign.Campaign{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0859cc3a-c3fe-11ec-b496-67a067678522"),
				},
				OutdialID: uuid.FromStringOrNil("bcad478e-c3fe-11ec-be28-a73254d6e0fc"),
				OutplanID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				QueueID:   uuid.Nil,
				FlowID:    uuid.FromStringOrNil("9574a6d6-c402-11ec-829f-33bd6e27d95f"),
				Status:    campaign.StatusRun,
				Type:      campaign.TypeFlow,
			},
			&outplan.Outplan{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d6e3422-c3fe-11ec-a89c-736f5faee9c0"),
				},
				MaxTryCount0: 4,
			},

			[]omoutdialtarget.OutdialTarget{},

			nil,
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

			mockReq.EXPECT().OutdialV1OutdialtargetGetsAvailable(
				ctx,
				tt.c.OutdialID,
				tt.p.MaxTryCount0,
				tt.p.MaxTryCount1,
				tt.p.MaxTryCount2,
				tt.p.MaxTryCount3,
				tt.p.MaxTryCount4,
				1,
			).Return(tt.responseOutdialtarget, nil)

			res, err := h.getTarget(ctx, tt.c, tt.p)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_isDialableTarget(t *testing.T) {

	tests := []struct {
		name string

		target   *omoutdialtarget.OutdialTarget
		interval int

		expectRes bool
	}{
		{
			"normal",

			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 0,
				TMCreate:  "2022-04-18 03:00:17.995000",
				TMUpdate:  "2022-04-18 03:22:17.995000",
			},
			600000, // 10 min

			true,
		},
		{
			"never tried before",

			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 0,
				TMCreate:  "2020-04-18 03:22:17.995000",
				TMUpdate:  "2020-04-18 03:22:17.995000",
			},
			315360000000, // 10 years

			true,
		},
		{
			"interval not yet",

			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("1771246a-c3ff-11ec-8cf4-9fb7fc5301a8"),
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 0,
				TMCreate:  "2022-04-18 03:00:17.995000",
				TMUpdate:  "2022-04-18 03:22:17.995000",
			},
			315360000000, // 10 years

			false,
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

			res := h.isDialableTarget(ctx, tt.target, tt.interval)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_getTargetDestination(t *testing.T) {

	tests := []struct {
		name string

		target *omoutdialtarget.OutdialTarget
		plan   *outplan.Outplan

		expectResDestination      *commonaddress.Address
		expectResDestinationIndex int
		expectResTryCount         int
	}{
		{
			"normal",

			&omoutdialtarget.OutdialTarget{
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 0,
			},
			&outplan.Outplan{
				MaxTryCount0: 3,
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			0,
			1,
		},
		{
			"destination 0,1 given but selected 1",

			&omoutdialtarget.OutdialTarget{
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination1: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				TryCount0: 3,
				TryCount1: 0,
			},
			&outplan.Outplan{
				MaxTryCount0: 3,
				MaxTryCount1: 1,
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			1,
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
			h := &campaignHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			resDestination, resDestinationIndex, resTrycount := h.getTargetDestination(ctx, tt.target, tt.plan)

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

func Test_getTargetDestinationError(t *testing.T) {

	tests := []struct {
		name string

		target *omoutdialtarget.OutdialTarget
		plan   *outplan.Outplan
	}{
		{
			"normal",

			&omoutdialtarget.OutdialTarget{
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				TryCount0: 3,
			},
			&outplan.Outplan{
				MaxTryCount0: 2,
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
			h := &campaignHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			resDestination, resDestinationIndex, resTrycount := h.getTargetDestination(ctx, tt.target, tt.plan)
			if resDestination != nil || resDestinationIndex != 0 || resTrycount != 0 {
				t.Errorf("Wrong match. expect: nil, got: %v", resDestination)
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
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c1ec0278-c406-11ec-8804-c738ff121aa6"),
					},
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

			mockReq.EXPECT().QueueV1QueueGetAgents(ctx, tt.queueID, gomock.Any()).Return(tt.responseAgents, nil)
			mockCampaigncall.EXPECT().ListByCampaignIDAndStatus(ctx, tt.campaignID, campaigncall.StatusDialing, gomock.Any(), uint64(100)).Return(tt.responseCampaingcalls, nil)

			res := h.isDialable(ctx, tt.campaignID, tt.queueID, tt.serviceLevel)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
