package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/cachehandler"
)

func Test_CampaignCreate(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		responseCurTime string

		expectRes *campaign.Campaign
	}{
		{
			"type call with endhandle stop",
			&campaign.Campaign{
				ID:           uuid.FromStringOrNil("b9d134a2-b3ce-11ec-87b1-df25314b0e76"),
				CustomerID:   uuid.FromStringOrNil("b9f87f80-b3ce-11ec-8442-537a6b140131"),
				Type:         campaign.TypeCall,
				Execute:      campaign.ExecuteStop,
				Name:         "test name",
				Detail:       "test detail",
				Status:       campaign.StatusStop,
				ServiceLevel: 0,
				EndHandle:    campaign.EndHandleStop,
				FlowID:       uuid.FromStringOrNil("8aaeab73-36ce-4ac7-9dd2-2e21fc6210b1"),
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				OutplanID:      uuid.FromStringOrNil("ba29f006-b3ce-11ec-80d2-a71d2212a7d7"),
				OutdialID:      uuid.FromStringOrNil("ba5c57c6-b3ce-11ec-b997-4b54d7754db6"),
				QueueID:        uuid.FromStringOrNil("ba91a87c-b3ce-11ec-993c-2f5317fef011"),
				NextCampaignID: uuid.FromStringOrNil("bc7a45f4-b3ce-11ec-978f-ebb914007273"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:           uuid.FromStringOrNil("b9d134a2-b3ce-11ec-87b1-df25314b0e76"),
				CustomerID:   uuid.FromStringOrNil("b9f87f80-b3ce-11ec-8442-537a6b140131"),
				Type:         campaign.TypeCall,
				Execute:      campaign.ExecuteStop,
				Name:         "test name",
				Detail:       "test detail",
				Status:       campaign.StatusStop,
				ServiceLevel: 0,
				EndHandle:    campaign.EndHandleStop,
				FlowID:       uuid.FromStringOrNil("8aaeab73-36ce-4ac7-9dd2-2e21fc6210b1"),
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				OutplanID:      uuid.FromStringOrNil("ba29f006-b3ce-11ec-80d2-a71d2212a7d7"),
				OutdialID:      uuid.FromStringOrNil("ba5c57c6-b3ce-11ec-b997-4b54d7754db6"),
				QueueID:        uuid.FromStringOrNil("ba91a87c-b3ce-11ec-993c-2f5317fef011"),
				NextCampaignID: uuid.FromStringOrNil("bc7a45f4-b3ce-11ec-978f-ebb914007273"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
		},
		{
			"type flow endhandle continue",
			&campaign.Campaign{
				ID:           uuid.FromStringOrNil("18ebc9ba-8765-4171-8b21-36f8792384ce"),
				CustomerID:   uuid.FromStringOrNil("b9f87f80-b3ce-11ec-8442-537a6b140131"),
				Type:         campaign.TypeFlow,
				Execute:      campaign.ExecuteRun,
				Name:         "test name",
				Detail:       "test detail",
				Status:       campaign.StatusStop,
				ServiceLevel: 0,
				EndHandle:    campaign.EndHandleContinue,
				FlowID:       uuid.FromStringOrNil("18972935-c90f-4e9b-bbd0-67aa9ae934b1"),
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				OutplanID:      uuid.FromStringOrNil("ba29f006-b3ce-11ec-80d2-a71d2212a7d7"),
				OutdialID:      uuid.FromStringOrNil("ba5c57c6-b3ce-11ec-b997-4b54d7754db6"),
				QueueID:        uuid.FromStringOrNil("ba91a87c-b3ce-11ec-993c-2f5317fef011"),
				NextCampaignID: uuid.FromStringOrNil("bc7a45f4-b3ce-11ec-978f-ebb914007273"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:           uuid.FromStringOrNil("18ebc9ba-8765-4171-8b21-36f8792384ce"),
				CustomerID:   uuid.FromStringOrNil("b9f87f80-b3ce-11ec-8442-537a6b140131"),
				Type:         campaign.TypeFlow,
				Execute:      campaign.ExecuteRun,
				Name:         "test name",
				Detail:       "test detail",
				Status:       campaign.StatusStop,
				ServiceLevel: 0,
				EndHandle:    campaign.EndHandleContinue,
				FlowID:       uuid.FromStringOrNil("18972935-c90f-4e9b-bbd0-67aa9ae934b1"),
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				OutplanID:      uuid.FromStringOrNil("ba29f006-b3ce-11ec-80d2-a71d2212a7d7"),
				OutdialID:      uuid.FromStringOrNil("ba5c57c6-b3ce-11ec-b997-4b54d7754db6"),
				QueueID:        uuid.FromStringOrNil("ba91a87c-b3ce-11ec-993c-2f5317fef011"),
				NextCampaignID: uuid.FromStringOrNil("bc7a45f4-b3ce-11ec-978f-ebb914007273"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.campaign, res)
			}
		})
	}
}

func Test_CampaignDelete(t *testing.T) {

	tests := []struct {
		name     string
		campaign *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("1cc92874-b480-11ec-b7cf-4f5d95304498"),
				CustomerID:     uuid.FromStringOrNil("1cf4905e-b480-11ec-8e27-038c9a252614"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("ba29f006-b3ce-11ec-80d2-a71d2212a7d7"),
				OutdialID:      uuid.FromStringOrNil("ba5c57c6-b3ce-11ec-b997-4b54d7754db6"),
				QueueID:        uuid.FromStringOrNil("ba91a87c-b3ce-11ec-993c-2f5317fef011"),
				NextCampaignID: uuid.FromStringOrNil("bc7a45f4-b3ce-11ec-978f-ebb914007273"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CampaignSet(gomock.Any(), gomock.Any())
			if err := h.CampaignCreate(context.Background(), tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignSet(gomock.Any(), gomock.Any())
			if err := h.CampaignDelete(context.Background(), tt.campaign.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(gomock.Any(), tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(gomock.Any(), gomock.Any())
			res, err := h.CampaignGet(context.Background(), tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}
		})
	}
}

func Test_CampaignGetsByCustomerID(t *testing.T) {
	tests := []struct {
		name      string
		campaigns []*campaign.Campaign

		customerID uuid.UUID
		token      string
		limit      uint64

		responseCurTime string
		expectRes       []*campaign.Campaign
	}{
		{
			"1 item",
			[]*campaign.Campaign{
				{
					ID:           uuid.FromStringOrNil("f902e478-b3d2-11ec-838c-f3f66784d081"),
					CustomerID:   uuid.FromStringOrNil("f940793c-b3d2-11ec-8a3e-2f48bac6f31a"),
					Name:         "test name",
					Detail:       "test detail",
					Status:       campaign.StatusStop,
					ServiceLevel: 10,
					EndHandle:    campaign.EndHandleStop,
					FlowID:       uuid.FromStringOrNil("7d469238-eb40-481f-99dd-bc59bb1d38f7"),
					Actions: []fmaction.Action{
						{
							Type: fmaction.TypeAnswer,
						},
					},
					OutplanID:      uuid.FromStringOrNil("f9771d70-b3d2-11ec-9154-dfb637b4a732"),
					OutdialID:      uuid.FromStringOrNil("f9a4deb8-b3d2-11ec-8ced-cfd5fa2a7c1b"),
					QueueID:        uuid.FromStringOrNil("f9ce6d96-b3d2-11ec-94ac-bb22aad0488d"),
					NextCampaignID: uuid.FromStringOrNil("f9f84bf2-b3d2-11ec-8a68-d7464098d793"),
				},
			},

			uuid.FromStringOrNil("f940793c-b3d2-11ec-8a3e-2f48bac6f31a"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:18.995000",
			[]*campaign.Campaign{
				{
					ID:           uuid.FromStringOrNil("f902e478-b3d2-11ec-838c-f3f66784d081"),
					CustomerID:   uuid.FromStringOrNil("f940793c-b3d2-11ec-8a3e-2f48bac6f31a"),
					Name:         "test name",
					Detail:       "test detail",
					Status:       campaign.StatusStop,
					ServiceLevel: 10,
					EndHandle:    campaign.EndHandleStop,
					FlowID:       uuid.FromStringOrNil("7d469238-eb40-481f-99dd-bc59bb1d38f7"),
					Actions: []fmaction.Action{
						{
							Type: fmaction.TypeAnswer,
						},
					},
					OutplanID:      uuid.FromStringOrNil("f9771d70-b3d2-11ec-9154-dfb637b4a732"),
					OutdialID:      uuid.FromStringOrNil("f9a4deb8-b3d2-11ec-8ced-cfd5fa2a7c1b"),
					QueueID:        uuid.FromStringOrNil("f9ce6d96-b3d2-11ec-94ac-bb22aad0488d"),
					NextCampaignID: uuid.FromStringOrNil("f9f84bf2-b3d2-11ec-8a68-d7464098d793"),
					TMCreate:       "2020-04-18 03:22:18.995000",
					TMUpdate:       DefaultTimeStamp,
					TMDelete:       DefaultTimeStamp,
				},
			},
		},
		{
			"2 items",
			[]*campaign.Campaign{
				{
					ID:             uuid.FromStringOrNil("4392eaa6-b3d3-11ec-94aa-339707f75f8e"),
					CustomerID:     uuid.FromStringOrNil("49e070b8-b3d3-11ec-9b5f-0f066e1f46e6"),
					Name:           "test name",
					Detail:         "test detail",
					Status:         campaign.StatusStop,
					OutplanID:      uuid.FromStringOrNil("f9771d70-b3d2-11ec-9154-dfb637b4a732"),
					OutdialID:      uuid.FromStringOrNil("f9a4deb8-b3d2-11ec-8ced-cfd5fa2a7c1b"),
					QueueID:        uuid.FromStringOrNil("f9ce6d96-b3d2-11ec-94ac-bb22aad0488d"),
					NextCampaignID: uuid.FromStringOrNil("f9f84bf2-b3d2-11ec-8a68-d7464098d793"),
				},
				{
					ID:             uuid.FromStringOrNil("43c183a2-b3d3-11ec-8bd7-b39a3d003ed6"),
					CustomerID:     uuid.FromStringOrNil("49e070b8-b3d3-11ec-9b5f-0f066e1f46e6"),
					Name:           "test name",
					Detail:         "test detail",
					Status:         campaign.StatusStop,
					OutplanID:      uuid.FromStringOrNil("f9771d70-b3d2-11ec-9154-dfb637b4a732"),
					OutdialID:      uuid.FromStringOrNil("f9a4deb8-b3d2-11ec-8ced-cfd5fa2a7c1b"),
					QueueID:        uuid.FromStringOrNil("f9ce6d96-b3d2-11ec-94ac-bb22aad0488d"),
					NextCampaignID: uuid.FromStringOrNil("f9f84bf2-b3d2-11ec-8a68-d7464098d793"),
				},
			},

			uuid.FromStringOrNil("49e070b8-b3d3-11ec-9b5f-0f066e1f46e6"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:17.995000",
			[]*campaign.Campaign{
				{
					ID:             uuid.FromStringOrNil("43c183a2-b3d3-11ec-8bd7-b39a3d003ed6"),
					CustomerID:     uuid.FromStringOrNil("49e070b8-b3d3-11ec-9b5f-0f066e1f46e6"),
					Name:           "test name",
					Detail:         "test detail",
					Status:         campaign.StatusStop,
					OutplanID:      uuid.FromStringOrNil("f9771d70-b3d2-11ec-9154-dfb637b4a732"),
					OutdialID:      uuid.FromStringOrNil("f9a4deb8-b3d2-11ec-8ced-cfd5fa2a7c1b"),
					QueueID:        uuid.FromStringOrNil("f9ce6d96-b3d2-11ec-94ac-bb22aad0488d"),
					NextCampaignID: uuid.FromStringOrNil("f9f84bf2-b3d2-11ec-8a68-d7464098d793"),
					TMCreate:       "2020-04-18 03:22:17.995000",
					TMUpdate:       DefaultTimeStamp,
					TMDelete:       DefaultTimeStamp,
				},
				{
					ID:             uuid.FromStringOrNil("4392eaa6-b3d3-11ec-94aa-339707f75f8e"),
					CustomerID:     uuid.FromStringOrNil("49e070b8-b3d3-11ec-9b5f-0f066e1f46e6"),
					Name:           "test name",
					Detail:         "test detail",
					Status:         campaign.StatusStop,
					OutplanID:      uuid.FromStringOrNil("f9771d70-b3d2-11ec-9154-dfb637b4a732"),
					OutdialID:      uuid.FromStringOrNil("f9a4deb8-b3d2-11ec-8ced-cfd5fa2a7c1b"),
					QueueID:        uuid.FromStringOrNil("f9ce6d96-b3d2-11ec-94ac-bb22aad0488d"),
					NextCampaignID: uuid.FromStringOrNil("f9f84bf2-b3d2-11ec-8a68-d7464098d793"),
					TMCreate:       "2020-04-18 03:22:17.995000",
					TMUpdate:       DefaultTimeStamp,
					TMDelete:       DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, p := range tt.campaigns {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
				if err := h.CampaignCreate(ctx, p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CampaignGetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created outdial. outdial: %v", res)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_CampaignUpdateBasicInfo(t *testing.T) {
	tests := []struct {
		name string
		data *campaign.Campaign

		campaignName string
		detail       string
		campaignType campaign.Type
		serviceLevel int
		endHandle    campaign.EndHandle

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			name: "normal",
			data: &campaign.Campaign{
				ID:             uuid.FromStringOrNil("cadc92e6-b3d3-11ec-a6ec-ab2d193ab762"),
				CustomerID:     uuid.FromStringOrNil("cb089aa8-b3d3-11ec-b831-0b9f457c4610"),
				Name:           "test name",
				Detail:         "test detail",
				Type:           campaign.TypeCall,
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("cb3ae990-b3d3-11ec-9393-5b195f388b72"),
				OutdialID:      uuid.FromStringOrNil("cb6a0d7e-b3d3-11ec-ae5f-1b6bf31623b8"),
				QueueID:        uuid.FromStringOrNil("cb9ec816-b3d3-11ec-8fea-578f80554cac"),
				NextCampaignID: uuid.FromStringOrNil("cbce4438-b3d3-11ec-9b72-cf0786bc7233"),
			},

			campaignName: "update name",
			detail:       "update detail",
			campaignType: campaign.TypeFlow,
			serviceLevel: 100,
			endHandle:    campaign.EndHandleContinue,

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &campaign.Campaign{
				ID:             uuid.FromStringOrNil("cadc92e6-b3d3-11ec-a6ec-ab2d193ab762"),
				CustomerID:     uuid.FromStringOrNil("cb089aa8-b3d3-11ec-b831-0b9f457c4610"),
				Name:           "update name",
				Detail:         "update detail",
				Type:           campaign.TypeFlow,
				Status:         campaign.StatusStop,
				ServiceLevel:   100,
				EndHandle:      campaign.EndHandleContinue,
				OutplanID:      uuid.FromStringOrNil("cb3ae990-b3d3-11ec-9393-5b195f388b72"),
				OutdialID:      uuid.FromStringOrNil("cb6a0d7e-b3d3-11ec-ae5f-1b6bf31623b8"),
				QueueID:        uuid.FromStringOrNil("cb9ec816-b3d3-11ec-8fea-578f80554cac"),
				NextCampaignID: uuid.FromStringOrNil("cbce4438-b3d3-11ec-9b72-cf0786bc7233"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateBasicInfo(ctx, tt.data.ID, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateResourceInfo(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			name: "normal",
			campaign: &campaign.Campaign{
				ID:             uuid.FromStringOrNil("2928a27c-b3d4-11ec-93ea-932164cd844b"),
				CustomerID:     uuid.FromStringOrNil("29594044-b3d4-11ec-98b2-730b7dd059bf"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
			},

			outplanID:      uuid.FromStringOrNil("2a56c49e-b3d4-11ec-adfe-b38bfdbaca15"),
			outdialID:      uuid.FromStringOrNil("2a8c690a-b3d4-11ec-b837-4fc7444b844f"),
			queueID:        uuid.FromStringOrNil("2ac5856e-b3d4-11ec-999d-376dbe88d746"),
			nextCampaignID: uuid.FromStringOrNil("7b753100-7ccb-11ee-a39a-5faf460167f7"),

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &campaign.Campaign{
				ID:             uuid.FromStringOrNil("2928a27c-b3d4-11ec-93ea-932164cd844b"),
				CustomerID:     uuid.FromStringOrNil("29594044-b3d4-11ec-98b2-730b7dd059bf"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("2a56c49e-b3d4-11ec-adfe-b38bfdbaca15"),
				OutdialID:      uuid.FromStringOrNil("2a8c690a-b3d4-11ec-b837-4fc7444b844f"),
				QueueID:        uuid.FromStringOrNil("2ac5856e-b3d4-11ec-999d-376dbe88d746"),
				NextCampaignID: uuid.FromStringOrNil("7b753100-7ccb-11ee-a39a-5faf460167f7"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(context.Background(), tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateResourceInfo(ctx, tt.campaign.ID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateNextCampaignID(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		nextCampaignID uuid.UUID

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("ba9569b6-b3d4-11ec-854c-778329a51158"),
				CustomerID:     uuid.FromStringOrNil("bac2fe58-b3d4-11ec-b992-f7d429877f14"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
			},

			uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("ba9569b6-b3d4-11ec-854c-778329a51158"),
				CustomerID:     uuid.FromStringOrNil("bac2fe58-b3d4-11ec-b992-f7d429877f14"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(context.Background(), tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateNextCampaignID(ctx, tt.campaign.ID, tt.nextCampaignID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(gomock.Any(), tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(gomock.Any(), gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateStatus(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		status campaign.Status

		responseCurTime string

		expectRes *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("ef762a33-24ef-486a-bb91-5496456ebaa5"),
				CustomerID:     uuid.FromStringOrNil("4117518d-ab99-45d2-813c-b32f78efa09b"),
				Type:           campaign.TypeCall,
				Execute:        campaign.ExecuteStop,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
			},

			campaign.StatusRun,

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("ef762a33-24ef-486a-bb91-5496456ebaa5"),
				CustomerID:     uuid.FromStringOrNil("4117518d-ab99-45d2-813c-b32f78efa09b"),
				Type:           campaign.TypeCall,
				Execute:        campaign.ExecuteStop,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusRun,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateStatus(ctx, tt.campaign.ID, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateStatusAndExecute(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		status  campaign.Status
		execute campaign.Execute

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("1b18e1e0-18a7-4a8f-b0ca-5f99597a0bea"),
				CustomerID:     uuid.FromStringOrNil("c329475f-11fd-452b-b838-d4d079090065"),
				Type:           campaign.TypeCall,
				Execute:        campaign.ExecuteStop,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
			},

			campaign.StatusRun,
			campaign.ExecuteRun,

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("1b18e1e0-18a7-4a8f-b0ca-5f99597a0bea"),
				CustomerID:     uuid.FromStringOrNil("c329475f-11fd-452b-b838-d4d079090065"),
				Type:           campaign.TypeCall,
				Execute:        campaign.ExecuteRun,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusRun,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateStatusAndExecute(ctx, tt.campaign.ID, tt.status, tt.execute); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateExecute(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		execute campaign.Execute

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("41089aad-d8c0-4685-a609-8ab9f264ab74"),
				CustomerID:     uuid.FromStringOrNil("b901d73c-a5fd-464e-9684-dc9f6970e6b3"),
				Type:           campaign.TypeCall,
				Execute:        campaign.ExecuteStop,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
			},

			campaign.ExecuteRun,

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("41089aad-d8c0-4685-a609-8ab9f264ab74"),
				CustomerID:     uuid.FromStringOrNil("b901d73c-a5fd-464e-9684-dc9f6970e6b3"),
				Type:           campaign.TypeCall,
				Execute:        campaign.ExecuteRun,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("2a21ba1a-b3d4-11ec-a5cf-bf03f62e70c7"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateExecute(ctx, tt.campaign.ID, tt.execute); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateServiceLevel(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		serviceLevel int

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("a1bf91b2-b741-11ec-8260-47e8860a9e3b"),
				CustomerID:     uuid.FromStringOrNil("a1eca4cc-b741-11ec-85e0-8bb267cad154"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				ServiceLevel:   100,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
			},

			300,

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("a1bf91b2-b741-11ec-8260-47e8860a9e3b"),
				CustomerID:     uuid.FromStringOrNil("a1eca4cc-b741-11ec-85e0-8bb267cad154"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				ServiceLevel:   300,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateServiceLevel(ctx, tt.campaign.ID, tt.serviceLevel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateEndHandle(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		endHandle campaign.EndHandle

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("38a47dc8-704e-4818-bb16-113335ed85ec"),
				CustomerID:     uuid.FromStringOrNil("a1eca4cc-b741-11ec-85e0-8bb267cad154"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				ServiceLevel:   100,
				EndHandle:      campaign.EndHandleStop,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
			},

			campaign.EndHandleContinue,

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("38a47dc8-704e-4818-bb16-113335ed85ec"),
				CustomerID:     uuid.FromStringOrNil("a1eca4cc-b741-11ec-85e0-8bb267cad154"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				ServiceLevel:   100,
				EndHandle:      campaign.EndHandleContinue,
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateEndHandle(ctx, tt.campaign.ID, tt.endHandle); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateActions(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		actions []fmaction.Action

		responseCurTime string
		expectRes       *campaign.Campaign
	}{
		{
			"normal",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("20e8968f-b36b-4b2c-acc7-d4240724d967"),
				CustomerID:     uuid.FromStringOrNil("a1f09a50-f917-4de5-a46f-1a8c1bb0afbc"),
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				ServiceLevel:   100,
				EndHandle:      campaign.EndHandleStop,
				FlowID:         uuid.FromStringOrNil("4a2530aa-9ea7-4441-a137-dfcf54e0f609"),
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
			},

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:           uuid.FromStringOrNil("20e8968f-b36b-4b2c-acc7-d4240724d967"),
				CustomerID:   uuid.FromStringOrNil("a1f09a50-f917-4de5-a46f-1a8c1bb0afbc"),
				Name:         "test name",
				Detail:       "test detail",
				Status:       campaign.StatusStop,
				ServiceLevel: 100,
				EndHandle:    campaign.EndHandleStop,
				FlowID:       uuid.FromStringOrNil("4a2530aa-9ea7-4441-a137-dfcf54e0f609"),
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateActions(ctx, tt.campaign.ID, tt.actions); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateType(t *testing.T) {
	tests := []struct {
		name     string
		campaign *campaign.Campaign

		campaignType campaign.Type

		responseCurtime string
		expectRes       *campaign.Campaign
	}{
		{
			"update to typecall",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("da627f5c-52c3-4509-b80d-505d346e631f"),
				CustomerID:     uuid.FromStringOrNil("a1f09a50-f917-4de5-a46f-1a8c1bb0afbc"),
				Type:           campaign.TypeFlow,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				ServiceLevel:   100,
				EndHandle:      campaign.EndHandleStop,
				FlowID:         uuid.FromStringOrNil("4a2530aa-9ea7-4441-a137-dfcf54e0f609"),
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
			},

			campaign.TypeCall,

			"2020-04-18 03:22:17.995000",
			&campaign.Campaign{
				ID:             uuid.FromStringOrNil("da627f5c-52c3-4509-b80d-505d346e631f"),
				CustomerID:     uuid.FromStringOrNil("a1f09a50-f917-4de5-a46f-1a8c1bb0afbc"),
				Type:           campaign.TypeCall,
				Name:           "test name",
				Detail:         "test detail",
				Status:         campaign.StatusStop,
				ServiceLevel:   100,
				EndHandle:      campaign.EndHandleStop,
				FlowID:         uuid.FromStringOrNil("4a2530aa-9ea7-4441-a137-dfcf54e0f609"),
				OutplanID:      uuid.FromStringOrNil("298c7482-b3d4-11ec-9ea5-ef75a2e6bfb6"),
				OutdialID:      uuid.FromStringOrNil("29b93706-b3d4-11ec-b884-57ba15a12519"),
				QueueID:        uuid.FromStringOrNil("29f12d00-b3d4-11ec-a884-dba81c6dc4da"),
				NextCampaignID: uuid.FromStringOrNil("baf03152-b3d4-11ec-bfe4-eb0cddbd111d"),
				TMCreate:       "2020-04-18 03:22:17.995000",
				TMUpdate:       "2020-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurtime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignCreate(ctx, tt.campaign); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurtime)
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaignUpdateType(ctx, tt.campaign.ID, tt.campaignType); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaignGet(ctx, tt.campaign.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaignSet(ctx, gomock.Any())
			res, err := h.CampaignGet(ctx, tt.campaign.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
