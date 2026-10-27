package channelhandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

func Test_ARIStasisStart_cov2_errors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		event *ari.StasisStart
		setup func(m *covMocks)
	}{
		{
			name: "set stasis info failed",
			event: &ari.StasisStart{
				Event:   ari.Event{Application: "voipbin"},
				Channel: ari.Channel{ID: cov2ChannelID, Name: "PJSIP/test-00000001"},
				Args:    ari.ArgsMap{"context": "call-in", "direction": "incoming"},
			},
			setup: func(m *covMocks) {
				m.db.EXPECT().ChannelSetStasisInfo(ctx, cov2ChannelID, channel.TypeCall, "voipbin",
					map[channel.StasisDataType]string{channel.StasisDataTypeContext: "call-in", channel.StasisDataTypeDirection: "incoming"},
					channel.DirectionIncoming).Return(errCov)
			},
		},
		{
			name: "update sip info failed for incoming call",
			event: &ari.StasisStart{
				Event:   ari.Event{Application: "voipbin"},
				Channel: ari.Channel{ID: cov2ChannelID, Name: "PJSIP/test-00000001"},
				Args:    ari.ArgsMap{"context": "call-in", "direction": "incoming"},
			},
			setup: func(m *covMocks) {
				m.db.EXPECT().ChannelSetStasisInfo(ctx, cov2ChannelID, channel.TypeCall, "voipbin", gomock.Any(), channel.DirectionIncoming).Return(nil)
				m.db.EXPECT().ChannelGet(ctx, cov2ChannelID).Return(&channel.Channel{
					ID:         cov2ChannelID,
					StasisData: map[channel.StasisDataType]string{channel.StasisDataTypeContext: "call-in"},
				}, nil)
				m.db.EXPECT().ChannelSetSIPCallID(ctx, cov2ChannelID, "").Return(errCov)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.setup(m)

			res, err := h.ARIStasisStart(ctx, tt.event)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, errCov) {
				t.Errorf("Wrong match. expect: wrapped errCov, got: %v", err)
			}
		})
	}
}

func Test_ARIChannelStateChange_cov2_errors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		state ari.ChannelState
		setup func(m *covMocks)

		expectIs error
	}{
		{
			name:  "unsupported state",
			state: ari.ChannelStateDown,
		},
		{
			name:  "outgoing channel got answer but sip info update failed",
			state: ari.ChannelStateUp,
			setup: func(m *covMocks) {
				m.db.EXPECT().ChannelSetStateAnswer(ctx, cov2ChannelID, ari.ChannelStateUp).Return(nil)
				m.db.EXPECT().ChannelGet(ctx, cov2ChannelID).Return(&channel.Channel{
					ID:         cov2ChannelID,
					AsteriskID: cov2AsteriskID,
					Direction:  channel.DirectionOutgoing,
					State:      ari.ChannelStateUp,
				}, nil)
				m.req.EXPECT().AstChannelVariableGet(ctx, cov2AsteriskID, cov2ChannelID, "CHANNEL(pjsip,call-id)").Return("", errCov)
				m.db.EXPECT().ChannelSetSIPCallID(ctx, cov2ChannelID, "").Return(errCov)
			},
			expectIs: errCov,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			res, err := h.ARIChannelStateChange(ctx, &ari.ChannelStateChange{Channel: ari.Channel{ID: cov2ChannelID, State: tt.state}})
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectIs != nil && !stderrors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

func Test_parseStasisData_cov2_audiosocket(t *testing.T) {
	h, _ := newCovHandler(t)

	e := &ari.StasisStart{
		Channel: ari.Channel{Name: "AudioSocket/127.0.0.1:9000-c2c00000"},
		Args:    ari.ArgsMap{"c2-external-media-id": ""},
	}

	expect := map[channel.StasisDataType]string{
		channel.StasisDataTypeExternalMediaID: "c2-external-media-id",
		channel.StasisDataTypeContext:         string(channel.ContextExternalMedia),
		channel.StasisDataTypeContextType:     string(channel.ContextTypeCall),
	}

	res := h.parseStasisData(e)
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_getChannelType_cov2_unknownContext(t *testing.T) {
	h, _ := newCovHandler(t)

	res := h.getChannelType(channel.Context("c2-unknown-context"))
	if res != channel.TypeNone {
		t.Errorf("Wrong match. expect: %v, got: %v", channel.TypeNone, res)
	}
}
