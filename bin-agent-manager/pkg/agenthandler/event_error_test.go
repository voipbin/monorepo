package agenthandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
)

func Test_EventGroupcallProgressing_skips_failures(t *testing.T) {
	goodID := uuid.FromStringOrNil("a1a1a1a1-6666-6666-6666-666666666666")
	badID := uuid.FromStringOrNil("a2a2a2a2-6666-6666-6666-666666666666")
	customerID := uuid.FromStringOrNil("c1c1c1c1-6666-6666-6666-666666666666")
	errBoom := fmt.Errorf("boom")

	goodAgent := &agent.Agent{Identity: commonidentity.Identity{ID: goodID, CustomerID: customerID}}

	// first registers the mock calls of the failing first destination and returns it; the second
	// destination is a healthy agent and must still be processed. Cases without a lookup ignore m and ctx.
	tests := []struct {
		name string

		first func(ctx context.Context, m *errMocks) commonaddress.Address
	}{
		{
			name: "destination that is not an agent",
			first: func(ctx context.Context, m *errMocks) commonaddress.Address {
				return commonaddress.Address{Type: commonaddress.TypeExtension, Target: "test-ext"}
			},
		},
		{
			name: "agent target that is not a uuid",
			first: func(ctx context.Context, m *errMocks) commonaddress.Address {
				return commonaddress.Address{Type: commonaddress.TypeAgent, Target: "not-a-uuid"}
			},
		},
		{
			name: "agent lookup failure",
			first: func(ctx context.Context, m *errMocks) commonaddress.Address {
				m.db.EXPECT().AgentGet(ctx, badID).Return(nil, errBoom)
				return commonaddress.Address{Type: commonaddress.TypeAgent, Target: badID.String()}
			},
		},
		{
			name: "status update failure",
			first: func(ctx context.Context, m *errMocks) commonaddress.Address {
				m.db.EXPECT().AgentGet(ctx, badID).Return(&agent.Agent{Identity: commonidentity.Identity{ID: badID}}, nil)
				m.db.EXPECT().AgentSetStatus(ctx, badID, agent.StatusBusy).Return(errBoom)
				return commonaddress.Address{Type: commonaddress.TypeAgent, Target: badID.String()}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrTestHandler(mc)
			ctx := context.Background()

			first := tt.first(ctx, m)

			// the second destination is a healthy agent: lookup, status update, re-read, webhook.
			m.db.EXPECT().AgentGet(ctx, goodID).Return(goodAgent, nil).Times(2)
			m.db.EXPECT().AgentSetStatus(ctx, goodID, agent.StatusBusy).Return(nil)
			m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, agent.EventTypeAgentStatusUpdated, goodAgent)

			gc := &cmgroupcall.Groupcall{
				Destinations: []commonaddress.Address{
					first,
					{Type: commonaddress.TypeAgent, Target: goodID.String()},
				},
			}

			if err := h.EventGroupcallProgressing(ctx, gc); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCustomerDeleted_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("82ed53fa-ccca-11ee-be19-17f582a54cf4")
	firstID := uuid.FromStringOrNil("e3722b4c-ccca-11ee-b18c-03025e4b324b")
	secondID := uuid.FromStringOrNil("e39bfb34-ccca-11ee-9c3e-2fba9dd3bf35")
	directID := uuid.FromStringOrNil("d1d1d1d1-7777-7777-7777-777777777777")
	errBoom := fmt.Errorf("boom")

	cu := &cmcustomer.Customer{ID: customerID}
	expectFilter := map[agent.Field]any{
		agent.FieldCustomerID: customerID,
		agent.FieldDeleted:    false,
	}

	t.Run("list failure is wrapped and returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newErrTestHandler(mc)
		ctx := context.Background()

		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01T00:00:00.000000Z")
		m.db.EXPECT().AgentList(ctx, uint64(1000), "2024-01-01T00:00:00.000000Z", expectFilter).Return(nil, errBoom)

		err := h.EventCustomerDeleted(ctx, cu)
		if err == nil || !strings.Contains(err.Error(), "could not get agents list") {
			t.Errorf("Wrong error. expect: could not get agents list, got: %v", err)
		}
	})

	t.Run("a failed deletion does not stop the remaining agents", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newErrTestHandler(mc)
		ctx := context.Background()

		first := &agent.Agent{Identity: commonidentity.Identity{ID: firstID, CustomerID: customerID}}
		second := &agent.Agent{Identity: commonidentity.Identity{ID: secondID, CustomerID: customerID}, DirectID: directID}

		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01T00:00:00.000000Z")
		m.db.EXPECT().AgentList(ctx, uint64(1000), "2024-01-01T00:00:00.000000Z", expectFilter).Return([]*agent.Agent{first, second}, nil)

		// the first agent fails at its first lookup, so it is never deleted.
		m.db.EXPECT().AgentGet(ctx, firstID).Return(nil, errBoom)

		// the second agent is deleted completely.
		m.db.EXPECT().AgentGet(ctx, secondID).Return(second, nil).Times(2)
		m.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(&dmdirect.Direct{}, nil)
		m.db.EXPECT().AgentDelete(ctx, secondID).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, agent.EventTypeAgentDeleted, second)

		if err := h.EventCustomerDeleted(ctx, cu); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func Test_EventCustomerCreated_create_failure(t *testing.T) {
	customerID := uuid.FromStringOrNil("9c0ea002-c8e4-11ef-bfbd-3316b71b50ac")
	agentID := uuid.FromStringOrNil("38979028-c8e5-11ef-ab04-9b5ea42ae2be")
	errBoom := fmt.Errorf("boom")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrTestHandler(mc)
	ctx := context.Background()

	cu := &cmcustomer.Customer{ID: customerID, Email: "test@voipbin.net"}

	// the idempotency lookup finds nothing, then the admin agent creation fails at the direct creation.
	m.db.EXPECT().AgentGetByUsername(ctx, cu.Email).Return(nil, fmt.Errorf("not found"))
	// the generated password is random, so only the hash cost is matched exactly.
	m.util.EXPECT().HashGenerate(gomock.Any(), defaultPasswordHashCost).Return("test-hash", nil)
	m.util.EXPECT().UUIDCreate().Return(agentID)
	m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, agentID).Return(nil, errBoom)

	err := h.EventCustomerCreated(ctx, cu, false)
	if err == nil || !strings.Contains(err.Error(), "could not create basic customer admin agent") {
		t.Errorf("Wrong error. expect: could not create basic customer admin agent, got: %v", err)
	}
}
