package callhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconference "monorepo/bin-conference-manager/models/conference"
	rmcustomerdomain "monorepo/bin-registrar-manager/models/customerdomain"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

const (
	cov2bRegCustomerID = "bb1b2c3d-0000-4000-8000-00000000c2b1"
	cov2bRegOtherID    = "bb1b2c3d-0000-4000-8000-00000000c2b2"
	cov2bRegTargetID   = "bb1b2c3d-0000-4000-8000-00000000c2b3"
	cov2bRegChannelID  = "ch-cov2b-reg"
)

func Test_startIncomingDomainTypeRegistrarDestination_cov2b(t *testing.T) {
	customerID := uuid.FromStringOrNil(cov2bRegCustomerID)
	otherID := uuid.FromStringOrNil(cov2bRegOtherID)
	targetID := uuid.FromStringOrNil(cov2bRegTargetID)
	source := &commonaddress.Address{Type: commonaddress.TypeExtension, Target: "2001"}

	tests := []struct {
		name        string
		destination *commonaddress.Address
		run         func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error
		setup       func(ctx context.Context, m *cov2bMocks)

		expectCause ari.ChannelCause
	}{
		{
			name:        "agent lookup fails",
			destination: &commonaddress.Address{Type: commonaddress.TypeAgent, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeAgent(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().AgentV1AgentGet(ctx, targetID).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectCause: ari.ChannelCauseNoRouteDestination,
		},
		{
			name:        "agent of another customer",
			destination: &commonaddress.Address{Type: commonaddress.TypeAgent, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeAgent(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().AgentV1AgentGet(ctx, targetID).Return(&amagent.Agent{Identity: commonidentity.Identity{ID: targetID, CustomerID: otherID}}, nil)
			},
			expectCause: ari.ChannelCauseNoRouteDestination,
		},
		{
			name:        "agent temp flow creation fails",
			destination: &commonaddress.Address{Type: commonaddress.TypeAgent, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeAgent(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().AgentV1AgentGet(ctx, targetID).Return(&amagent.Agent{Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID}, Name: "agent A"}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectCause: ari.ChannelCauseNetworkOutOfOrder,
		},
		{
			name:        "conference lookup fails",
			destination: &commonaddress.Address{Type: commonaddress.TypeConference, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeConference(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, targetID).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectCause: ari.ChannelCauseNoRouteDestination,
		},
		{
			name:        "conference of another customer",
			destination: &commonaddress.Address{Type: commonaddress.TypeConference, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeConference(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, targetID).Return(&cfconference.Conference{Identity: commonidentity.Identity{ID: targetID, CustomerID: otherID}}, nil)
			},
			expectCause: ari.ChannelCauseNoRouteDestination,
		},
		{
			name:        "conference temp flow creation fails",
			destination: &commonaddress.Address{Type: commonaddress.TypeConference, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeConference(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, targetID).Return(&cfconference.Conference{Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID}}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, gomock.Any(), "conference incoming handle", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectCause: ari.ChannelCauseNetworkOutOfOrder,
		},
		{
			name:        "tel temp flow creation fails",
			destination: &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002"},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeTel(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectCause: ari.ChannelCauseNetworkOutOfOrder,
		},
		{
			name:        "extension destination that cannot be resolved",
			destination: &commonaddress.Address{Type: commonaddress.TypeExtension, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeExtension(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, targetID).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectCause: ari.ChannelCauseNoRouteDestination,
		},
		{
			name:        "extension temp flow creation fails",
			destination: &commonaddress.Address{Type: commonaddress.TypeExtension, Target: targetID.String()},
			run: func(h *callHandler, ctx context.Context, cn *channel.Channel, d *commonaddress.Address) error {
				return h.startIncomingDomainTypeRegistrarDestinationTypeExtension(ctx, cn, customerID, source, d)
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, targetID).Return(&rmextension.Extension{Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID}, Extension: "2002"}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectCause: ari.ChannelCauseNetworkOutOfOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			cn := &channel.Channel{ID: cov2bRegChannelID}

			tt.setup(ctx, m)
			m.channel.EXPECT().HangingUp(ctx, cov2bRegChannelID, tt.expectCause).Return(nil, nil)

			if err := tt.run(h, ctx, cn, tt.destination); err != nil {
				t.Errorf("Wrong error. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeRegistrar_cov2b_unsupportedDestination(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil(cov2bRegCustomerID)
	cn := &channel.Channel{ID: cov2bRegChannelID, StasisData: map[channel.StasisDataType]string{channel.StasisDataTypeDomain: "abcd.reg.example.invalid"}}

	m.req.EXPECT().RegistrarV1CustomerDomainGetByRealm(ctx, "abcd.reg.example.invalid").Return(&rmcustomerdomain.CustomerDomain{CustomerID: customerID}, nil)
	m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeExtension).Return(&commonaddress.Address{Type: commonaddress.TypeExtension, Target: "2001"})
	// the source extension cannot be resolved, so the raw source is kept
	m.req.EXPECT().RegistrarV1ExtensionList(ctx, "", uint64(1), gomock.Any()).Return(nil, fmt.Errorf("rpc failure"))
	m.channel.EXPECT().AddressGetDestinationWithoutSpecificType(cn).Return(&commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"})
	m.channel.EXPECT().HangingUp(ctx, cov2bRegChannelID, ari.ChannelCauseNoRouteDestination).Return(nil, nil)

	if err := h.startIncomingDomainTypeRegistrar(ctx, cn); err != nil {
		t.Errorf("Wrong error. expect: nil, got: %v", err)
	}
}

func Test_parseAddressTypeExtension_cov2b(t *testing.T) {
	customerID := uuid.FromStringOrNil(cov2bRegCustomerID)
	otherID := uuid.FromStringOrNil(cov2bRegOtherID)
	targetID := uuid.FromStringOrNil(cov2bRegTargetID)

	tests := []struct {
		name    string
		address *commonaddress.Address
		setup   func(ctx context.Context, m *cov2bMocks)

		expectErr string
	}{
		{
			name:    "extension id lookup fails",
			address: &commonaddress.Address{Type: commonaddress.TypeExtension, Target: targetID.String()},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, targetID).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectErr: "rpc failure",
		},
		{
			name:    "extension of another customer",
			address: &commonaddress.Address{Type: commonaddress.TypeExtension, Target: targetID.String()},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, targetID).Return(&rmextension.Extension{Identity: commonidentity.Identity{ID: targetID, CustomerID: otherID}}, nil)
			},
			expectErr: "extension has wrong customer id",
		},
		{
			name:    "extension list lookup fails",
			address: &commonaddress.Address{Type: commonaddress.TypeExtension, Target: "2001"},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RegistrarV1ExtensionList(ctx, "", uint64(1), gomock.Any()).Return(nil, fmt.Errorf("rpc failure"))
			},
			expectErr: "rpc failure",
		},
		{
			name:    "extension number not found",
			address: &commonaddress.Address{Type: commonaddress.TypeExtension, Target: "2001"},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RegistrarV1ExtensionList(ctx, "", uint64(1), gomock.Any()).Return([]rmextension.Extension{}, nil)
			},
			expectErr: "no destination extension found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			res, err := h.parseAddressTypeExtension(ctx, customerID, tt.address)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) || res != nil {
				t.Errorf("Wrong match. res: %v, err: %v", res, err)
			}
		})
	}
}
