package dbhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
)

func Test_AgentCountByCustomerID(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID
		agents     []*agent.Agent
		expectRes  int
	}{
		{
			name:       "normal",
			customerID: uuid.FromStringOrNil("e1f20001-7fde-11ec-80e1-33e6bbba4dac"),
			agents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e1f20002-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("e1f20001-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "counttest1",
					PasswordHash: "hash",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e1f20003-4b42-11ec-afb2-3f23cd119aa6"),
						CustomerID: uuid.FromStringOrNil("e1f20001-7fde-11ec-80e1-33e6bbba4dac"),
					},
					Username:     "counttest2",
					PasswordHash: "hash",
				},
			},
			expectRes: 2,
		},
		{
			name:       "no agents",
			customerID: uuid.FromStringOrNil("e1f20004-7fde-11ec-80e1-33e6bbba4dac"),
			agents:     []*agent.Agent{},
			expectRes:  0,
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
			for _, a := range tt.agents {
				mockUtil.EXPECT().TimeNow().Return(testTime("2020-04-18T03:22:17.995000Z"))
				if err := h.AgentCreate(ctx, a); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.AgentCountByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentGetByUsername(t *testing.T) {
	tests := []struct {
		name      string
		agent     *agent.Agent
		username  string
		expectErr bool
	}{
		{
			name: "normal",
			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1a20001-cbb7-11ee-b58c-9b64857cf4b2"),
					CustomerID: uuid.FromStringOrNil("f1a20002-7fde-11ec-80e1-33e6bbba4dac"),
				},
				Username:     "testadditional@voipbin.net",
				PasswordHash: "hash",
			},
			username:  "testadditional@voipbin.net",
			expectErr: false,
		},
		{
			name: "not found",
			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1a20003-cbb7-11ee-b58c-9b64857cf4b2"),
					CustomerID: uuid.FromStringOrNil("f1a20002-7fde-11ec-80e1-33e6bbba4dac"),
				},
				Username:     "testadditional2@voipbin.net",
				PasswordHash: "hash",
			},
			username:  "notfound@voipbin.net",
			expectErr: true,
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

			mockUtil.EXPECT().TimeNow().Return(testTime("2020-04-18T03:22:17.995000Z"))
			mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return("2021-04-18T03:22:17.995000Z")
			res, err := h.AgentGetByUsername(ctx, tt.username)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error match. expect error: %v, got error: %v", tt.expectErr, err)
			}

			if !tt.expectErr && res.Username != tt.username {
				t.Errorf("Wrong username. expect: %v, got: %v", tt.username, res.Username)
			}
		})
	}
}

func Test_AgentSetBasicInfo(t *testing.T) {
	tests := []struct {
		name string

		id         uuid.UUID
		agentName  string
		detail     string
		ringMethod agent.RingMethod

		agent *agent.Agent

		responseCurTime *time.Time
		expectRes       *agent.Agent
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("d1a10001-4c6b-11ec-922d-27336e407864"),
			agentName:  "updated name",
			detail:     "updated detail",
			ringMethod: agent.RingMethodLinear,

			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10001-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username:     "test1",
				PasswordHash: "hash",
				Name:         "test1name",
				Detail:       "test1detail",
				RingMethod:   agent.RingMethodRingAll,
				Permission:   0,
				TagIDs:       []uuid.UUID{},
				Addresses:    []commonaddress.Address{},
			},

			responseCurTime: testTime("2020-04-18T03:22:17.995000Z"),
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10001-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username:     "test1",
				PasswordHash: "hash",
				Name:         "updated name",
				Detail:       "updated detail",
				RingMethod:   agent.RingMethodLinear,
				Permission:   0,
				TagIDs:       []uuid.UUID{},
				Addresses:    []commonaddress.Address{},
				TMCreate:     testTime("2020-04-18T03:22:17.995000Z"),
				TMUpdate:     testTime("2020-04-18T03:22:17.995000Z"),
				TMDelete:     nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			err := h.AgentSetBasicInfo(ctx, tt.id, tt.agentName, tt.detail, tt.ringMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			res, err := h.AgentGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if res.Name != tt.expectRes.Name {
				t.Errorf("Wrong name.\nexpect: %v\ngot: %v", tt.expectRes.Name, res.Name)
			}
			if res.Detail != tt.expectRes.Detail {
				t.Errorf("Wrong detail.\nexpect: %v\ngot: %v", tt.expectRes.Detail, res.Detail)
			}
			if res.RingMethod != tt.expectRes.RingMethod {
				t.Errorf("Wrong ring method.\nexpect: %v\ngot: %v", tt.expectRes.RingMethod, res.RingMethod)
			}
		})
	}
}

func Test_AgentSetPasswordHash(t *testing.T) {
	tests := []struct {
		name string

		id           uuid.UUID
		passwordHash string

		agent *agent.Agent

		responseCurTime *time.Time
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("d1a10002-4c6b-11ec-922d-27336e407864"),
			passwordHash: "new_hash",

			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10002-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username:     "test1",
				PasswordHash: "old_hash",
			},

			responseCurTime: testTime("2020-04-18T03:22:17.995000Z"),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			err := h.AgentSetPasswordHash(ctx, tt.id, tt.passwordHash)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			res, err := h.AgentGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if res.PasswordHash != tt.passwordHash {
				t.Errorf("Wrong password hash.\nexpect: %v\ngot: %v", tt.passwordHash, res.PasswordHash)
			}
		})
	}
}

func Test_AgentSetStatus(t *testing.T) {
	tests := []struct {
		name string

		id     uuid.UUID
		status agent.Status

		agent *agent.Agent

		responseCurTime *time.Time
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("d1a10003-4c6b-11ec-922d-27336e407864"),
			status: agent.StatusAvailable,

			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10003-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username: "test1",
				Status:   agent.StatusOffline,
			},

			responseCurTime: testTime("2020-04-18T03:22:17.995000Z"),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			err := h.AgentSetStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			res, err := h.AgentGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if res.Status != tt.status {
				t.Errorf("Wrong status.\nexpect: %v\ngot: %v", tt.status, res.Status)
			}
		})
	}
}

func Test_AgentSetPermission(t *testing.T) {
	tests := []struct {
		name string

		id         uuid.UUID
		permission agent.Permission

		agent *agent.Agent

		responseCurTime *time.Time
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("d1a10004-4c6b-11ec-922d-27336e407864"),
			permission: agent.PermissionCustomerAdmin,

			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10004-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username:   "test1",
				Permission: agent.PermissionNone,
			},

			responseCurTime: testTime("2020-04-18T03:22:17.995000Z"),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			err := h.AgentSetPermission(ctx, tt.id, tt.permission)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			res, err := h.AgentGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if res.Permission != tt.permission {
				t.Errorf("Wrong permission.\nexpect: %v\ngot: %v", tt.permission, res.Permission)
			}
		})
	}
}

func Test_AgentSetTagIDs(t *testing.T) {
	tests := []struct {
		name string

		id     uuid.UUID
		tagIDs []uuid.UUID

		agent *agent.Agent

		responseCurTime *time.Time
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("d1a10005-4c6b-11ec-922d-27336e407864"),
			tagIDs: []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},

			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10005-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username: "test1",
				TagIDs:   []uuid.UUID{},
			},

			responseCurTime: testTime("2020-04-18T03:22:17.995000Z"),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			err := h.AgentSetTagIDs(ctx, tt.id, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AgentGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			res, err := h.AgentGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if len(res.TagIDs) != len(tt.tagIDs) {
				t.Errorf("Wrong tag IDs length.\nexpect: %v\ngot: %v", len(tt.tagIDs), len(res.TagIDs))
			}
		})
	}
}

func Test_AgentUpdate(t *testing.T) {
	tests := []struct {
		name string

		id     uuid.UUID
		fields map[agent.Field]any

		agent *agent.Agent

		responseCurTime *time.Time
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("d1a10006-4c6b-11ec-922d-27336e407864"),
			fields: map[agent.Field]any{
				agent.FieldName: "updated name",
			},

			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10006-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username: "test1",
				Name:     "old name",
			},

			responseCurTime: testTime("2020-04-18T03:22:17.995000Z"),
		},
		{
			name: "empty fields",

			id:     uuid.FromStringOrNil("d1a10007-4c6b-11ec-922d-27336e407864"),
			fields: map[agent.Field]any{},

			agent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a10007-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("835498de-7fde-11ec-8bf4-0b4a81c8b61d"),
				},
				Username: "test1",
			},

			responseCurTime: testTime("2020-04-18T03:22:17.995000Z"),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			if err := h.AgentCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(tt.fields) > 0 {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().AgentSet(ctx, gomock.Any())
			}
			err := h.AgentUpdate(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
