package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
)

func Test_AgentCreate(t *testing.T) {

	tests := []struct {
		name  string
		agent *agent.Agent

		responseCurTime string
		expectRes       *agent.Agent
	}{
		{
			name: "normal",
			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses:    []commonaddress.Address{},
				TMCreate:     "2020-04-18 03:22:17.995000",
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
		},
		{
			name: "have address",
			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0e2f3d1c-4b4e-11ec-9455-9f4517cb3460"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0e2f3d1c-4b4e-11ec-9455-9f4517cb3460"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			name: "have addresses",
			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("523b3a6a-4b4e-11ec-b8fc-03aa2e2902d4"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656522",
						Name:   "",
					},
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("523b3a6a-4b4e-11ec-b8fc-03aa2e2902d4"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
						Name:   "",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656522",
						Name:   "",
					},
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			name: "have tag",
			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("48436342-4b4f-11ec-9fcb-0be19dd3beda"),
					CustomerID: uuid.FromStringOrNil("33f9ca84-7fde-11ec-a186-9f2e8c3a62aa"),
				},
				Username:     "test4",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("48436342-4b4f-11ec-9fcb-0be19dd3beda"),
					CustomerID: uuid.FromStringOrNil("33f9ca84-7fde-11ec-a186-9f2e8c3a62aa"),
				},
				Username:     "test4",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
				Addresses:    []commonaddress.Address{},
				TMCreate:     "2020-04-18 03:22:17.995000",
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(gomock.Any(), tt.agent.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any())
			res, err := h.AgentGet(ctx, tt.agent.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent *agent.Agent

		responseCurTime string
		expectRes       *agent.Agent
	}{
		{
			name: "normal",
			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				},
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TagIDs:       []uuid.UUID{},
				Addresses:    []commonaddress.Address{},
				TMCreate:     "2020-04-18 03:22:17.995000",
				TMUpdate:     "2020-04-18 03:22:17.995000",
				TMDelete:     "2020-04-18 03:22:17.995000",
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentDelete(ctx, tt.agent.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(ctx, tt.agent.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			res, err := h.AgentGet(ctx, tt.agent.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentGets(t *testing.T) {

	tests := []struct {
		name   string
		agents []*agent.Agent

		size    uint64
		filters map[string]string

		responseCurTime string
		expectRes       []*agent.Agent
	}{
		{
			name: "normal",
			agents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("48788c16-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "test2",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
						CustomerID: uuid.FromStringOrNil("48788c16-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "test3",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				},
			},

			size: 2,
			filters: map[string]string{
				"customer_id": "48788c16-7fde-11ec-80e1-33e6bbba4dac",
				"deleted":     "false",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("48788c16-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "test2",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TagIDs:       []uuid.UUID{},
					Addresses:    []commonaddress.Address{},
					TMCreate:     "2020-04-18 03:22:17.995000",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
						CustomerID: uuid.FromStringOrNil("48788c16-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "test3",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TagIDs:       []uuid.UUID{},
					Addresses:    []commonaddress.Address{},
					TMCreate:     "2020-04-18 03:22:17.995000",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
			},
		},
		{
			name: "gets by username",
			agents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("04ab77b0-cbb7-11ee-b58c-9b64857cf4b2"),
						CustomerID: uuid.FromStringOrNil("48788c16-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "test3@test.com",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				},
			},

			size: 2,
			filters: map[string]string{
				"username": "test3@test.com",
				"deleted":  "false",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("04ab77b0-cbb7-11ee-b58c-9b64857cf4b2"),
						CustomerID: uuid.FromStringOrNil("48788c16-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "test3@test.com",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TagIDs:       []uuid.UUID{},
					Addresses:    []commonaddress.Address{},
					TMCreate:     "2020-04-18 03:22:17.995000",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
			},
		},
		{
			name:   "empty",
			agents: []*agent.Agent{},

			size: 2,
			filters: map[string]string{
				"username": "282b439e-eca7-11ee-9d38-637a094feef1@test.com",
				"deleted":  "false",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes:       []*agent.Agent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.agents {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				if err := h.AgentCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.AgentGets(ctx, tt.size, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentSetAddresses(t *testing.T) {
	tests := []struct {
		name string

		id        uuid.UUID
		addresses []commonaddress.Address

		agents []*agent.Agent

		responseCurTime string
		expectRes       *agent.Agent
	}{
		{
			name: "normal",
			agents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
						CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
					},
					Username:     "test1",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					Name:         "test1name",
					Detail:       "test1detail",
					RingMethod:   agent.RingMethodRingAll,
					Permission:   0,
					TagIDs:       []uuid.UUID{},
					Addresses:    []commonaddress.Address{},
				},
			},

			id: uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
			addresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username:     "test1",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:         "test1name",
				Detail:       "test1detail",
				RingMethod:   agent.RingMethodRingAll,
				Permission:   0,
				TagIDs:       []uuid.UUID{},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
					},
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, u := range tt.agents {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().AgentSet(ctx, gomock.Any())
				if err := h.AgentCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			err := h.AgentSetAddresses(ctx, tt.id, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			res, err := h.AgentGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
