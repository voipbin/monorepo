package callhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

func Test_recoveryRun_cov2b(t *testing.T) {
	callID := uuid.FromStringOrNil("8a1b2c3d-0000-4000-8000-00000000c2b1")
	channelID := "8a1b2c3d-0000-4000-8000-00000000c2b2"
	newChannelID := uuid.FromStringOrNil("8a1b2c3d-0000-4000-8000-00000000c2b3")

	activeCall := &call.Call{
		Identity:  commonidentity.Identity{ID: callID},
		Status:    call.StatusProgressing,
		Direction: call.DirectionIncoming,
	}
	validChannel := &channel.Channel{ID: channelID, Type: channel.TypeCall, SIPCallID: "sip-call-id-cov2b"}

	tests := []struct {
		name string

		ch    *channel.Channel
		setup func(ctx context.Context, m *cov2bMocks)

		expectErr string
	}{
		{
			name:      "nil channel",
			ch:        nil,
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "channel is nil",
		},
		{
			name:      "channel is not a call channel",
			ch:        &channel.Channel{ID: channelID, Type: channel.TypeConfbridge},
			setup:     func(ctx context.Context, m *cov2bMocks) {},
			expectErr: "channel type is not call",
		},
		{
			name: "call lookup by channel fails",
			ch:   validChannel,
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGetByChannelID(ctx, channelID).Return(nil, fmt.Errorf("db failure"))
			},
			expectErr: "could not get call by channel ID",
		},
		{
			name: "recovery detail lookup fails",
			ch:   validChannel,
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGetByChannelID(ctx, channelID).Return(activeCall, nil)
				m.db.EXPECT().CallRecoveryClaim(ctx, callID, recoveryClaimTTL).Return(true, nil)
				m.recovery.EXPECT().GetRecoveryDetail(ctx, validChannel.SIPCallID, asteriskRoleUAS).Return(nil, fmt.Errorf("homer failure"))
			},
			expectErr: "could not get recovery detail for channel",
		},
		{
			name: "recovery channel creation fails",
			ch:   validChannel,
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGetByChannelID(ctx, channelID).Return(activeCall, nil)
				m.db.EXPECT().CallRecoveryClaim(ctx, callID, recoveryClaimTTL).Return(true, nil)
				m.recovery.EXPECT().GetRecoveryDetail(ctx, validChannel.SIPCallID, asteriskRoleUAS).Return(&recoveryDetail{RequestURI: "sip:peer@example.com", CSeq: 0}, nil)
				m.util.EXPECT().UUIDCreate().Return(newChannelID)
				m.channel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, newChannelID.String(), gomock.Any(),
					"pjsip/"+pjsipEndpointOutgoing+"/sip:peer@example.com", "", "", "", gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _, _, _, _, _ string, vars map[string]string) (*channel.Channel, error) {
						if _, ok := vars[channelVariableRecoveryCSeq]; ok {
							t.Errorf("CSeq variable must not be set when the detail has no CSeq")
						}
						return nil, fmt.Errorf("ari failure")
					})
			},
			expectErr: "ari failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			err := h.recoveryRun(ctx, tt.ch)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong error. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// Test_RecoveryStart_cov2b_goroutineFailure checks the per-channel recovery goroutine started by RecoveryStart,
// including the path where recoveryRun fails.
func Test_RecoveryStart_cov2b_goroutineFailure(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()

	start := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	channelID := "8a1b2c3d-0000-4000-8000-00000000c2b4"

	m.util.EXPECT().TimeNowAdd(-(time.Hour * 24)).Return(&start)
	m.util.EXPECT().TimeNow().Return(&end)
	m.channel.EXPECT().GetChannelsForRecovery(ctx, "asterisk-cov2b", channel.TypeCall, &start, &end, defaultRecoveryChannelLimit).
		Return([]*channel.Channel{{ID: channelID, Type: channel.TypeCall}}, nil)

	done := make(chan struct{})
	m.db.EXPECT().CallGetByChannelID(gomock.Any(), channelID).DoAndReturn(func(_ context.Context, _ string) (*call.Call, error) {
		defer close(done)
		return nil, fmt.Errorf("db failure")
	})

	if err := h.RecoveryStart(ctx, "asterisk-cov2b"); err != nil {
		t.Fatalf("Wrong error. got: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Timed out waiting for the recovery goroutine")
	}
}

func Test_RecoveryStart_cov2b_channelsError(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()

	start := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	m.util.EXPECT().TimeNowAdd(-(time.Hour * 24)).Return(&start)
	m.util.EXPECT().TimeNow().Return(&end)
	m.channel.EXPECT().GetChannelsForRecovery(ctx, "asterisk-cov2b", channel.TypeCall, &start, &end, defaultRecoveryChannelLimit).
		Return(nil, fmt.Errorf("db failure"))

	err := h.RecoveryStart(ctx, "asterisk-cov2b")
	if err == nil || !strings.Contains(err.Error(), "could not get channels for recovery") {
		t.Errorf("Wrong error. got: %v", err)
	}
}
