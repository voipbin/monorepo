package callhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	rmprovider "monorepo/bin-route-manager/models/provider"
	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/outboundconfig"
)

// cov2bOutgoingStage selects how far the SIP outgoing call flow runs and which step fails.
type cov2bOutgoingStage struct {
	customerErr  error
	balanceValid bool
	balanceErr   error

	activeflowErr error
	agentErr      error
	createErr     error
	variablesErr  error
	chainedErr    error
	channelErr    error

	masterCallID uuid.UUID
	anonymous    string

	// checks the call passed to the database create
	checkCreate func(c *call.Call)
}

const (
	cov2bCustomerID = "9a1b2c3d-0000-4000-8000-00000000c2b1"
	cov2bCallID     = "9a1b2c3d-0000-4000-8000-00000000c2b2"
	cov2bChannelID  = "9a1b2c3d-0000-4000-8000-00000000c2b3"
	cov2bFlowID     = "9a1b2c3d-0000-4000-8000-00000000c2b4"
	cov2bMasterID   = "9a1b2c3d-0000-4000-8000-00000000c2b5"
)

// cov2bExpectOutgoingSIP registers the mock expectations of CreateCallOutgoing for a sip destination up to the failing step.
func cov2bExpectOutgoingSIP(t *testing.T, ctx context.Context, m *cov2bMocks, st cov2bOutgoingStage) {
	t.Helper()

	customerID := uuid.FromStringOrNil(cov2bCustomerID)
	callID := uuid.FromStringOrNil(cov2bCallID)

	if st.customerErr != nil {
		m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(nil, st.customerErr)
		return
	}
	m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
		ID:     customerID,
		Status: cucustomer.StatusActive,
	}, nil)

	m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(st.balanceValid, st.balanceErr)
	if !st.balanceValid {
		return
	}

	m.outbound.EXPECT().GetByCustomerID(ctx, customerID).Return(nil, nil)

	var af *fmactiveflow.Activeflow
	if st.activeflowErr != nil {
		m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, uuid.FromStringOrNil(cov2bFlowID), fmactiveflow.ReferenceTypeCall, callID, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, st.activeflowErr)
	} else {
		af = &fmactiveflow.Activeflow{ID: uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2b6")}
		m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, uuid.FromStringOrNil(cov2bFlowID), fmactiveflow.ReferenceTypeCall, callID, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(af, nil)
	}
	m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil(cov2bChannelID))

	if st.agentErr != nil {
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, customerID, gomock.Any()).Return(nil, st.agentErr)
	} else {
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, customerID, gomock.Any()).Return(&amagent.Agent{}, nil)
	}

	m.db.EXPECT().CallCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *call.Call) error {
		if st.checkCreate != nil {
			st.checkCreate(c)
		}
		return st.createErr
	})
	if st.createErr != nil {
		return
	}
	created := &call.Call{
		Identity:  commonidentity.Identity{ID: callID, CustomerID: customerID},
		ChannelID: cov2bChannelID,
		Direction: call.DirectionOutgoing,
		Type:      call.TypeFlow,
		Destination: commonaddress.Address{
			Type:   commonaddress.TypeSIP,
			Target: "bob@example.com",
		},
		Source: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"},
	}
	m.db.EXPECT().CallGet(ctx, callID).Return(created, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, call.EventTypeCallCreated, created)
	m.req.EXPECT().CallV1CallHealth(ctx, callID, defaultHealthDelay, 0).Return(nil)

	m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(st.variablesErr)
	if st.variablesErr != nil {
		return
	}

	if st.masterCallID != uuid.Nil {
		m.db.EXPECT().CallTXStart(ctx, st.masterCallID).Return(nil, nil, st.chainedErr)
	}

	m.channel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, cov2bChannelID, gomock.Any(), "pjsip/"+pjsipEndpointOutgoing+"/sip:bob@example.com", "", "", "", gomock.Any()).
		Return(&channel.Channel{ID: cov2bChannelID}, st.channelErr)
}

func Test_CreateCallOutgoing_cov2b_sip(t *testing.T) {
	tests := []struct {
		name string

		destination commonaddress.Address
		stage       cov2bOutgoingStage

		expectErr string
	}{
		{
			name:        "destination type is neither sip nor tel",
			destination: commonaddress.Address{Type: commonaddress.TypeAgent, Target: "agent-target"},
			expectErr:   "must be sip or tel",
		},
		{
			name:        "customer lookup fails",
			destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"},
			stage:       cov2bOutgoingStage{customerErr: fmt.Errorf("rpc failure")},
			expectErr:   "could not get customer info",
		},
		{
			name:        "balance check says no balance",
			destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"},
			stage:       cov2bOutgoingStage{balanceValid: false},
			expectErr:   "balance validation",
		},
		{
			name:        "call create fails",
			destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"},
			stage:       cov2bOutgoingStage{balanceValid: true, createErr: fmt.Errorf("db failure")},
			expectErr:   "db failure",
		},
		{
			name:        "set variables fails",
			destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"},
			stage:       cov2bOutgoingStage{balanceValid: true, variablesErr: fmt.Errorf("rpc failure")},
			expectErr:   "could not set the variable",
		},
		{
			name:        "channel creation fails",
			destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"},
			stage:       cov2bOutgoingStage{balanceValid: true, channelErr: fmt.Errorf("ari failure")},
			expectErr:   "ari failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			if tt.destination.Type == commonaddress.TypeSIP {
				cov2bExpectOutgoingSIP(t, ctx, m, tt.stage)
			}

			res, err := h.CreateCallOutgoing(ctx, uuid.FromStringOrNil(cov2bCallID), uuid.FromStringOrNil(cov2bCustomerID), uuid.FromStringOrNil(cov2bFlowID), uuid.Nil, uuid.Nil, uuid.Nil,
				commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"}, tt.destination, false, false, "", nil, nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong error. expect: %q, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_CreateCallOutgoing_cov2b_sipContinues(t *testing.T) {
	masterID := uuid.FromStringOrNil(cov2bMasterID)

	tests := []struct {
		name  string
		stage cov2bOutgoingStage

		expectAnonymous string
		expectOwnerType commonidentity.OwnerType
		expectNoFlow    bool
	}{
		{
			name:            "activeflow creation failure falls back to a dummy activeflow",
			stage:           cov2bOutgoingStage{balanceValid: true, activeflowErr: fmt.Errorf("rpc failure")},
			expectAnonymous: "false",
			expectOwnerType: commonidentity.OwnerTypeAgent,
			expectNoFlow:    true,
		},
		{
			name:            "anonymous yes is stored as true",
			stage:           cov2bOutgoingStage{balanceValid: true, anonymous: "yes"},
			expectAnonymous: "true",
			expectOwnerType: commonidentity.OwnerTypeAgent,
		},
		{
			name:            "owner lookup failure leaves the owner empty",
			stage:           cov2bOutgoingStage{balanceValid: true, agentErr: fmt.Errorf("rpc failure")},
			expectAnonymous: "false",
			expectOwnerType: commonidentity.OwnerTypeNone,
		},
		{
			name:            "chained call id failure does not fail the call",
			stage:           cov2bOutgoingStage{balanceValid: true, masterCallID: masterID, chainedErr: fmt.Errorf("db failure")},
			expectAnonymous: "false",
			expectOwnerType: commonidentity.OwnerTypeAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			st := tt.stage
			st.checkCreate = func(c *call.Call) {
				if got := c.Data[call.DataTypeAnonymous]; got != tt.expectAnonymous {
					t.Errorf("Wrong anonymous data. expect: %s, got: %s", tt.expectAnonymous, got)
				}
				if c.OwnerType != tt.expectOwnerType {
					t.Errorf("Wrong owner type. expect: %s, got: %s", tt.expectOwnerType, c.OwnerType)
				}
				if tt.expectNoFlow && c.ActiveflowID != uuid.Nil {
					t.Errorf("Expected the dummy activeflow id. got: %s", c.ActiveflowID)
				}
			}
			cov2bExpectOutgoingSIP(t, ctx, m, st)

			res, err := h.CreateCallOutgoing(ctx, uuid.FromStringOrNil(cov2bCallID), uuid.FromStringOrNil(cov2bCustomerID), uuid.FromStringOrNil(cov2bFlowID), uuid.Nil, st.masterCallID, uuid.Nil,
				commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"},
				commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"}, false, false, st.anonymous, nil, nil)
			if err != nil {
				t.Fatalf("Wrong error. got: %v", err)
			}
			if res == nil || res.ID != uuid.FromStringOrNil(cov2bCallID) {
				t.Errorf("Wrong result. got: %v", res)
			}
		})
	}
}

func Test_CreateCallOutgoing_cov2b_telNoValidSource(t *testing.T) {
	customerID := uuid.FromStringOrNil(cov2bCustomerID)
	callID := uuid.FromStringOrNil(cov2bCallID)
	flowID := uuid.FromStringOrNil(cov2bFlowID)
	activeflowID := uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2b6")

	tests := []struct {
		name    string
		afID    uuid.UUID
		stopErr error
	}{
		{name: "orphaned activeflow is stopped", afID: activeflowID},
		{name: "orphaned activeflow stop failure is only logged", afID: activeflowID, stopErr: fmt.Errorf("rpc failure")},
		{name: "dummy activeflow is not stopped", afID: uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
				ID:                         customerID,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeCall, "kr", 1).Return(true, nil)
			m.outbound.EXPECT().GetByCustomerID(ctx, customerID).Return(&outboundconfig.OutboundConfig{DestinationWhitelist: []string{"kr"}}, nil)
			m.req.EXPECT().RouteV1DialrouteList(ctx, gomock.Any(), gomock.Any()).Return([]rmroute.Route{{ID: uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2b7")}}, nil)
			if tt.afID != uuid.Nil {
				m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, flowID, fmactiveflow.ReferenceTypeCall, callID, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(&fmactiveflow.Activeflow{ID: tt.afID}, nil)
			} else {
				m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, flowID, fmactiveflow.ReferenceTypeCall, callID, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("rpc failure"))
			}
			m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil(cov2bChannelID))
			// the source is not in E.164 format and the customer has no default source number
			if tt.afID != uuid.Nil {
				m.req.EXPECT().FlowV1ActiveflowStop(ctx, tt.afID).Return(nil, tt.stopErr)
			}

			res, err := h.CreateCallOutgoing(ctx, callID, customerID, flowID, uuid.Nil, uuid.Nil, uuid.Nil,
				commonaddress.Address{Type: commonaddress.TypeTel, Target: "not-e164"},
				commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821055550001"}, false, false, "", nil, nil)
			if err == nil || !strings.Contains(err.Error(), "no valid source number") {
				t.Errorf("Wrong error. got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong result. got: %v", res)
			}
		})
	}
}

func Test_CreateCallOutgoing_cov2b_telDialrouteListError(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil(cov2bCustomerID)

	m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
		ID:                         customerID,
		Status:                     cucustomer.StatusActive,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
	}, nil)
	m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeCall, "kr", 1).Return(true, nil)
	m.outbound.EXPECT().GetByCustomerID(ctx, customerID).Return(&outboundconfig.OutboundConfig{DestinationWhitelist: []string{"kr"}}, nil)
	m.req.EXPECT().RouteV1DialrouteList(ctx, gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("rpc failure"))

	res, err := h.CreateCallOutgoing(ctx, uuid.FromStringOrNil(cov2bCallID), customerID, uuid.FromStringOrNil(cov2bFlowID), uuid.Nil, uuid.Nil, uuid.Nil,
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821055550002"},
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821055550001"}, false, false, "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "could not get the dialroutes") {
		t.Errorf("Wrong error. got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong result. got: %v", res)
	}
}

func Test_CreateCallsOutgoing_cov2b(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()

	// CreateCallsOutgoing passes uuid.Nil as the call id, so the first UUIDCreate is the call id.
	// (registered before the channel id expectation of the helper.)
	m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil(cov2bCallID))
	cov2bExpectOutgoingSIP(t, ctx, m, cov2bOutgoingStage{balanceValid: true})
	unsupported := commonaddress.Address{Type: commonaddress.TypeNone, Target: "x"}
	m.groupcall.EXPECT().IsGroupcallTypeAddress(gomock.Any()).DoAndReturn(func(a *commonaddress.Address) bool {
		if a.Type != unsupported.Type {
			t.Errorf("Wrong address type checked. got: %s", a.Type)
		}
		return false
	})

	calls, groupcalls, err := h.CreateCallsOutgoing(ctx, uuid.FromStringOrNil(cov2bCustomerID), uuid.FromStringOrNil(cov2bFlowID), uuid.Nil,
		commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"},
		[]commonaddress.Address{{Type: commonaddress.TypeSIP, Target: "bob@example.com"}, unsupported},
		false, false, "", nil, nil)
	if err != nil {
		t.Fatalf("Wrong error. got: %v", err)
	}
	if len(calls) != 1 || calls[0].ID != uuid.FromStringOrNil(cov2bCallID) {
		t.Errorf("Wrong calls. got: %v", calls)
	}
	if len(groupcalls) != 0 {
		t.Errorf("Wrong groupcalls. got: %v", groupcalls)
	}
}

func Test_CreateCallsOutgoing_cov2b_allUnsupported(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()

	m.groupcall.EXPECT().IsGroupcallTypeAddress(gomock.Any()).Return(false)

	calls, groupcalls, err := h.CreateCallsOutgoing(ctx, uuid.FromStringOrNil(cov2bCustomerID), uuid.FromStringOrNil(cov2bFlowID), uuid.Nil,
		commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"},
		[]commonaddress.Address{{Type: commonaddress.TypeNone, Target: "x"}},
		false, false, "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "all destinations failed") {
		t.Errorf("Wrong error. got: %v", err)
	}
	if calls != nil || groupcalls != nil {
		t.Errorf("Wrong result. calls: %v, groupcalls: %v", calls, groupcalls)
	}
}

func Test_getDialURITel_cov2b_noDialrouteLeft(t *testing.T) {
	h, _ := newCov2bHandler(t)

	c := &call.Call{
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821055550001"},
		DialrouteID: uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2b8"),
		Dialroutes:  []rmroute.Route{{ID: uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2b9")}},
	}
	res, err := h.getDialURITel(context.Background(), c)
	if err == nil || !strings.Contains(err.Error(), "no available dialroute left") || res != nil {
		t.Errorf("Wrong match. res: %v, err: %v", res, err)
	}
}

func Test_getDialURI_cov2b_sipWebsocketDirect(t *testing.T) {
	h, _ := newCov2bHandler(t)

	tests := []struct {
		name   string
		target string
		expect string
	}{
		{"ws target without scheme gets sip prefix", "bob@example.com;transport=ws", "pjsip/" + pjsipEndpointOutgoingDirect + "/sip:bob@example.com;transport=ws"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &call.Call{Destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: tt.target}}
			res, err := h.getDialURI(context.Background(), c)
			if err != nil {
				t.Fatalf("Wrong error. got: %v", err)
			}
			if res.URI != tt.expect {
				t.Errorf("Wrong URI. expect: %s, got: %s", tt.expect, res.URI)
			}
		})
	}
}

func Test_getGroupcallRingMethod_cov2b(t *testing.T) {
	agentID := uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2ba")

	tests := []struct {
		name        string
		destination commonaddress.Address
		agentErr    error

		expectErr bool
	}{
		{
			name:        "agent lookup fails",
			destination: commonaddress.Address{Type: commonaddress.TypeAgent, Target: agentID.String()},
			agentErr:    fmt.Errorf("rpc failure"),
			expectErr:   true,
		},
		{
			name:        "non-agent destination defaults to ring all",
			destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			if tt.agentErr != nil {
				m.req.EXPECT().AgentV1AgentGet(ctx, agentID).Return(nil, tt.agentErr)
			}

			res, err := h.getGroupcallRingMethod(ctx, tt.destination)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				return
			}
			if string(res) != "ring_all" {
				t.Errorf("Wrong ring method. got: %s", res)
			}
		})
	}
}

func Test_getDialroutes_cov2b(t *testing.T) {
	customerID := uuid.FromStringOrNil(cov2bCustomerID)
	validProvider := uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2bb")

	tests := []struct {
		name        string
		destination commonaddress.Address
		metadata    map[string]interface{}
		listErr     error
		expectList  bool
		expectIDs   []uuid.UUID

		expectErr bool
	}{
		{
			name:        "non-tel destination returns no routes",
			destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"},
		},
		{
			name:        "unparsable number",
			destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "not-a-number"},
			expectErr:   true,
		},
		{
			name:        "non-string provider entries are skipped",
			destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821055550001"},
			metadata:    map[string]interface{}{call.MetadataKeyRouteProviderIDs: []interface{}{123, validProvider.String()}},
			expectList:  true,
			expectIDs:   []uuid.UUID{validProvider},
		},
		{
			name:        "route list failure",
			destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821055550001"},
			listErr:     fmt.Errorf("rpc failure"),
			expectList:  true,
			expectErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			if tt.expectList {
				m.req.EXPECT().RouteV1DialrouteList(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, filters map[rmroute.Field]any, ids []uuid.UUID) ([]rmroute.Route, error) {
						if filters[rmroute.FieldTarget] != "+82" {
							t.Errorf("Wrong target filter. got: %v", filters[rmroute.FieldTarget])
						}
						if len(ids) != len(tt.expectIDs) {
							t.Errorf("Wrong provider ids. expect: %v, got: %v", tt.expectIDs, ids)
						}
						return []rmroute.Route{{ID: validProvider}}, tt.listErr
					})
			}

			res, err := h.getDialroutes(ctx, customerID, &tt.destination, tt.metadata)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error. expect error: %v, got: %v", tt.expectErr, err)
			}
			if err == nil && tt.destination.Type != commonaddress.TypeTel && len(res) != 0 {
				t.Errorf("Wrong routes. got: %v", res)
			}
		})
	}
}

func Test_createChannelOutgoing_cov2b(t *testing.T) {
	callID := uuid.FromStringOrNil(cov2bCallID)

	tests := []struct {
		name  string
		call  *call.Call
		setup func(ctx context.Context, m *cov2bMocks)

		expectErr string
	}{
		{
			name: "dial uri cannot be built",
			call: &call.Call{
				Identity:    commonidentity.Identity{ID: callID},
				Destination: commonaddress.Address{Type: commonaddress.TypeAgent, Target: "x"},
			},
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "unsupported address type",
		},
		{
			name: "anonymous tel call with an invalid source number",
			call: &call.Call{
				Identity:    commonidentity.Identity{ID: callID},
				Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "not-e164"},
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821055550001"},
				Data:        map[call.DataType]string{call.DataTypeAnonymous: "true"},
				DialrouteID: uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2bc"),
				Dialroutes:  []rmroute.Route{{ID: uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2bc"), ProviderID: uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2bd")}},
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RouteV1ProviderGet(ctx, uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2bd")).Return(&rmprovider.Provider{Hostname: "carrier.example.com"}, nil)
			},
			expectErr: "source target is invalid",
		},
		{
			name: "channel start fails",
			call: &call.Call{
				Identity:    commonidentity.Identity{ID: callID},
				ChannelID:   cov2bChannelID,
				Source:      commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"},
				Destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"},
			},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, cov2bChannelID, gomock.Any(), "pjsip/"+pjsipEndpointOutgoing+"/sip:bob@example.com", "", "", "", gomock.Any()).
					Return(nil, fmt.Errorf("ari failure"))
			},
			expectErr: "ari failure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			err := h.createChannelOutgoing(ctx, tt.call)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong error. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_createFailoverChannel_cov2b(t *testing.T) {
	callID := uuid.FromStringOrNil(cov2bCallID)
	route1 := uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2be")
	route2 := uuid.FromStringOrNil("9a1b2c3d-0000-4000-8000-00000000c2bf")
	newCall := func() *call.Call {
		return &call.Call{
			Identity:    commonidentity.Identity{ID: callID, CustomerID: uuid.FromStringOrNil(cov2bCustomerID)},
			DialrouteID: route1,
			Dialroutes:  []rmroute.Route{{ID: route1}, {ID: route2}},
		}
	}

	tests := []struct {
		name  string
		call  *call.Call
		setup func(ctx context.Context, m *cov2bMocks)

		expectErr string
	}{
		{
			name:      "no next dialroute",
			call:      &call.Call{Identity: commonidentity.Identity{ID: callID}, DialrouteID: route1, Dialroutes: []rmroute.Route{{ID: route1}}},
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "no more dialroute left",
		},
		{
			name: "call update for failover fails",
			call: newCall(),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil(cov2bChannelID))
				m.db.EXPECT().CallSetForRouteFailover(ctx, callID, cov2bChannelID, route2).Return(fmt.Errorf("db failure"))
			},
			expectErr: "db failure",
		},
		{
			name: "channel creation for failover fails",
			call: newCall(),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil(cov2bChannelID))
				m.db.EXPECT().CallSetForRouteFailover(ctx, callID, cov2bChannelID, route2).Return(nil)
				updated := &call.Call{
					Identity:    commonidentity.Identity{ID: callID, CustomerID: uuid.FromStringOrNil(cov2bCustomerID)},
					ChannelID:   cov2bChannelID,
					DialrouteID: route2,
					Destination: commonaddress.Address{Type: commonaddress.TypeAgent, Target: "x"},
				}
				m.db.EXPECT().CallGet(ctx, callID).Return(updated, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, updated.CustomerID, call.EventTypeCallUpdated, updated)
			},
			expectErr: "unsupported address type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			res, err := h.createFailoverChannel(ctx, tt.call)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong error. expect: %q, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong result. got: %v", res)
			}
		})
	}
}

func Test_CreateCallOutgoing_cov2b_inactiveCustomer(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil(cov2bCustomerID)

	m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{ID: customerID, Status: cucustomer.StatusDeleted}, nil)

	res, err := h.CreateCallOutgoing(ctx, uuid.FromStringOrNil(cov2bCallID), customerID, uuid.FromStringOrNil(cov2bFlowID), uuid.Nil, uuid.Nil, uuid.Nil,
		commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"},
		commonaddress.Address{Type: commonaddress.TypeSIP, Target: "bob@example.com"}, false, false, "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not active") || res != nil {
		t.Errorf("Wrong match. res: %v, err: %v", res, err)
	}
}
