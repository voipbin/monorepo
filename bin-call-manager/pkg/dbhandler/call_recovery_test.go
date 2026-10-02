package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

// Test_CallSetChannelIDAndBridgeIDIfOwned checks the recovery switch is applied only to a progressing call that is
// still owned by the old channel (VOIP-1556).
func Test_CallSetChannelIDAndBridgeIDIfOwned(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		oldChannelID string
		newChannelID string
		bridgeID     string

		expectChanged   bool
		expectChannelID string
		expectBridgeID  string
	}{
		{
			name: "progressing call owned by the old channel",
			call: &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("3a4f8e2c-9f71-11f1-9b1e-0b6d2e4f1a01")},
				ChannelID: "old-3a4f8e2c",
				BridgeID:  "old-bridge-3a4f8e2c",
				Status:    call.StatusProgressing,
			},
			oldChannelID: "old-3a4f8e2c",
			newChannelID: "new-3a4f8e2c",
			bridgeID:     "new-bridge-3a4f8e2c",

			expectChanged:   true,
			expectChannelID: "new-3a4f8e2c",
			expectBridgeID:  "new-bridge-3a4f8e2c",
		},
		{
			name: "hung up call",
			call: &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("3a8b1d64-9f71-11f1-8c2f-4f1e3a6b2c02")},
				ChannelID: "old-3a8b1d64",
				BridgeID:  "old-bridge-3a8b1d64",
				Status:    call.StatusHangup,
			},
			oldChannelID: "old-3a8b1d64",
			newChannelID: "new-3a8b1d64",
			bridgeID:     "new-bridge-3a8b1d64",

			expectChanged:   false,
			expectChannelID: "old-3a8b1d64",
			expectBridgeID:  "old-bridge-3a8b1d64",
		},
		{
			name: "terminating call",
			call: &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("3ac7f0a2-9f71-11f1-a3d4-8e2b7c1d3e03")},
				ChannelID: "old-3ac7f0a2",
				BridgeID:  "old-bridge-3ac7f0a2",
				Status:    call.StatusTerminating,
			},
			oldChannelID: "old-3ac7f0a2",
			newChannelID: "new-3ac7f0a2",
			bridgeID:     "new-bridge-3ac7f0a2",

			expectChanged:   false,
			expectChannelID: "old-3ac7f0a2",
			expectBridgeID:  "old-bridge-3ac7f0a2",
		},
		{
			name: "call owned by another channel",
			call: &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("3b04c8e0-9f71-11f1-b5e6-2d9f4a8c5b04")},
				ChannelID: "other-3b04c8e0",
				BridgeID:  "other-bridge-3b04c8e0",
				Status:    call.StatusProgressing,
			},
			oldChannelID: "old-3b04c8e0",
			newChannelID: "new-3b04c8e0",
			bridgeID:     "new-bridge-3b04c8e0",

			expectChanged:   false,
			expectChannelID: "other-3b04c8e0",
			expectBridgeID:  "other-bridge-3b04c8e0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			curTime := testhelper.TimePtr("2026-10-02T01:00:00.000000Z")
			mockUtil.EXPECT().TimeNow().Return(curTime).AnyTimes()
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Fatalf("Could not create the call. err: %v", err)
			}

			if tt.expectChanged {
				mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			}
			changed, err := h.CallSetChannelIDAndBridgeIDIfOwned(ctx, tt.call.ID, tt.oldChannelID, tt.newChannelID, tt.bridgeID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if changed != tt.expectChanged {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChanged, changed)
			}

			res, err := h.CallGetFromDB(ctx, tt.call.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.ChannelID != tt.expectChannelID || res.BridgeID != tt.expectBridgeID {
				t.Errorf("Wrong match. expect: %s/%s, got: %s/%s", tt.expectChannelID, tt.expectBridgeID, res.ChannelID, res.BridgeID)
			}
			if res.Status != tt.call.Status {
				t.Errorf("Wrong match. expect status: %s, got: %s", tt.call.Status, res.Status)
			}
		})
	}
}

// Test_CallSetHangupIfChannel checks the hangup is recorded only while the call is owned by the given channel
// (VOIP-1556).
func Test_CallSetHangupIfChannel(t *testing.T) {

	tests := []struct {
		name string

		call      *call.Call
		channelID string

		expectWritten bool
		expectCall    call.Call
	}{
		{
			name: "owned by the channel",
			call: &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("5c2e7a90-9f71-11f1-8f3a-6b1c2d3e4f05")},
				ChannelID: "5c2e7a90-channel",
				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,
				TMCreate:  testhelper.TimePtr("2026-10-02T01:00:00.000000Z"),
			},
			channelID: "5c2e7a90-channel",

			expectWritten: true,
			expectCall: call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("5c2e7a90-9f71-11f1-8f3a-6b1c2d3e4f05")},
				ChannelID: "5c2e7a90-channel",

				ChainedCallIDs:   []uuid.UUID{},
				RecordingIDs:     []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusHangup,
				Direction: call.DirectionIncoming,

				HangupReason: call.HangupReasonNormal,
				HangupBy:     call.HangupByRemote,
				Data:         map[call.DataType]string{},
				Metadata:     map[string]interface{}{},
				Dialroutes:   nil,

				TMHangup: testhelper.TimePtr("2026-10-02T01:00:00.000000Z"),
				TMCreate: testhelper.TimePtr("2026-10-02T01:00:00.000000Z"),
				TMUpdate: testhelper.TimePtr("2026-10-02T01:00:00.000000Z"),
			},
		},
		{
			name: "moved to another channel",
			call: &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("5c6b2f3e-9f71-11f1-a4b5-1c2d3e4f5a06")},
				ChannelID: "5c6b2f3e-recovery",
				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,
				TMCreate:  testhelper.TimePtr("2026-10-02T01:00:00.000000Z"),
			},
			channelID: "5c6b2f3e-old",

			expectWritten: false,
			expectCall: call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("5c6b2f3e-9f71-11f1-a4b5-1c2d3e4f5a06")},
				ChannelID: "5c6b2f3e-recovery",

				ChainedCallIDs:   []uuid.UUID{},
				RecordingIDs:     []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,

				Data:       map[call.DataType]string{},
				Metadata:   map[string]interface{}{},
				Dialroutes: nil,

				TMCreate: testhelper.TimePtr("2026-10-02T01:00:00.000000Z"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(testhelper.TimePtr("2026-10-02T01:00:00.000000Z")).AnyTimes()
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Fatalf("Could not create the call. err: %v", err)
			}

			if tt.expectWritten {
				mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			}
			written, err := h.CallSetHangupIfChannel(ctx, tt.call.ID, tt.channelID, call.HangupReasonNormal, call.HangupByRemote)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if written != tt.expectWritten {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectWritten, written)
			}

			res, err := h.CallGetFromDB(ctx, tt.call.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			res.Dialroutes = nil
			if !reflect.DeepEqual(tt.expectCall, *res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, *res)
			}
		})
	}
}

// Test_CallRecoveryClaim checks the claim is passed through to the cache.
func Test_CallRecoveryClaim(t *testing.T) {

	tests := []struct {
		name string

		id  uuid.UUID
		ttl time.Duration

		responseClaimed bool
	}{
		{
			name:            "claimed",
			id:              uuid.FromStringOrNil("7d1e2f30-9f71-11f1-9a8b-5c6d7e8f9a07"),
			ttl:             180 * time.Second,
			responseClaimed: true,
		},
		{
			name:            "held by another",
			id:              uuid.FromStringOrNil("7d5a3b4c-9f71-11f1-8b9c-6d7e8f9a0b08"),
			ttl:             180 * time.Second,
			responseClaimed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				cache: mockCache,
			}
			ctx := context.Background()

			mockCache.EXPECT().CallRecoveryClaim(ctx, tt.id, tt.ttl).Return(tt.responseClaimed, nil)

			res, err := h.CallRecoveryClaim(ctx, tt.id, tt.ttl)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.responseClaimed {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseClaimed, res)
			}
		})
	}
}
