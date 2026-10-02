package callhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// VOIP-1556 call recovery switch tests. The strict mock controller is the main assertion: every Asterisk and DB
// side effect not expected here fails the test.

var (
	tRecoveryCallID     = uuid.FromStringOrNil("2b9e4f10-9f73-11f1-8a2b-3c4d5e6f7a01")
	tRecoveryChannelID  = "2bdb4a4c-9f73-11f1-9b3c-4d5e6f7a8b02"
	tRecoveryOldChannel = "2c183f88-9f73-11f1-ac4d-5e6f7a8b9c03"
	tRecoveryBridgeID   = uuid.FromStringOrNil("2c5534c4-9f73-11f1-bd5e-6f7a8b9c0d04")
)

func recoveryChannel(state ari.ChannelState) *channel.Channel {
	return &channel.Channel{
		ID:         tRecoveryChannelID,
		AsteriskID: "3e:50:6b:43:bb:32",
		Type:       channel.TypeCall,
		State:      state,
		StasisData: map[channel.StasisDataType]string{
			channel.StasisDataTypeContextType:       string(channel.TypeCall),
			channel.StasisDataTypeContext:           string(channel.ContextCallRecovery),
			channel.StasisDataTypeCallID:            tRecoveryCallID.String(),
			channel.StasisDataTypeRecoveryChannelID: tRecoveryOldChannel,
		},
	}
}

type recoveryMocks struct {
	util    *utilhandler.MockUtilHandler
	db      *dbhandler.MockDBHandler
	channel *channelhandler.MockChannelHandler
	bridge  *bridgehandler.MockBridgeHandler
	req     *requesthandler.MockRequestHandler
	notify  *notifyhandler.MockNotifyHandler
}

func newRecoveryTestHandler(mc *gomock.Controller) (*callHandler, recoveryMocks) {
	m := recoveryMocks{
		util:    utilhandler.NewMockUtilHandler(mc),
		db:      dbhandler.NewMockDBHandler(mc),
		channel: channelhandler.NewMockChannelHandler(mc),
		bridge:  bridgehandler.NewMockBridgeHandler(mc),
		req:     requesthandler.NewMockRequestHandler(mc),
		notify:  notifyhandler.NewMockNotifyHandler(mc),
	}
	h := &callHandler{
		utilHandler:    m.util,
		db:             m.db,
		channelHandler: m.channel,
		bridgeHandler:  m.bridge,
		reqHandler:     m.req,
		notifyHandler:  m.notify,
	}
	return h, m
}

func terminatingSwitchedCall() *call.Call {
	c := switchedCall()
	c.Status = call.StatusTerminating
	return c
}

func switchedCall() *call.Call {
	return &call.Call{
		Identity:  commonidentity.Identity{ID: tRecoveryCallID},
		ChannelID: tRecoveryChannelID,
		BridgeID:  tRecoveryBridgeID.String(),
		Status:    call.StatusProgressing,
		Action:    fmaction.Action{Type: fmaction.TypeHangup},
	}
}

func Test_recoverySwitch_switched(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newRecoveryTestHandler(mc)
	ctx := context.Background()
	cn := recoveryChannel(ari.ChannelStateUp)

	gomock.InOrder(
		m.util.EXPECT().UUIDCreate().Return(tRecoveryBridgeID),
		m.db.EXPECT().CallSetChannelIDAndBridgeIDIfOwned(ctx, tRecoveryCallID, tRecoveryOldChannel, tRecoveryChannelID, tRecoveryBridgeID.String()).Return(true, nil),
		m.bridge.EXPECT().Start(ctx, cn.AsteriskID, tRecoveryBridgeID.String(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeVideoSFU}).Return(&bridge.Bridge{ID: tRecoveryBridgeID.String()}, nil),
		m.bridge.EXPECT().ChannelJoin(ctx, tRecoveryBridgeID.String(), tRecoveryChannelID, "", false, false).Return(nil),
		m.channel.EXPECT().HangingUpWithDelay(ctx, tRecoveryChannelID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(cn, nil),
		m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(switchedCall(), nil),
	)

	// the current action (hangup here) runs again on the recovery channel.
	m.db.EXPECT().CallGet(ctx, tRecoveryCallID).Return(switchedCall(), nil)
	m.db.EXPECT().CallSetStatus(ctx, tRecoveryCallID, call.StatusTerminating).Return(nil)
	m.db.EXPECT().CallGet(ctx, tRecoveryCallID).Return(terminatingSwitchedCall(), nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallTerminating, gomock.Any())
	m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(switchedCall(), nil)
	m.channel.EXPECT().HangingUp(ctx, tRecoveryChannelID, gomock.Any()).Return(&channel.Channel{ID: tRecoveryChannelID}, nil)

	if err := h.recoverySwitch(ctx, cn); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_recoverySwitch_notSwitched(t *testing.T) {

	tests := []struct {
		name string

		responseSwitchErr error
		responseCall      *call.Call
		responseGetErr    error

		expectHangup bool
		expectErr    bool
	}{
		{
			name:         "call hung up meanwhile",
			responseCall: &call.Call{Identity: commonidentity.Identity{ID: tRecoveryCallID}, ChannelID: tRecoveryOldChannel, Status: call.StatusHangup},
			expectHangup: true,
		},
		{
			name:         "call owned by another channel",
			responseCall: &call.Call{Identity: commonidentity.Identity{ID: tRecoveryCallID}, ChannelID: "other", Status: call.StatusProgressing},
			expectHangup: true,
		},
		{
			name:           "call not found",
			responseGetErr: dbhandler.ErrNotFound,
			expectHangup:   true,
		},
		{
			name:         "duplicate or redelivered Up after another delivery switched the call",
			responseCall: switchedCall(),
			expectHangup: false,
		},
		{
			name:              "update error that applied",
			responseSwitchErr: fmt.Errorf("connection reset"),
			responseCall:      switchedCall(),
			expectHangup:      false,
		},
		{
			name:              "update error that did not apply",
			responseSwitchErr: fmt.Errorf("connection reset"),
			responseCall:      &call.Call{Identity: commonidentity.Identity{ID: tRecoveryCallID}, ChannelID: tRecoveryOldChannel, Status: call.StatusTerminating},
			expectHangup:      true,
		},
		{
			name:           "re-read error",
			responseGetErr: fmt.Errorf("db down"),
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newRecoveryTestHandler(mc)
			ctx := context.Background()
			cn := recoveryChannel(ari.ChannelStateUp)

			m.util.EXPECT().UUIDCreate().Return(tRecoveryBridgeID)
			m.db.EXPECT().CallSetChannelIDAndBridgeIDIfOwned(ctx, tRecoveryCallID, tRecoveryOldChannel, tRecoveryChannelID, tRecoveryBridgeID.String()).Return(false, tt.responseSwitchErr)
			m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(tt.responseCall, tt.responseGetErr)
			if tt.expectHangup {
				// only the recovery channel itself; no bridge is created.
				m.channel.EXPECT().HangingUp(ctx, tRecoveryChannelID, ari.ChannelCauseNormalClearing).Return(cn, nil)
			}

			err := h.recoverySwitch(ctx, cn)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_recoverySwitch_bridgeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newRecoveryTestHandler(mc)
	ctx := context.Background()
	cn := recoveryChannel(ari.ChannelStateUp)

	m.util.EXPECT().UUIDCreate().Return(tRecoveryBridgeID)
	m.db.EXPECT().CallSetChannelIDAndBridgeIDIfOwned(ctx, tRecoveryCallID, tRecoveryOldChannel, tRecoveryChannelID, tRecoveryBridgeID.String()).Return(true, nil)
	m.bridge.EXPECT().Start(ctx, cn.AsteriskID, tRecoveryBridgeID.String(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{ID: tRecoveryBridgeID.String()}, nil)
	m.bridge.EXPECT().ChannelJoin(ctx, tRecoveryBridgeID.String(), tRecoveryChannelID, "", false, false).Return(fmt.Errorf("channel gone"))
	m.bridge.EXPECT().Destroy(ctx, tRecoveryBridgeID.String()).Return(nil)
	// after the commit, only the recovery channel is hung up; its destroy records the call's hangup.
	m.channel.EXPECT().HangingUp(ctx, tRecoveryChannelID, ari.ChannelCauseNormalClearing).Return(cn, nil)

	if err := h.recoverySwitch(ctx, cn); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_recoverySwitch_afterCommit(t *testing.T) {

	tests := []struct {
		name string

		responseCall   *call.Call
		responseGetErr error

		expectCallHangup bool
	}{
		{
			name:             "re-read error",
			responseGetErr:   fmt.Errorf("db down"),
			expectCallHangup: true,
		},
		{
			name:         "hung up before the action",
			responseCall: &call.Call{Identity: commonidentity.Identity{ID: tRecoveryCallID}, ChannelID: tRecoveryChannelID, Status: call.StatusTerminating},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newRecoveryTestHandler(mc)
			ctx := context.Background()
			cn := recoveryChannel(ari.ChannelStateUp)

			m.util.EXPECT().UUIDCreate().Return(tRecoveryBridgeID)
			m.db.EXPECT().CallSetChannelIDAndBridgeIDIfOwned(ctx, tRecoveryCallID, tRecoveryOldChannel, tRecoveryChannelID, tRecoveryBridgeID.String()).Return(true, nil)
			m.bridge.EXPECT().Start(ctx, cn.AsteriskID, tRecoveryBridgeID.String(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{ID: tRecoveryBridgeID.String()}, nil)
			m.bridge.EXPECT().ChannelJoin(ctx, tRecoveryBridgeID.String(), tRecoveryChannelID, "", false, false).Return(nil)
			m.channel.EXPECT().HangingUpWithDelay(ctx, tRecoveryChannelID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(cn, nil)
			m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(tt.responseCall, tt.responseGetErr)

			if tt.expectCallHangup {
				m.db.EXPECT().CallGet(ctx, tRecoveryCallID).Return(switchedCall(), nil)
				m.db.EXPECT().CallSetStatus(ctx, tRecoveryCallID, call.StatusTerminating).Return(nil)
				m.db.EXPECT().CallGet(ctx, tRecoveryCallID).Return(terminatingSwitchedCall(), nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallTerminating, gomock.Any())
				m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(switchedCall(), nil)
				m.channel.EXPECT().HangingUp(ctx, tRecoveryChannelID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{ID: tRecoveryChannelID}, nil)
			}

			if err := h.recoverySwitch(ctx, cn); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startContextCallRecovery(t *testing.T) {

	tests := []struct {
		name string

		responseDialErr error
		expectHangup    bool
	}{
		{
			name: "dial only",
		},
		{
			name:            "dial error",
			responseDialErr: fmt.Errorf("dial failed"),
			expectHangup:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newRecoveryTestHandler(mc)
			ctx := context.Background()
			cn := recoveryChannel(ari.ChannelStateDown)

			// no bridge, no call update, no action before the remote answered.
			m.channel.EXPECT().Dial(ctx, tRecoveryChannelID, "", defaultDialTimeout).Return(tt.responseDialErr)
			if tt.expectHangup {
				m.channel.EXPECT().HangingUp(ctx, tRecoveryChannelID, ari.ChannelCauseNormalClearing).Return(cn, nil)
			}

			if err := h.startContextCallRecovery(ctx, cn); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_ARIChannelStateChange_recovery checks a recovery channel's state changes never update the call; only Up
// starts the switch.
func Test_ARIChannelStateChange_recovery(t *testing.T) {

	tests := []struct {
		name  string
		state ari.ChannelState

		expectSwitch bool
	}{
		{name: "ringing", state: ari.ChannelStateRinging},
		{name: "ring", state: ari.ChannelStateRing},
		{name: "up", state: ari.ChannelStateUp, expectSwitch: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newRecoveryTestHandler(mc)
			ctx := context.Background()
			cn := recoveryChannel(tt.state)

			if tt.expectSwitch {
				m.util.EXPECT().UUIDCreate().Return(tRecoveryBridgeID)
				m.db.EXPECT().CallSetChannelIDAndBridgeIDIfOwned(ctx, tRecoveryCallID, tRecoveryOldChannel, tRecoveryChannelID, tRecoveryBridgeID.String()).Return(false, nil)
				m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(switchedCall(), nil)
			}

			if err := h.ARIChannelStateChange(ctx, cn); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_Hangup_movedCall checks the old channel's destroy does not hang up a progressing call that was switched to
// a recovery channel after the destroy read it: no webhook, no activeflow stop, no follow-ups.
func Test_Hangup_movedCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newRecoveryTestHandler(mc)
	ctx := context.Background()

	oldChannel := &channel.Channel{
		ID:          tRecoveryOldChannel,
		HangupCause: ari.ChannelCauseNormalClearing,
	}
	readCall := &call.Call{
		Identity:  commonidentity.Identity{ID: tRecoveryCallID},
		ChannelID: tRecoveryOldChannel,
		BridgeID:  "old-bridge",
		Status:    call.StatusProgressing,
		Direction: call.DirectionIncoming,
	}

	m.db.EXPECT().CallGetByChannelID(ctx, tRecoveryOldChannel).Return(readCall, nil)
	m.bridge.EXPECT().Destroy(ctx, "old-bridge").Return(nil)
	m.db.EXPECT().CallSetHangupIfChannel(ctx, tRecoveryCallID, tRecoveryOldChannel, call.HangupReasonNormal, call.HangupByRemote).Return(false, nil)
	m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(switchedCall(), nil)

	res, err := h.Hangup(ctx, oldChannel)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, switchedCall()) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", switchedCall(), res)
	}
}

// Test_Hangup_notProgressing checks a call that is not progressing keeps the unconditional hangup write.
func Test_Hangup_notProgressing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newRecoveryTestHandler(mc)
	ctx := context.Background()

	cn := &channel.Channel{
		ID:          "4f2a1b3c-9f73-11f1-8e6f-7a8b9c0d1e05",
		HangupCause: ari.ChannelCauseNormalClearing,
	}
	readCall := &call.Call{
		Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("4f67107c-9f73-11f1-9f7a-8b9c0d1e2f06")},
		ChannelID: cn.ID,
		Status:    call.StatusTerminating,
		Direction: call.DirectionIncoming,
	}
	hungup := *readCall
	hungup.Status = call.StatusHangup

	m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(readCall, nil)
	m.bridge.EXPECT().Destroy(ctx, "").Return(nil)
	m.db.EXPECT().CallSetHangup(ctx, readCall.ID, call.HangupReasonNormal, call.HangupByLocal).Return(nil)
	m.db.EXPECT().CallGet(ctx, readCall.ID).Return(&hungup, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, hungup.CustomerID, call.EventTypeCallHangup, &hungup)
	m.req.EXPECT().FlowV1ActiveflowStop(ctx, readCall.ActiveflowID).Return(nil, nil)

	if _, err := h.Hangup(ctx, cn); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_hangingUpWithCause_channelMoved checks the hangup request ends the channel that owns the call after the status
// write, and keeps the return contract when that channel cannot be hung up.
func Test_hangingUpWithCause_channelMoved(t *testing.T) {

	tests := []struct {
		name string

		responseDBCall *call.Call
		responseDBErr  error
		responseHangup error

		expectChannelID string
	}{
		{
			name:            "switched to the recovery channel",
			responseDBCall:  switchedCall(),
			expectChannelID: tRecoveryChannelID,
		},
		{
			name:            "moved channel cannot be hung up",
			responseDBCall:  switchedCall(),
			responseHangup:  fmt.Errorf("channel not found"),
			expectChannelID: tRecoveryChannelID,
		},
		{
			name:            "db read error falls back",
			responseDBErr:   fmt.Errorf("db down"),
			expectChannelID: tRecoveryOldChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newRecoveryTestHandler(mc)
			ctx := context.Background()

			readCall := &call.Call{
				Identity:  commonidentity.Identity{ID: tRecoveryCallID},
				ChannelID: tRecoveryOldChannel,
				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,
			}
			terminating := *readCall
			terminating.Status = call.StatusTerminating

			m.db.EXPECT().CallGet(ctx, tRecoveryCallID).Return(readCall, nil)
			m.db.EXPECT().CallSetStatus(ctx, tRecoveryCallID, call.StatusTerminating).Return(nil)
			m.db.EXPECT().CallGet(ctx, tRecoveryCallID).Return(&terminating, nil)
			m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallTerminating, gomock.Any())
			m.db.EXPECT().CallGetFromDB(ctx, tRecoveryCallID).Return(tt.responseDBCall, tt.responseDBErr)
			m.channel.EXPECT().HangingUp(ctx, tt.expectChannelID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{ID: tt.expectChannelID}, tt.responseHangup)

			res, err := h.hangingUpWithCause(ctx, tRecoveryCallID, ari.ChannelCauseNormalClearing)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, &terminating) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", &terminating, res)
			}
		})
	}
}
