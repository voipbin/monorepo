package callhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amteam "monorepo/bin-ai-manager/models/team"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

// covBExpectStartCallTypeFlowUntilCreate registers the mocks for the startCallTypeFlow path up to the call creation.
// The call creation fails and the created call record is stored in the returned pointer.
func covBExpectStartCallTypeFlowUntilCreate(ctx context.Context, m *covBMocks, cn *channel.Channel, custID uuid.UUID, flowID uuid.UUID) *call.Call {
	created := &call.Call{}

	m.util.EXPECT().UUIDCreate().Return(uuid.Nil).AnyTimes()
	m.req.EXPECT().CustomerV1CustomerGet(ctx, custID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
	m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, custID, gomock.Any(), gomock.Any(), 1).Return(true, nil)
	m.bridge.EXPECT().Start(ctx, cn.AsteriskID, gomock.Any(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{}, nil)
	m.bridge.EXPECT().ChannelJoin(ctx, gomock.Any(), cn.ID, "", false, false).Return(nil)
	m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, custID, gomock.Any()).Return(nil, nil)
	m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, custID, flowID, fmactiveflow.ReferenceTypeCall, gomock.Any(), uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).Return(&fmactiveflow.Activeflow{}, nil)
	m.db.EXPECT().CallCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *call.Call) error {
		*created = *c
		return fmt.Errorf("err")
	})
	m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder).Return(nil, nil)

	return created
}

func Test_startIncomingDomainTypeSIPDirect_destination_covB(t *testing.T) {
	custID := uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000001")
	flowID := uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000002")
	resID := uuid.FromStringOrNil("12345678-1234-1234-1234-123456789012")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0100"}
	cn := &channel.Channel{ID: "ch-1", AsteriskID: "ast-1"}

	telTarget, _ := commonaddress.NormalizeTarget(commonaddress.TypeTel, resID.String())

	t.Run("conference", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		d := &dmdirect.Direct{ResourceID: resID}

		m.req.EXPECT().ConferenceV1ConferenceGet(ctx, resID).Return(&cfconference.Conference{Identity: commonidentity.Identity{ID: resID, CustomerID: custID}}, nil)
		m.req.EXPECT().FlowV1FlowCreate(ctx, custID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}, nil)
		created := covBExpectStartCallTypeFlowUntilCreate(ctx, m, cn, custID, flowID)

		if err := h.startIncomingDomainTypeSIPDirectConference(ctx, cn, d, source); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		expect := commonaddress.Address{Type: commonaddress.TypeConference, Target: resID.String()}
		if !reflect.DeepEqual(created.Destination, expect) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, created.Destination)
		}
	})

	t.Run("flow", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		d := &dmdirect.Direct{ResourceID: resID}

		m.req.EXPECT().FlowV1FlowGet(ctx, resID).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID, CustomerID: custID}}, nil)
		created := covBExpectStartCallTypeFlowUntilCreate(ctx, m, cn, custID, flowID)

		if err := h.startIncomingDomainTypeSIPDirectFlow(ctx, cn, d, source); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		if telTarget == "" {
			t.Fatalf("Wrong match. expect: a non empty normalized target")
		}
		expect := commonaddress.Address{Type: commonaddress.TypeTel, Target: telTarget}
		if !reflect.DeepEqual(created.Destination, expect) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, created.Destination)
		}
	})
}

func Test_startIncomingDomainTypeSIPDirect_flowCreateError_covB(t *testing.T) {
	custID := uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000001")
	resID := uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000002")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0100"}
	cn := &channel.Channel{ID: "ch-1"}
	d := &dmdirect.Direct{ResourceID: resID}

	tests := []struct {
		name   string
		expect func(ctx context.Context, m *covBMocks)
		run    func(ctx context.Context, h *callHandler) error
	}{
		{
			name: "extension",
			expect: func(ctx context.Context, m *covBMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, resID).Return(&rmextension.Extension{Identity: commonidentity.Identity{ID: resID, CustomerID: custID}, Extension: "1001"}, nil)
			},
			run: func(ctx context.Context, h *callHandler) error {
				return h.startIncomingDomainTypeSIPDirectExtension(ctx, cn, d, source)
			},
		},
		{
			name: "ai",
			expect: func(ctx context.Context, m *covBMocks) {
				m.req.EXPECT().AIV1AIGet(ctx, resID).Return(&amai.AI{Identity: commonidentity.Identity{ID: resID, CustomerID: custID}}, nil)
			},
			run: func(ctx context.Context, h *callHandler) error {
				return h.startIncomingDomainTypeSIPDirectAI(ctx, cn, d, source)
			},
		},
		{
			name: "ai team",
			expect: func(ctx context.Context, m *covBMocks) {
				m.req.EXPECT().AIV1TeamGet(ctx, resID).Return(&amteam.Team{Identity: commonidentity.Identity{ID: resID, CustomerID: custID}}, nil)
			},
			run: func(ctx context.Context, h *callHandler) error {
				return h.startIncomingDomainTypeSIPDirectAITeam(ctx, cn, d, source)
			},
		},
		{
			name: "agent",
			expect: func(ctx context.Context, m *covBMocks) {
				m.req.EXPECT().AgentV1AgentGet(ctx, resID).Return(&amagent.Agent{Identity: commonidentity.Identity{ID: resID, CustomerID: custID}}, nil)
			},
			run: func(ctx context.Context, h *callHandler) error {
				return h.startIncomingDomainTypeSIPDirectAgent(ctx, cn, d, source)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			tt.expect(ctx, m)
			m.req.EXPECT().FlowV1FlowCreate(ctx, custID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("err"))
			m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder).Return(nil, nil)

			if err := tt.run(ctx, h); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
