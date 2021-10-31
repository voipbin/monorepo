package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
)

func TestConfbridgeCreateAndGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		confbridge       *confbridge.Confbridge
		expectConfbridge *confbridge.Confbridge
	}

	tests := []test{
		{
			"type conference",
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("fc07eed6-3301-11ec-8218-f37dfb357914"),
				ConferenceID:   uuid.FromStringOrNil("151e5f90-3302-11ec-acc0-afdac0cb7cb2"),
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("fc07eed6-3301-11ec-8218-f37dfb357914"),
				ConferenceID:   uuid.FromStringOrNil("151e5f90-3302-11ec-acc0-afdac0cb7cb2"),
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any()).Return(nil)
			if err := h.ConfbridgeCreate(context.Background(), tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeGet(gomock.Any(), tt.confbridge.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Any())
			res, err := h.ConfbridgeGet(ctx, tt.confbridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectConfbridge, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConfbridge, res)
			}
		})
	}
}

func TestConfbridgeGetByBridgeID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		confbridge       *confbridge.Confbridge
		expectConfbridge *confbridge.Confbridge
	}

	tests := []test{
		{
			"type conference",
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("bf738558-34ef-11ec-a927-6ba7cd3ff490"),
				ConferenceID:   uuid.FromStringOrNil("bf9c36e2-34ef-11ec-8fc4-7b7734a478f0"),
				BridgeID:       "bfc5a1e4-34ef-11ec-ad12-870a5704955c",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("bf738558-34ef-11ec-a927-6ba7cd3ff490"),
				ConferenceID:   uuid.FromStringOrNil("bf9c36e2-34ef-11ec-8fc4-7b7734a478f0"),
				BridgeID:       "bfc5a1e4-34ef-11ec-ad12-870a5704955c",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any()).Return(nil)
			if err := h.ConfbridgeCreate(context.Background(), tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ConfbridgeGetByBridgeID(ctx, tt.confbridge.BridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectConfbridge, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConfbridge, res)
			}
		})
	}
}

func TestConfbridgeSetRecordID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name             string
		confbridge       *confbridge.Confbridge
		recordID         uuid.UUID
		expectConfbridge *confbridge.Confbridge
	}

	tests := []test{
		{
			"test normal",
			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("75b1275e-3305-11ec-8dba-8bf525336b2b"),
			},
			uuid.FromStringOrNil("760b193a-3305-11ec-a9af-0fbbe717a04f"),
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("75b1275e-3305-11ec-8dba-8bf525336b2b"),
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingID:    uuid.FromStringOrNil("760b193a-3305-11ec-a9af-0fbbe717a04f"),
				RecordingIDs:   []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Any())
			if err := h.ConfbridgeCreate(context.Background(), tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Any())
			if err := h.ConfbridgeSetRecordID(context.Background(), tt.confbridge.ID, tt.recordID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeGet(gomock.Any(), tt.confbridge.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Any())
			res, err := h.ConfbridgeGet(context.Background(), tt.confbridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectConfbridge, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConfbridge, res)
			}
		})
	}
}
