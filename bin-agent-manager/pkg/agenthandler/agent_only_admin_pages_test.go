package agenthandler

import (
	"context"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
)

func Test_isOnlyAdmin_pages(t *testing.T) {
	tests := []struct {
		name string

		targetPermission agent.Permission // CustomerAdmin when zero
		pages            [][]*agent.Agent // returned by successive AgentList calls

		expectRes bool
		expectErr bool
	}{
		{
			name: "other admin on the first page",

			pages: [][]*agent.Agent{
				{onlyAdminTarget(), onlyAdminOtherAdmin()},
			},
			expectRes: false,
		},
		{
			name: "other project super admin counts as another admin",

			pages: [][]*agent.Agent{
				append(onlyAdminAgents(2, agent.PermissionProjectSuperAdmin, onlyAdminBaseTime), onlyAdminTarget()),
			},
			expectRes: false,
		},
		{
			name: "project super admin target without another admin",

			targetPermission: agent.PermissionProjectSuperAdmin,
			pages: [][]*agent.Agent{
				{func() *agent.Agent {
					a := onlyAdminTarget()
					a.Permission = agent.PermissionProjectSuperAdmin
					return a
				}()},
			},
			expectRes: true,
		},
		{
			name: "only the target on a short page",

			pages: [][]*agent.Agent{
				{onlyAdminTarget()},
			},
			expectRes: true,
		},
		{
			name: "full page without admin then an empty page",

			pages: [][]*agent.Agent{
				append(onlyAdminAgents(testOnlyAdminPageSize-1, agent.PermissionCustomerAgent, onlyAdminBaseTime), onlyAdminTarget()),
				{},
			},
			expectRes: true,
		},
		{
			name: "target is the last row of the second page",

			pages: [][]*agent.Agent{
				onlyAdminAgents(testOnlyAdminPageSize, agent.PermissionCustomerAgent, onlyAdminBaseTime),
				{onlyAdminTarget()},
			},
			expectRes: true,
		},
		{
			name: "short page of 999 agents ends the search",

			pages: [][]*agent.Agent{
				append(onlyAdminAgents(testOnlyAdminPageSize-2, agent.PermissionCustomerAgent, onlyAdminBaseTime), onlyAdminTarget()),
			},
			expectRes: true,
		},
		{
			name: "next token is taken in UTC from a non UTC tm_create",

			pages: [][]*agent.Agent{
				onlyAdminAgents(testOnlyAdminPageSize, agent.PermissionCustomerAgent, onlyAdminBaseTime.In(time.FixedZone("KST", 9*3600))),
				{onlyAdminTarget(), onlyAdminOtherAdmin()},
			},
			expectRes: false,
		},
		{
			name: "full page whose last row has no tm_create",

			pages: [][]*agent.Agent{
				func() []*agent.Agent {
					l := onlyAdminAgents(testOnlyAdminPageSize, agent.PermissionCustomerAgent, onlyAdminBaseTime)
					l[len(l)-1].TMCreate = nil
					return l
				}(),
			},
			expectRes: false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newOnlyAdminHandler(mc)
			ctx := context.Background()

			target := onlyAdminTarget()
			if tt.targetPermission != 0 {
				target.Permission = tt.targetPermission
			}
			mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
			var calls []*gomock.Call
			for i, page := range tt.pages {
				token := ""
				if i > 0 {
					last := tt.pages[i-1][len(tt.pages[i-1])-1]
					token = last.TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
				}
				calls = append(calls, mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), token, onlyAdminFilters()).Return(page, nil))
			}
			ordered := make([]any, 0, len(calls))
			for _, c := range calls {
				ordered = append(ordered, c)
			}
			gomock.InOrder(ordered...)

			res, err := h.isOnlyAdmin(ctx, onlyAdminTargetID)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_isOnlyAdmin_notAdmin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newOnlyAdminHandler(mc)
	ctx := context.Background()

	notAdmin := onlyAdminTarget()
	notAdmin.Permission = agent.PermissionCustomerAgent
	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(notAdmin, nil)

	res, err := h.isOnlyAdmin(ctx, onlyAdminTargetID)
	if err != nil || res {
		t.Errorf("Wrong match. expect: false/nil, got: %v/%v", res, err)
	}
}
