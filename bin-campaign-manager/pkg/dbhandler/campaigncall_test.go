package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/cachehandler"
)

func Test_CampaigncallCreate(t *testing.T) {
	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall

		responseCurTime string
		expectRes       *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5ed54e04-b4fe-11ec-bab7-1bbc3ac23720"),
					CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
				},
				CampaignID:      uuid.FromStringOrNil("5f3bf276-b4fe-11ec-b032-47340d4fb85e"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				FlowID:          uuid.FromStringOrNil("e27259f8-c446-11ec-8a54-13cf3fd4f66a"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusProgressing,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
			},

			"2020-04-18 03:22:17.995000",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5ed54e04-b4fe-11ec-bab7-1bbc3ac23720"),
					CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
				},
				CampaignID:      uuid.FromStringOrNil("5f3bf276-b4fe-11ec-b032-47340d4fb85e"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				FlowID:          uuid.FromStringOrNil("e27259f8-c446-11ec-8a54-13cf3fd4f66a"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusProgressing,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         DefaultTimeStamp,
				TMDelete:         DefaultTimeStamp,
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
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallCreate(context.Background(), tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaigncallGet(gomock.Any(), tt.campaigncall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaigncallSet(gomock.Any(), gomock.Any())
			res, err := h.CampaigncallGet(ctx, tt.campaigncall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaigncallGetByReferenceID(t *testing.T) {
	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall

		responseCurTime string
		expectRes       *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ab033dd-1509-4f25-b831-8d1ccfdf3c92"),
					CustomerID: uuid.FromStringOrNil("d993bcab-03de-40d1-bf91-e94ae143a420"),
				},
				CampaignID:      uuid.FromStringOrNil("5f3bf276-b4fe-11ec-b032-47340d4fb85e"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("033d0d0b-1ce1-4c9a-94cf-e205db00cd39"),
				Status:          campaigncall.StatusProgressing,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
			},

			"2020-04-18 03:22:17.995000",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ab033dd-1509-4f25-b831-8d1ccfdf3c92"),
					CustomerID: uuid.FromStringOrNil("d993bcab-03de-40d1-bf91-e94ae143a420"),
				},
				CampaignID:      uuid.FromStringOrNil("5f3bf276-b4fe-11ec-b032-47340d4fb85e"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("033d0d0b-1ce1-4c9a-94cf-e205db00cd39"),
				Status:          campaigncall.StatusProgressing,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         DefaultTimeStamp,
				TMDelete:         DefaultTimeStamp,
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
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallCreate(ctx, tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CampaigncallGetByReferenceID(ctx, tt.campaigncall.ReferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaigncallGetByActiveflowID(t *testing.T) {
	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall

		responseCurTime string

		expectRes *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ff63cbaf-2221-4631-9b0d-215784734f80"),
					CustomerID: uuid.FromStringOrNil("b4c8a95c-f263-44cb-870b-e36c452b4eb4"),
				},
				CampaignID:      uuid.FromStringOrNil("5f3bf276-b4fe-11ec-b032-47340d4fb85e"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("951540b8-d39d-4b1a-b696-1621aa90d3ee"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("1da2d1f6-8ea2-4274-9bc3-225fee645cb8"),
				Status:          campaigncall.StatusProgressing,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
			},

			"2020-04-18 03:22:17.995000",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ff63cbaf-2221-4631-9b0d-215784734f80"),
					CustomerID: uuid.FromStringOrNil("b4c8a95c-f263-44cb-870b-e36c452b4eb4"),
				},
				CampaignID:      uuid.FromStringOrNil("5f3bf276-b4fe-11ec-b032-47340d4fb85e"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("951540b8-d39d-4b1a-b696-1621aa90d3ee"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("1da2d1f6-8ea2-4274-9bc3-225fee645cb8"),
				Status:          campaigncall.StatusProgressing,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         DefaultTimeStamp,
				TMDelete:         DefaultTimeStamp,
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
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallCreate(ctx, tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CampaigncallGetByActiveflowID(ctx, tt.campaigncall.ActiveflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaigncallListByCustomerID(t *testing.T) {
	tests := []struct {
		name string
		data []*campaigncall.Campaigncall

		customerID uuid.UUID
		token      string
		limit      uint64

		responseCurTime string
		expectRes       []*campaigncall.Campaigncall
	}{
		{
			name: "1 item",
			data: []*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3cf4996a-6e30-11ee-b4df-e3435f75c8d9"),
						CustomerID: uuid.FromStringOrNil("3d286678-6e30-11ee-82b1-d7f075ddecab"),
					},
				},
			},

			customerID: uuid.FromStringOrNil("3d286678-6e30-11ee-82b1-d7f075ddecab"),
			token:      "2022-04-18 03:22:17.995000",
			limit:      100,

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3cf4996a-6e30-11ee-b4df-e3435f75c8d9"),
						CustomerID: uuid.FromStringOrNil("3d286678-6e30-11ee-82b1-d7f075ddecab"),
					},
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
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

			for _, p := range tt.data {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
				if err := h.CampaigncallCreate(context.Background(), p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CampaigncallListByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_CampaigncallListByCampaignID(t *testing.T) {
	tests := []struct {
		name          string
		campaigncalls []*campaigncall.Campaigncall

		campaignID uuid.UUID
		token      string
		limit      uint64

		responseCurTime string
		expectRes       []*campaigncall.Campaigncall
	}{
		{
			"1 item",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("82b47bbe-b4ff-11ec-8b2a-4f8b8358d3ef"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("82dbd470-b4ff-11ec-b0f4-db46cf5d928e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
			},

			uuid.FromStringOrNil("82dbd470-b4ff-11ec-b0f4-db46cf5d928e"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:17.995000",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("82b47bbe-b4ff-11ec-8b2a-4f8b8358d3ef"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("82dbd470-b4ff-11ec-b0f4-db46cf5d928e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:17.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
				},
			},
		},
		{
			"2 items",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("da5fbd24-b4ff-11ec-9317-fb5fd7c5fd4f"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("da92f482-b4ff-11ec-a879-e3cde17ee57d"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
			},

			uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:17.995000",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("da92f482-b4ff-11ec-a879-e3cde17ee57d"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:17.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("da5fbd24-b4ff-11ec-9317-fb5fd7c5fd4f"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:17.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
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

			for _, p := range tt.campaigncalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
				if err := h.CampaigncallCreate(context.Background(), p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CampaigncallListByCampaignID(ctx, tt.campaignID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_CampaigncallListByCampaignIDAndStatus(t *testing.T) {
	tests := []struct {
		name          string
		campaigncalls []*campaigncall.Campaigncall

		campaignID uuid.UUID
		status     campaigncall.Status
		token      string
		limit      uint64

		responseCurTime string
		expectRes       []*campaigncall.Campaigncall
	}{
		{
			"1 item",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("622497f2-b500-11ec-9f5d-4776190aaae1"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("624d111e-b500-11ec-a605-e3f84329977e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
			},

			uuid.FromStringOrNil("624d111e-b500-11ec-a605-e3f84329977e"),
			campaigncall.StatusProgressing,
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:17.995000",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("622497f2-b500-11ec-9f5d-4776190aaae1"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("624d111e-b500-11ec-a605-e3f84329977e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:17.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
				},
			},
		},
		{
			"2 items",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("6275f0ac-b500-11ec-b356-b3eb98e5d2cb"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("62a16e1c-b500-11ec-82e0-a31b4abba9ba"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
			},

			uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
			campaigncall.StatusProgressing,
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:18.995000",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("62a16e1c-b500-11ec-82e0-a31b4abba9ba"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:18.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("6275f0ac-b500-11ec-b356-b3eb98e5d2cb"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:18.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
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

			for _, p := range tt.campaigncalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
				if err := h.CampaigncallCreate(ctx, p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CampaigncallListByCampaignIDAndStatus(ctx, tt.campaignID, tt.status, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_CampaigncallListOngoingByCampaignID(t *testing.T) {
	tests := []struct {
		name          string
		campaigncalls []*campaigncall.Campaigncall

		campaignID uuid.UUID
		token      string
		limit      uint64

		responseCurTime string
		expectRes       []*campaigncall.Campaigncall
	}{
		{
			"1 item",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("32d6f71e-c447-11ec-b8d8-ff74cee56931"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("3388b6fc-c447-11ec-863e-67f3cc218843"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
			},

			uuid.FromStringOrNil("3388b6fc-c447-11ec-863e-67f3cc218843"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:17.995000",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("32d6f71e-c447-11ec-b8d8-ff74cee56931"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("3388b6fc-c447-11ec-863e-67f3cc218843"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:17.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
				},
			},
		},
		{
			"2 items",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("33139c3c-c447-11ec-ba59-6312277c868c"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("336245d0-c447-11ec-a3da-87ac346b98ad"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("333cbefa-c447-11ec-b7dc-573a743e072e"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("336245d0-c447-11ec-a3da-87ac346b98ad"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusDialing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
				},
			},

			uuid.FromStringOrNil("336245d0-c447-11ec-a3da-87ac346b98ad"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:18.995000",
			[]*campaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("333cbefa-c447-11ec-b7dc-573a743e072e"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("336245d0-c447-11ec-a3da-87ac346b98ad"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusDialing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:18.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("33139c3c-c447-11ec-ba59-6312277c868c"),
						CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					},
					CampaignID:      uuid.FromStringOrNil("336245d0-c447-11ec-a3da-87ac346b98ad"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:18.995000",
					TMUpdate:         DefaultTimeStamp,
					TMDelete:         DefaultTimeStamp,
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

			for _, p := range tt.campaigncalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
				if err := h.CampaigncallCreate(ctx, p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CampaigncallListOngoingByCampaignID(ctx, tt.campaignID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_CampaigncallUpdateStatus(t *testing.T) {
	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall

		status campaigncall.Status

		responseCurTime string
		expectRes       *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a408f39c-b501-11ec-8222-cfde83409646"),
					CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
				},
				CampaignID:      uuid.FromStringOrNil("a43632bc-b501-11ec-8c14-dbf345739172"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusProgressing,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
			},

			campaigncall.StatusDone,

			"2020-04-18 03:22:18.995000",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a408f39c-b501-11ec-8222-cfde83409646"),
					CustomerID: uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
				},
				CampaignID:      uuid.FromStringOrNil("a43632bc-b501-11ec-8c14-dbf345739172"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusDone,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:18.995000",
				TMUpdate:         "2020-04-18 03:22:18.995000",
				TMDelete:         DefaultTimeStamp,
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
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallCreate(ctx, tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallUpdateStatus(ctx, tt.campaigncall.ID, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaigncallGet(ctx, tt.campaigncall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any())
			res, err := h.CampaigncallGet(ctx, tt.campaigncall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaigncallUpdateStatusAndResult(t *testing.T) {
	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall

		status campaigncall.Status
		result campaigncall.Result

		responseCurTime string
		expectRes       *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bb76dc75-4b32-446c-98e3-bf08eaa85a54"),
					CustomerID: uuid.FromStringOrNil("b2ba23bf-fe6c-49e9-9121-fde7d725d9f2"),
				},
				CampaignID:      uuid.FromStringOrNil("a43632bc-b501-11ec-8c14-dbf345739172"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusProgressing,
				Result:          campaigncall.ResultNone,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
			},

			campaigncall.StatusDone,
			campaigncall.ResultSuccess,

			"2020-04-18 03:22:18.995000",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bb76dc75-4b32-446c-98e3-bf08eaa85a54"),
					CustomerID: uuid.FromStringOrNil("b2ba23bf-fe6c-49e9-9121-fde7d725d9f2"),
				},
				CampaignID:      uuid.FromStringOrNil("a43632bc-b501-11ec-8c14-dbf345739172"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusDone,
				Result:          campaigncall.ResultSuccess,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:18.995000",
				TMUpdate:         "2020-04-18 03:22:18.995000",
				TMDelete:         DefaultTimeStamp,
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
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallCreate(context.Background(), tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallUpdateStatusAndResult(ctx, tt.campaigncall.ID, tt.status, tt.result); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaigncallGet(gomock.Any(), tt.campaigncall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaigncallSet(gomock.Any(), gomock.Any())
			res, err := h.CampaigncallGet(ctx, tt.campaigncall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaigncallDelete(t *testing.T) {

	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1cc92874-b480-11ec-b7cf-4f5d95304498"),
					CustomerID: uuid.FromStringOrNil("1cf4905e-b480-11ec-8e27-038c9a252614"),
				},
				OutplanID: uuid.FromStringOrNil("ba29f006-b3ce-11ec-80d2-a71d2212a7d7"),
				OutdialID: uuid.FromStringOrNil("ba5c57c6-b3ce-11ec-b997-4b54d7754db6"),
				QueueID:   uuid.FromStringOrNil("ba91a87c-b3ce-11ec-993c-2f5317fef011"),
				TMCreate:  "2020-04-18 03:22:17.995000",
				TMUpdate:  "2020-04-18 03:22:17.995000",
				TMDelete:  DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any())
			if err := h.CampaigncallCreate(ctx, tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any())
			if err := h.CampaigncallDelete(ctx, tt.campaigncall.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaigncallGet(ctx, tt.campaigncall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any())
			res, err := h.CampaigncallGet(ctx, tt.campaigncall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}
		})
	}
}
