package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cacampaign "monorepo/bin-campaign-manager/models/campaign"

	fmaction "monorepo/bin-flow-manager/models/action"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_CampaignCreate(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		campaignName string
		detail       string

		campaignType   cacampaign.Type
		serviceLevel   int
		endHandle      cacampaign.EndHandle
		actions        []fmaction.Action
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"test name",
			"test detail",

			cacampaign.TypeCall,
			100,
			cacampaign.EndHandleStop,
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			uuid.FromStringOrNil("a44727da-c653-11ec-a0b7-a7d6b873b66d"),
			uuid.FromStringOrNil("a4aafd28-c653-11ec-9b79-47790e39b9be"),
			uuid.FromStringOrNil("a4e4ccce-c653-11ec-b64b-1b6af5c458a8"),
			uuid.FromStringOrNil("a51c8010-c653-11ec-953a-43eabdb60873"),

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c5edb1ce-c653-11ec-bb63-1f0413e1ebdc"),
				},
			},
			&cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c5edb1ce-c653-11ec-bb63-1f0413e1ebdc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignCreate(ctx, uuid.Nil, tt.agent.CustomerID, tt.campaignType, tt.campaignName, tt.detail, tt.serviceLevel, tt.endHandle, tt.actions, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID).Return(tt.response, nil)
			res, err := h.CampaignCreate(ctx, tt.agent, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle, tt.actions, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		response  []cacampaign.Campaign
		expectRes []*cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]cacampaign.Campaign{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bf203708-c654-11ec-910b-63ef1793516d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bf4959a8-c654-11ec-bc10-53da5a6de123"),
					},
				},
			},
			[]*cacampaign.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bf203708-c654-11ec-910b-63ef1793516d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bf4959a8-c654-11ec-bc10-53da5a6de123"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGets(ctx, tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.response, nil)
			res, err := h.CampaignGetsByCustomerID(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignGet(t *testing.T) {

	tests := []struct {
		name       string
		agent      *amagent.Agent
		campaignID uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			res, err := h.CampaignGet(ctx, tt.agent, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignDelete(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		id    uuid.UUID

		responseCampaign *cacampaign.Campaign
		expectRes        *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("bcbcffb4-c640-11ec-bdab-03b1d679601d"),

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("32d63a4e-c655-11ec-8288-2707fffc29b5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("32d63a4e-c655-11ec-8288-2707fffc29b5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)
			mockReq.EXPECT().CampaignV1CampaignDelete(ctx, tt.id).Return(tt.responseCampaign, nil)
			res, err := h.CampaignDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name         string
		agent        *amagent.Agent
		campaignID   uuid.UUID
		campaignName string
		detail       string
		campaignType cacampaign.Type
		serviceLevel int
		endHandle    cacampaign.EndHandle

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			campaignID:   uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
			campaignName: "test name",
			detail:       "test detail",
			campaignType: cacampaign.TypeCall,
			serviceLevel: 100,
			endHandle:    cacampaign.EndHandleContinue,

			response: &cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateBasicInfo(ctx, tt.campaignID, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle).Return(tt.response, nil)
			res, err := h.CampaignUpdateBasicInfo(ctx, tt.agent, tt.campaignID, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateStatus(t *testing.T) {

	tests := []struct {
		name       string
		agent      *amagent.Agent
		campaignID uuid.UUID
		status     cacampaign.Status

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
			cacampaign.StatusRun,

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateStatus(ctx, tt.campaignID, tt.status).Return(tt.response, nil)
			res, err := h.CampaignUpdateStatus(ctx, tt.agent, tt.campaignID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateServiceLevel(t *testing.T) {

	tests := []struct {
		name         string
		agent        *amagent.Agent
		campaignID   uuid.UUID
		serviceLevel int

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
			100,

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateServiceLevel(ctx, tt.campaignID, tt.serviceLevel).Return(tt.response, nil)
			res, err := h.CampaignUpdateServiceLevel(ctx, tt.agent, tt.campaignID, tt.serviceLevel)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateActions(t *testing.T) {

	tests := []struct {
		name       string
		agent      *amagent.Agent
		campaignID uuid.UUID
		actions    []fmaction.Action

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("eb889654-c655-11ec-a97a-636c4c1455d8"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eb889654-c655-11ec-a97a-636c4c1455d8"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eb889654-c655-11ec-a97a-636c4c1455d8"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateActions(ctx, tt.campaignID, tt.actions).Return(tt.response, nil)
			res, err := h.CampaignUpdateActions(ctx, tt.agent, tt.campaignID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateResourceInfo(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		campaignID     uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			campaignID:     uuid.FromStringOrNil("6589627a-c6b6-11ec-80ec-eb94b8bc76e7"),
			outplanID:      uuid.FromStringOrNil("65c34a94-c6b6-11ec-b153-6be43b327a5e"),
			outdialID:      uuid.FromStringOrNil("65ef10ca-c6b6-11ec-93a6-af4ca7079371"),
			queueID:        uuid.FromStringOrNil("661dcbc2-c6b6-11ec-934c-c3c128f1d3b9"),
			nextCampaignID: uuid.FromStringOrNil("d1b8ec48-7cd3-11ee-abac-3f9b33935d6a"),

			response: &cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6589627a-c6b6-11ec-80ec-eb94b8bc76e7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6589627a-c6b6-11ec-80ec-eb94b8bc76e7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateResourceInfo(ctx, tt.campaignID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID).Return(tt.response, nil)
			res, err := h.CampaignUpdateResourceInfo(ctx, tt.agent, tt.campaignID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateNextCampaignID(t *testing.T) {

	tests := []struct {
		name           string
		agent          *amagent.Agent
		campaignID     uuid.UUID
		nextCampaignID uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("916c14be-c6b6-11ec-83a5-8f67784590f9"),
			uuid.FromStringOrNil("919d20e0-c6b6-11ec-bdc6-9f571a70547e"),

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("916c14be-c6b6-11ec-83a5-8f67784590f9"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaign.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("916c14be-c6b6-11ec-83a5-8f67784590f9"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateNextCampaignID(ctx, tt.campaignID, tt.nextCampaignID).Return(tt.response, nil)
			res, err := h.CampaignUpdateNextCampaignID(ctx, tt.agent, tt.campaignID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
