package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/cachehandler"
)

func Test_CampaigncallCreate(t *testing.T) {
	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				ID:              uuid.FromStringOrNil("5ed54e04-b4fe-11ec-bab7-1bbc3ac23720"),
				CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
				CampaignID:      uuid.FromStringOrNil("5f3bf276-b4fe-11ec-b032-47340d4fb85e"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusProgressing,
				Source: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().CampaigncallSet(ctx, tt.campaigncall).Return(nil)
			if err := h.CampaigncallCreate(context.Background(), tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaigncallGet(gomock.Any(), tt.campaigncall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaigncallSet(gomock.Any(), gomock.Any())
			res, err := h.CampaigncallGet(ctx, tt.campaigncall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.campaigncall, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.campaigncall, res)
			}
		})
	}
}

func Test_CampaigncallGetsByCampaignID(t *testing.T) {
	tests := []struct {
		name          string
		campaigncalls []*campaigncall.Campaigncall

		campaignID uuid.UUID
		token      string
		limit      uint64
	}{
		{
			"1 item",
			[]*campaigncall.Campaigncall{
				{
					ID:              uuid.FromStringOrNil("82b47bbe-b4ff-11ec-8b2a-4f8b8358d3ef"),
					CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					CampaignID:      uuid.FromStringOrNil("82dbd470-b4ff-11ec-b0f4-db46cf5d928e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:17.995000",
					TMUpdate:         "2020-04-18 03:22:17.995000",
				},
			},

			uuid.FromStringOrNil("82dbd470-b4ff-11ec-b0f4-db46cf5d928e"),
			"2022-04-18 03:22:17.995000",
			100,
		},
		{
			"2 items",
			[]*campaigncall.Campaigncall{
				{
					ID:              uuid.FromStringOrNil("da5fbd24-b4ff-11ec-9317-fb5fd7c5fd4f"),
					CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					CampaignID:      uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:19.995000",
					TMUpdate:         "2020-04-18 03:22:19.995000",
				},
				{
					ID:              uuid.FromStringOrNil("da92f482-b4ff-11ec-a879-e3cde17ee57d"),
					CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					CampaignID:      uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:18.995000",
					TMUpdate:         "2020-04-18 03:22:18.995000",
				},
			},

			uuid.FromStringOrNil("db0c5d9a-b4ff-11ec-95e4-ebaba2bdd23e"),
			"2022-04-18 03:22:17.995000",
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, p := range tt.campaigncalls {
				mockCache.EXPECT().CampaigncallSet(ctx, p).Return(nil)
				if err := h.CampaigncallCreate(context.Background(), p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CampaigncallGetsByCampaignID(ctx, tt.campaignID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.campaigncalls, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.campaigncalls, res)
			}
		})
	}
}

func Test_CampaigncallGetsByCampaignIDAndStatus(t *testing.T) {
	tests := []struct {
		name          string
		campaigncalls []*campaigncall.Campaigncall

		campaignID uuid.UUID
		status     campaigncall.Status
		token      string
		limit      uint64
	}{
		{
			"1 item",
			[]*campaigncall.Campaigncall{
				{
					ID:              uuid.FromStringOrNil("622497f2-b500-11ec-9f5d-4776190aaae1"),
					CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					CampaignID:      uuid.FromStringOrNil("624d111e-b500-11ec-a605-e3f84329977e"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:17.995000",
					TMUpdate:         "2020-04-18 03:22:17.995000",
				},
			},

			uuid.FromStringOrNil("624d111e-b500-11ec-a605-e3f84329977e"),
			campaigncall.StatusProgressing,
			"2022-04-18 03:22:17.995000",
			100,
		},
		{
			"2 items",
			[]*campaigncall.Campaigncall{
				{
					ID:              uuid.FromStringOrNil("6275f0ac-b500-11ec-b356-b3eb98e5d2cb"),
					CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					CampaignID:      uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:19.995000",
					TMUpdate:         "2020-04-18 03:22:19.995000",
				},
				{
					ID:              uuid.FromStringOrNil("62a16e1c-b500-11ec-82e0-a31b4abba9ba"),
					CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
					CampaignID:      uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
					OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
					OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
					OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
					QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
					ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
					ReferenceType:   campaigncall.ReferenceTypeCall,
					ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
					Status:          campaigncall.StatusProgressing,
					Source: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000001",
					},
					Destination: &cmaddress.Address{
						Type:   cmaddress.TypeTel,
						Target: "+821100000002",
					},
					DestinationIndex: 0,
					TryCount:         1,
					TMCreate:         "2020-04-18 03:22:18.995000",
					TMUpdate:         "2020-04-18 03:22:18.995000",
				},
			},

			uuid.FromStringOrNil("0b529838-b501-11ec-8e54-1b6991614914"),
			campaigncall.StatusProgressing,
			"2022-04-18 03:22:17.995000",
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, p := range tt.campaigncalls {
				mockCache.EXPECT().CampaigncallSet(ctx, p).Return(nil)
				if err := h.CampaigncallCreate(context.Background(), p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CampaigncallGetsByCampaignIDAndStatus(ctx, tt.campaignID, tt.status, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.campaigncalls, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.campaigncalls, res)
			}
		})
	}
}

func Test_CampaigncallUpdateStatus(t *testing.T) {
	tests := []struct {
		name         string
		campaigncall *campaigncall.Campaigncall

		status campaigncall.Status

		expectRes *campaigncall.Campaigncall
	}{
		{
			"normal",
			&campaigncall.Campaigncall{
				ID:              uuid.FromStringOrNil("a408f39c-b501-11ec-8222-cfde83409646"),
				CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
				CampaignID:      uuid.FromStringOrNil("a43632bc-b501-11ec-8c14-dbf345739172"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusProgressing,
				Source: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:18.995000",
				TMUpdate:         "2020-04-18 03:22:18.995000",
			},

			campaigncall.StatusDone,

			&campaigncall.Campaigncall{
				ID:              uuid.FromStringOrNil("a408f39c-b501-11ec-8222-cfde83409646"),
				CustomerID:      uuid.FromStringOrNil("5f07bfd8-b4fe-11ec-9444-4b5ae1d828a2"),
				CampaignID:      uuid.FromStringOrNil("a43632bc-b501-11ec-8c14-dbf345739172"),
				OutplanID:       uuid.FromStringOrNil("5f6f2cea-b4fe-11ec-9b36-eb1f55d879de"),
				OutdialID:       uuid.FromStringOrNil("5fa1b52a-b4fe-11ec-a2f7-f73dfe01ba97"),
				OutdialTargetID: uuid.FromStringOrNil("5fd06ca8-b4fe-11ec-b8b8-1fd108444321"),
				QueueID:         uuid.FromStringOrNil("6003b072-b4fe-11ec-afc8-df78feb301b9"),
				ActiveflowID:    uuid.FromStringOrNil("6038a2b4-b4fe-11ec-885f-170de0f5681f"),
				ReferenceType:   campaigncall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("606cda84-b4fe-11ec-8791-afef3711acc8"),
				Status:          campaigncall.StatusDone,
				Source: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         "2020-04-18 03:22:18.995000",
				TMUpdate:         "2020-04-18 03:22:18.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().CampaigncallSet(ctx, tt.campaigncall).Return(nil)
			if err := h.CampaigncallCreate(context.Background(), tt.campaigncall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaigncallSet(ctx, gomock.Any()).Return(nil)
			if err := h.CampaigncallUpdateStatus(ctx, tt.campaigncall.ID, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CampaigncallGet(gomock.Any(), tt.campaigncall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CampaigncallSet(gomock.Any(), gomock.Any())
			res, err := h.CampaigncallGet(ctx, tt.campaigncall.ID)
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
