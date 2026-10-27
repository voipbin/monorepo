package callhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/externalmedia"
)

func Test_Start_cov2b(t *testing.T) {
	callID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b1")
	bridgeUUID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b2")
	extID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b3")
	const chID = "ch-cov2b"

	data := func(kv ...string) map[channel.StasisDataType]string {
		res := map[channel.StasisDataType]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			res[channel.StasisDataType(kv[i])] = kv[i+1]
		}
		return res
	}
	ctxKey := string(channel.StasisDataTypeContext)

	tests := []struct {
		name  string
		data  map[channel.StasisDataType]string
		setup func(ctx context.Context, m *cov2bMocks)

		expectErr string
	}{
		{
			name:      "no context in the stasis data",
			data:      data(),
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "no context found",
		},
		{
			name:      "unknown context",
			data:      data(ctxKey, "bogus"),
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "no route found for stasisstart",
		},
		{
			name: "service call default without a call",
			data: data(ctxKey, string(channel.ContextCallService)),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGetByChannelID(ctx, chID).Return(nil, fmt.Errorf("db failure"))
			},
			expectErr: "get call info",
		},
		{
			name: "recording start fails",
			data: data(ctxKey, string(channel.ContextRecording),
				string(channel.StasisDataTypeRecordingName), "rec", string(channel.StasisDataTypeRecordingDirection), "in",
				string(channel.StasisDataTypeRecordingFormat), "wav", string(channel.StasisDataTypeRecordingDuration), "30",
				string(channel.StasisDataTypeRecordingEndOfSilence), "5", string(channel.StasisDataTypeRecordingEndOfKey), "#"),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().Record(ctx, chID, "rec_in", "wav", 30, 5, false, "#", "fail").Return(fmt.Errorf("ari failure"))
			},
			expectErr: "could not start the recording",
		},
		{
			name: "external snoop join fails",
			data: data(ctxKey, string(channel.ContextExternalSnoop), string(channel.StasisDataTypeBridgeID), "br-1"),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.bridge.EXPECT().ChannelJoin(ctx, "br-1", chID, "", false, false).Return(fmt.Errorf("ari failure"))
			},
			expectErr: "could not set a call type for channel",
		},
		{
			name: "external media not found",
			data: data(ctxKey, string(channel.ContextExternalMedia), string(channel.StasisDataTypeExternalMediaID), extID.String()),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.external.EXPECT().Get(ctx, extID).Return(nil, fmt.Errorf("db failure"))
			},
			expectErr: "could not find external media",
		},
		{
			name: "external media join fails",
			data: data(ctxKey, string(channel.ContextExternalMedia), string(channel.StasisDataTypeExternalMediaID), extID.String()),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.external.EXPECT().Get(ctx, extID).Return(&externalmedia.ExternalMedia{ID: extID, BridgeID: "br-ext"}, nil)
				m.bridge.EXPECT().ChannelJoin(ctx, "br-ext", chID, "", false, false).Return(fmt.Errorf("ari failure"))
			},
			expectErr: "could not add the external media channel to the bridge",
		},
		{
			name: "join call bridge join fails",
			data: data(ctxKey, string(channel.ContextJoinCall), string(channel.StasisDataTypeBridgeID), "br-join"),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.bridge.EXPECT().ChannelJoin(ctx, "br-join", chID, "", false, false).Return(fmt.Errorf("ari failure"))
			},
			expectErr: "could not add the external snoop channel to the channel",
		},
		{
			name: "join call dial fails",
			data: data(ctxKey, string(channel.ContextJoinCall), string(channel.StasisDataTypeBridgeID), "br-join"),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.bridge.EXPECT().ChannelJoin(ctx, "br-join", chID, "", false, false).Return(nil)
				m.channel.EXPECT().Dial(ctx, chID, "", defaultDialTimeout).Return(fmt.Errorf("ari failure"))
			},
			expectErr: "could not dial the channel to the destination",
		},
		{
			name:      "application with an unknown name",
			data:      data(ctxKey, string(channel.ContextApplication), string(channel.StasisDataTypeApplicationName), "bogus"),
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "could not find correct event handler",
		},
		{
			name: "application amd continue fails",
			data: data(ctxKey, string(channel.ContextApplication), string(channel.StasisDataTypeApplicationName), applicationAMD),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().Continue(ctx, chID, serviceContextAMD, "", 0, "").Return(fmt.Errorf("ari failure"))
			},
			expectErr: "could not continue the channel",
		},
		{
			name: "call recovery dial failure hangs up the recovery channel",
			data: data(ctxKey, string(channel.ContextCallRecovery)),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().Dial(ctx, chID, "", defaultDialTimeout).Return(fmt.Errorf("ari failure"))
				m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNormalClearing).Return(nil, nil)
			},
		},
		{
			name: "incoming call with an unknown domain",
			data: data(ctxKey, string(channel.ContextCallIncoming), string(channel.StasisDataTypeDomain), "unknown.example.invalid"),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().HangingUpWithDelay(ctx, chID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(nil, fmt.Errorf("ari failure"))
			},
			expectErr: "no route found for stasisstart",
		},
		{
			name: "incoming trunk call without a trunk is rejected",
			data: data(ctxKey, string(channel.ContextCallIncoming), string(channel.StasisDataTypeDomain), "abc"+common.DomainTrunkSuffix),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().HangingUpWithDelay(ctx, chID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(nil, nil)
				m.channel.EXPECT().AddressGetSource(gomock.Any(), commonaddress.TypeTel).Return(&commonaddress.Address{})
				m.channel.EXPECT().AddressGetDestination(gomock.Any(), commonaddress.TypeTel).Return(&commonaddress.Address{})
				m.req.EXPECT().RegistrarV1TrunkGetByDomainName(ctx, "abc").Return(nil, fmt.Errorf("rpc failure"))
				m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNoRouteDestination).Return(nil, nil)
			},
		},
		{
			name: "incoming registrar call without a customer domain is rejected",
			data: data(ctxKey, string(channel.ContextCallIncoming), string(channel.StasisDataTypeDomain), "abc"+common.DomainRegistrarSuffix),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().HangingUpWithDelay(ctx, chID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(nil, nil)
				m.req.EXPECT().RegistrarV1CustomerDomainGetByRealm(ctx, "abc"+common.DomainRegistrarSuffix).Return(nil, fmt.Errorf("rpc failure"))
				m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNoRouteDestination).Return(nil, nil)
			},
		},
		{
			name:      "outgoing call without a call id",
			data:      data(ctxKey, string(channel.ContextCallOutgoing)),
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "could not get correct call_id",
		},
		{
			name: "outgoing call bridge creation fails",
			data: data(ctxKey, string(channel.ContextCallOutgoing), string(channel.StasisDataTypeCallID), callID.String()),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().HangingUpWithDelay(ctx, chID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(nil, fmt.Errorf("ari failure"))
				m.util.EXPECT().UUIDCreate().Return(bridgeUUID)
				m.bridge.EXPECT().Start(ctx, "ast-cov2b", bridgeUUID.String(), "reference_type=call,reference_id="+callID.String(), []bridge.Type{bridge.TypeMixing, bridge.TypeVideoSFU}).Return(nil, fmt.Errorf("ari failure"))
				m.db.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf("db failure"))
			},
			expectErr: "could not add the channel to the join bridge",
		},
		{
			name: "outgoing call bridge id save fails",
			data: data(ctxKey, string(channel.ContextCallOutgoing), string(channel.StasisDataTypeCallID), callID.String()),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().HangingUpWithDelay(ctx, chID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(nil, nil)
				m.util.EXPECT().UUIDCreate().Return(bridgeUUID)
				m.bridge.EXPECT().Start(ctx, "ast-cov2b", bridgeUUID.String(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{ID: bridgeUUID.String()}, nil)
				m.bridge.EXPECT().ChannelJoin(ctx, bridgeUUID.String(), chID, "", false, false).Return(nil)
				m.db.EXPECT().CallSetBridgeID(ctx, callID, bridgeUUID.String()).Return(fmt.Errorf("db failure"))
				m.db.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf("db failure"))
			},
			expectErr: "could not set call bridge id",
		},
		{
			name: "outgoing call dial fails",
			data: data(ctxKey, string(channel.ContextCallOutgoing), string(channel.StasisDataTypeCallID), callID.String()),
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.channel.EXPECT().HangingUpWithDelay(ctx, chID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(nil, nil)
				m.util.EXPECT().UUIDCreate().Return(bridgeUUID)
				m.bridge.EXPECT().Start(ctx, "ast-cov2b", bridgeUUID.String(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{ID: bridgeUUID.String()}, nil)
				m.bridge.EXPECT().ChannelJoin(ctx, bridgeUUID.String(), chID, "", false, false).Return(nil)
				m.db.EXPECT().CallSetBridgeID(ctx, callID, bridgeUUID.String()).Return(nil)
				m.channel.EXPECT().Dial(ctx, chID, chID, defaultDialTimeout).Return(fmt.Errorf("ari failure"))
				m.db.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf("db failure"))
			},
			expectErr: "could not dial the channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			cn := &channel.Channel{ID: chID, AsteriskID: "ast-cov2b", StasisData: tt.data}
			err := h.Start(ctx, cn)
			switch {
			case tt.expectErr == "" && err != nil:
				t.Errorf("Wrong error. expect: nil, got: %v", err)
			case tt.expectErr != "" && (err == nil || !strings.Contains(err.Error(), tt.expectErr)):
				t.Errorf("Wrong error. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_startIncomingDomainTypeConference_cov2b(t *testing.T) {
	confID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b4")
	customerID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b5")
	const chID = "ch-cov2b"

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *cov2bMocks)
	}{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, confID).Return(nil, fmt.Errorf("rpc failure"))
			},
		},
		{
			name: "temp flow creation fails",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, confID).Return(&cfconference.Conference{Identity: commonidentity.Identity{ID: confID, CustomerID: customerID}}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("rpc failure"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			cn := &channel.Channel{ID: chID}
			m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeSIP).Return(&commonaddress.Address{Type: commonaddress.TypeSIP, Target: "alice@example.com"})
			m.channel.EXPECT().AddressGetDestination(cn, commonaddress.TypeConference).Return(&commonaddress.Address{Type: commonaddress.TypeConference, Target: confID.String()})
			tt.setup(ctx, m)
			m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNoRouteDestination).Return(nil, nil)

			if err := h.startIncomingDomainTypeConference(ctx, cn); err != nil {
				t.Errorf("Wrong error. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypePSTN_cov2b(t *testing.T) {
	const chID = "ch-cov2b"
	destination := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002"}

	tests := []struct {
		name string
		nums []nmnumber.Number
		err  error
	}{
		{name: "number lookup fails", err: fmt.Errorf("rpc failure")},
		{name: "number not found", nums: []nmnumber.Number{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			cn := &channel.Channel{ID: chID}
			m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeTel).Return(&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"})
			m.channel.EXPECT().AddressGetDestination(cn, commonaddress.TypeTel).Return(destination)
			m.req.EXPECT().NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
				nmnumber.FieldNumber:  destination.Target,
				nmnumber.FieldDeleted: false,
			}).Return(tt.nums, tt.err)
			m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNoRouteDestination).Return(nil, nil)

			if err := h.startIncomingDomainTypePSTN(ctx, cn); err != nil {
				t.Errorf("Wrong error. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_startCallTypeFlow_cov2b(t *testing.T) {
	customerID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b6")
	flowID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b7")
	callID := uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b8")
	const chID = "ch-cov2b"

	tests := []struct {
		name string

		source   *commonaddress.Address
		customer *cucustomer.Customer
		setup    func(ctx context.Context, m *cov2bMocks)
	}{
		{
			name:     "agent source lookup failure is only logged and activeflow creation failure rejects the call",
			source:   &commonaddress.Address{Type: commonaddress.TypeAgent, Target: "aa1b2c3d-0000-4000-8000-00000000c2b9"},
			customer: &cucustomer.Customer{ID: customerID, Status: cucustomer.StatusActive},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().AgentV1AgentGet(ctx, uuid.FromStringOrNil("aa1b2c3d-0000-4000-8000-00000000c2b9")).Return(nil, fmt.Errorf("rpc failure"))
				m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, flowID, fmactiveflow.ReferenceTypeCall, callID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).Return(nil, fmt.Errorf("rpc failure"))
				m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNetworkOutOfOrder).Return(nil, nil)
			},
		},
		{
			name:     "customer with rtp debug marks the call and a variable failure hangs up the call",
			source:   &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"},
			customer: &cucustomer.Customer{ID: customerID, Status: cucustomer.StatusActive, Metadata: cucustomer.Metadata{RTPDebug: true}},
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, customerID, gomock.Any()).Return(nil, nil)
				m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, flowID, fmactiveflow.ReferenceTypeCall, callID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).Return(&fmactiveflow.Activeflow{}, nil)

				m.db.EXPECT().CallCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *call.Call) error {
					if rtp, _ := c.Metadata[call.MetadataKeyRTPDebug].(bool); !rtp {
						t.Errorf("Expected the rtp debug metadata on the created call. got: %v", c.Metadata)
					}
					return nil
				})
				created := &call.Call{
					Identity: commonidentity.Identity{ID: callID, CustomerID: customerID},
					Metadata: map[string]any{call.MetadataKeyRTPDebug: true},
				}
				m.db.EXPECT().CallGet(ctx, callID).Return(created, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, call.EventTypeCallCreated, created)
				m.req.EXPECT().CallV1CallHealth(ctx, callID, defaultHealthDelay, 0).Return(nil)
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(fmt.Errorf("rpc failure"))
				// the call is hung up with the server error cause. the hangup then fails on the call lookup.
				m.db.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf("db failure"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			cn := &channel.Channel{ID: chID, AsteriskID: "ast-cov2b"}
			destination := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002"}

			m.util.EXPECT().UUIDCreate().Return(callID).AnyTimes()
			m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(tt.customer, nil)
			m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, gomock.Any(), gomock.Any(), 1).Return(true, nil)
			m.bridge.EXPECT().Start(ctx, "ast-cov2b", callID.String(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{ID: callID.String()}, nil)
			m.bridge.EXPECT().ChannelJoin(ctx, callID.String(), chID, "", false, false).Return(nil)
			tt.setup(ctx, m)

			h.startCallTypeFlow(ctx, cn, customerID, flowID, tt.source, destination, nil)
		})
	}
}
