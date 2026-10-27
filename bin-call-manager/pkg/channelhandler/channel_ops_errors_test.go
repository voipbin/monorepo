package channelhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

const (
	testOpsChannelID  = "a1b2c3d4-d13c-11ed-9a8b-7c6d5e4f3a2b"
	testOpsAsteriskID = "3e:50:6b:43:bb:30"
)

func newOpsHandler(mc *gomock.Controller) (*channelHandler, *dbhandler.MockDBHandler, *requesthandler.MockRequestHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	return &channelHandler{
		utilHandler:   utilhandler.NewMockUtilHandler(mc),
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
	}, mockDB, mockReq
}

// Test_channelOps_hungupChannel verifies that an operation on a channel that is already deleted
// fails without any request to the asterisk.
func Test_channelOps_hungupChannel(t *testing.T) {
	tests := []struct {
		name string
		run  func(h *channelHandler, ctx context.Context, id string) error
	}{
		{"hold on", func(h *channelHandler, ctx context.Context, id string) error { return h.HoldOn(ctx, id) }},
		{"hold off", func(h *channelHandler, ctx context.Context, id string) error { return h.HoldOff(ctx, id) }},
		{"moh on", func(h *channelHandler, ctx context.Context, id string) error { return h.MOHOn(ctx, id) }},
		{"moh off", func(h *channelHandler, ctx context.Context, id string) error { return h.MOHOff(ctx, id) }},
		{"silence on", func(h *channelHandler, ctx context.Context, id string) error { return h.SilenceOn(ctx, id) }},
		{"silence off", func(h *channelHandler, ctx context.Context, id string) error { return h.SilenceOff(ctx, id) }},
		{"mute on", func(h *channelHandler, ctx context.Context, id string) error {
			return h.MuteOn(ctx, id, channel.MuteDirectionBoth)
		}},
		{"mute off", func(h *channelHandler, ctx context.Context, id string) error {
			return h.MuteOff(ctx, id, channel.MuteDirectionBoth)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _ := newOpsHandler(mc)
			ctx := context.Background()

			tmDelete := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
			mockDB.EXPECT().ChannelGet(gomock.Any(), testOpsChannelID).Return(&channel.Channel{
				ID:         testOpsChannelID,
				AsteriskID: testOpsAsteriskID,
				TMDelete:   &tmDelete,
			}, nil)

			err := tt.run(h, ctx, testOpsChannelID)
			if err == nil || !strings.Contains(err.Error(), "hungup already") {
				t.Errorf("Wrong match. expect: hungup error, got: %v", err)
			}
		})
	}
}

// Test_channelOps_requestError verifies that a failing asterisk request is returned wrapped.
func Test_channelOps_requestError(t *testing.T) {
	errReq := fmt.Errorf("asterisk unavailable")

	tests := []struct {
		name string

		expect func(m *requesthandler.MockRequestHandler)
		run    func(h *channelHandler, ctx context.Context, id string) error

		expectMessage string
	}{
		{
			name: "hold on",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelHoldOn(gomock.Any(), testOpsAsteriskID, testOpsChannelID).Return(errReq)
			},
			run:           func(h *channelHandler, ctx context.Context, id string) error { return h.HoldOn(ctx, id) },
			expectMessage: "could not hold the channel",
		},
		{
			name: "hold off",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelHoldOff(gomock.Any(), testOpsAsteriskID, testOpsChannelID).Return(errReq)
			},
			run:           func(h *channelHandler, ctx context.Context, id string) error { return h.HoldOff(ctx, id) },
			expectMessage: "could not unhold the channel",
		},
		{
			name: "moh on",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelMusicOnHoldOn(gomock.Any(), testOpsAsteriskID, testOpsChannelID).Return(errReq)
			},
			run:           func(h *channelHandler, ctx context.Context, id string) error { return h.MOHOn(ctx, id) },
			expectMessage: "asterisk unavailable",
		},
		{
			name: "moh off",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelMusicOnHoldOff(gomock.Any(), testOpsAsteriskID, testOpsChannelID).Return(errReq)
			},
			run:           func(h *channelHandler, ctx context.Context, id string) error { return h.MOHOff(ctx, id) },
			expectMessage: "asterisk unavailable",
		},
		{
			name: "silence on",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelSilenceOn(gomock.Any(), testOpsAsteriskID, testOpsChannelID).Return(errReq)
			},
			run:           func(h *channelHandler, ctx context.Context, id string) error { return h.SilenceOn(ctx, id) },
			expectMessage: "asterisk unavailable",
		},
		{
			name: "silence off",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelSilenceOff(gomock.Any(), testOpsAsteriskID, testOpsChannelID).Return(errReq)
			},
			run:           func(h *channelHandler, ctx context.Context, id string) error { return h.SilenceOff(ctx, id) },
			expectMessage: "asterisk unavailable",
		},
		{
			name: "mute on",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelMuteOn(gomock.Any(), testOpsAsteriskID, testOpsChannelID, "in").Return(errReq)
			},
			run: func(h *channelHandler, ctx context.Context, id string) error {
				return h.MuteOn(ctx, id, channel.MuteDirectionIn)
			},
			expectMessage: "could not mute the channel",
		},
		{
			name: "mute off",
			expect: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AstChannelMuteOff(gomock.Any(), testOpsAsteriskID, testOpsChannelID, "out").Return(errReq)
			},
			run: func(h *channelHandler, ctx context.Context, id string) error {
				return h.MuteOff(ctx, id, channel.MuteDirectionOut)
			},
			expectMessage: "could not unmute the channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockReq := newOpsHandler(mc)

			// no ChannelSetMuteDirection expectation: a failed request must not touch the stored state
			mockDB.EXPECT().ChannelGet(gomock.Any(), testOpsChannelID).Return(&channel.Channel{
				ID:         testOpsChannelID,
				AsteriskID: testOpsAsteriskID,
			}, nil)
			tt.expect(mockReq)

			err := tt.run(h, context.Background(), testOpsChannelID)
			if err == nil || !strings.Contains(err.Error(), tt.expectMessage) || !strings.Contains(err.Error(), "asterisk unavailable") {
				t.Errorf("Wrong match. expect: error containing %q and the cause, got: %v", tt.expectMessage, err)
			}
		})
	}
}

// Test_MuteOff_remainingDirection verifies which direction stays muted after an unmute request.
func Test_MuteOff_remainingDirection(t *testing.T) {
	tests := []struct {
		name string

		current channel.MuteDirection
		unmute  channel.MuteDirection

		expectStored channel.MuteDirection
	}{
		{"unmute both clears everything", channel.MuteDirectionBoth, channel.MuteDirectionBoth, channel.MuteDirectionNone},
		{"unmute both while only in is muted", channel.MuteDirectionIn, channel.MuteDirectionBoth, channel.MuteDirectionNone},
		{"unmute in while both are muted keeps out", channel.MuteDirectionBoth, channel.MuteDirectionIn, channel.MuteDirectionOut},
		{"unmute out while both are muted keeps in", channel.MuteDirectionBoth, channel.MuteDirectionOut, channel.MuteDirectionIn},
		{"unmute in while out is muted keeps out", channel.MuteDirectionOut, channel.MuteDirectionIn, channel.MuteDirectionOut},
		{"unmute in while in is muted clears", channel.MuteDirectionIn, channel.MuteDirectionIn, channel.MuteDirectionNone},
		{"unmute in while nothing is muted stays none", channel.MuteDirectionNone, channel.MuteDirectionIn, channel.MuteDirectionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockReq := newOpsHandler(mc)
			ctx := context.Background()

			cn := &channel.Channel{ID: testOpsChannelID, AsteriskID: testOpsAsteriskID, MuteDirection: tt.current}
			mockDB.EXPECT().ChannelGet(gomock.Any(), testOpsChannelID).Return(cn, nil).Times(2)
			mockReq.EXPECT().AstChannelMuteOff(ctx, testOpsAsteriskID, testOpsChannelID, string(tt.unmute)).Return(nil)
			mockDB.EXPECT().ChannelSetMuteDirection(ctx, testOpsChannelID, tt.expectStored).Return(nil)

			if err := h.MuteOff(ctx, testOpsChannelID, tt.unmute); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
