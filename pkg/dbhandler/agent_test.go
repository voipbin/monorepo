package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/cachehandler"
)

func TestAgentCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name      string
		ag        *agent.Agent
		expectRes *agent.Agent
	}{
		{
			"test normal",
			&agent.Agent{
				ID:           uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
			&agent.Agent{
				ID:           uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses:    []cmaddress.Address{},
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
		},
		{
			"have address",
			&agent.Agent{
				ID:           uuid.FromStringOrNil("0e2f3d1c-4b4e-11ec-9455-9f4517cb3460"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Addresses: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&agent.Agent{
				ID:           uuid.FromStringOrNil("0e2f3d1c-4b4e-11ec-9455-9f4517cb3460"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
		{
			"have addresses",
			&agent.Agent{
				ID:           uuid.FromStringOrNil("523b3a6a-4b4e-11ec-b8fc-03aa2e2902d4"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Addresses: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656522",
						Name:   "",
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&agent.Agent{
				ID:           uuid.FromStringOrNil("523b3a6a-4b4e-11ec-b8fc-03aa2e2902d4"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656522",
						Name:   "",
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
		{
			"have tag",
			&agent.Agent{
				ID:           uuid.FromStringOrNil("48436342-4b4f-11ec-9fcb-0be19dd3beda"),
				UserID:       1,
				Username:     "test4",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
			&agent.Agent{
				ID:           uuid.FromStringOrNil("48436342-4b4f-11ec-9fcb-0be19dd3beda"),
				UserID:       1,
				Username:     "test4",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
				Addresses:    []cmaddress.Address{},
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
		},
		{
			"have webhook",
			&agent.Agent{
				ID:            uuid.FromStringOrNil("796ace40-79a0-11ec-9944-5bcaa66e32ac"),
				UserID:        1,
				Username:      "test5",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
				TagIDs:        []uuid.UUID{uuid.FromStringOrNil("7d4dba90-79a0-11ec-bd7b-4f343e208260")},
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
			&agent.Agent{
				ID:            uuid.FromStringOrNil("796ace40-79a0-11ec-9944-5bcaa66e32ac"),
				UserID:        1,
				Username:      "test5",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
				TagIDs:        []uuid.UUID{uuid.FromStringOrNil("7d4dba90-79a0-11ec-bd7b-4f343e208260")},
				Addresses:     []cmaddress.Address{},
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any())
			if err := h.AgentCreate(ctx, tt.ag); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(gomock.Any(), tt.ag.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any())
			res, err := h.AgentGet(ctx, tt.ag.ID)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestAgentDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name      string
		ag        *agent.Agent
		expectRes *agent.Agent
	}{
		{
			"test normal",
			&agent.Agent{
				ID:           uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
			&agent.Agent{
				ID:           uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses:    []cmaddress.Address{},
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().AgentGet(gomock.Any(), tt.ag.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).AnyTimes()

			if err := h.AgentCreate(ctx, tt.ag); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.AgentDelete(ctx, tt.ag.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.AgentGet(ctx, tt.ag.ID)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMDelete = res.TMDelete
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestAgentGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name      string
		userID    uint64
		data      []*agent.Agent
		size      uint64
		expectRes []*agent.Agent
	}{
		{
			"test normal",
			11,
			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					UserID:       11,
					Username:     "test2",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TMCreate:     "2020-04-18T03:22:17.995000",
				},
				{
					ID:           uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					UserID:       11,
					Username:     "test3",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TMCreate:     "2020-04-18T03:22:17.994000",
				},
			},
			2,
			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					UserID:       11,
					Username:     "test2",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TagIDs:       []uuid.UUID{},
					Addresses:    []cmaddress.Address{},
					TMCreate:     "2020-04-18T03:22:17.995000",
				},
				{
					ID:           uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					UserID:       11,
					Username:     "test3",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TagIDs:       []uuid.UUID{},
					Addresses:    []cmaddress.Address{},
					TMCreate:     "2020-04-18T03:22:17.994000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.data {
				if err := h.AgentCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.AgentGets(ctx, tt.userID, tt.size, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestAgentSetAddresses(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		id        uuid.UUID
		addresses []cmaddress.Address

		agents []*agent.Agent

		expectRes *agent.Agent
	}{
		{
			"test normal",

			uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
					UserID:       1,
					Username:     "test1",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					Name:         "test1name",
					Detail:       "test1detail",
					RingMethod:   agent.RingMethodRingAll,
					Permission:   0,
					TagIDs:       []uuid.UUID{},
					Addresses:    []cmaddress.Address{},
					TMCreate:     "",
					TMUpdate:     "",
					TMDelete:     "",
				},
			},

			&agent.Agent{
				ID:           uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
				UserID:       1,
				Username:     "test1",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:         "test1name",
				Detail:       "test1detail",
				RingMethod:   agent.RingMethodRingAll,
				Permission:   0,
				TagIDs:       []uuid.UUID{},
				Addresses: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656521",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, u := range tt.agents {
				mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any())
				if err := h.AgentCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any())
			err := h.AgentSetAddresses(ctx, tt.id, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any())
			res, err := h.AgentGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			res.TMCreate = ""
			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
