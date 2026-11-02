package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

func Test_RecordingSetStatusStoppingIfActive(t *testing.T) {
	resetTestDB(t)

	created := testhelper.TimePtr("2020-04-18T03:22:17.995000Z")
	changed := testhelper.TimePtr("2020-04-18T03:22:19.995000Z")

	tests := []struct {
		name         string
		id           uuid.UUID
		status       recording.Status
		expectUpdate bool
		expectStatus recording.Status
	}{
		{
			name:         "initiating moves to stopping",
			id:           uuid.FromStringOrNil("c1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e01"),
			status:       recording.StatusInitiating,
			expectUpdate: true,
			expectStatus: recording.StatusStopping,
		},
		{
			name:         "recording moves to stopping",
			id:           uuid.FromStringOrNil("c1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e02"),
			status:       recording.StatusRecording,
			expectUpdate: true,
			expectStatus: recording.StatusStopping,
		},
		{
			name:         "ended stays ended",
			id:           uuid.FromStringOrNil("c1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e03"),
			status:       recording.StatusEnded,
			expectUpdate: false,
			expectStatus: recording.StatusEnded,
		},
		{
			name:         "stopping stays stopping",
			id:           uuid.FromStringOrNil("c1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e04"),
			status:       recording.StatusStopping,
			expectUpdate: false,
			expectStatus: recording.StatusStopping,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
			ctx := context.Background()

			r := &recording.Recording{Identity: commonidentity.Identity{ID: tt.id}, Status: tt.status}
			mockUtil.EXPECT().TimeNow().Return(created)
			mockCache.EXPECT().RecordingSet(ctx, gomock.Any())
			if err := h.RecordingCreate(ctx, r); err != nil {
				t.Fatalf("could not create the recording. err: %v", err)
			}

			// the cache is refreshed exactly once when the row changed and never otherwise.
			mockUtil.EXPECT().TimeNow().Return(changed).AnyTimes()
			cacheCalls := 0
			if tt.expectUpdate {
				cacheCalls = 1
			}
			mockCache.EXPECT().RecordingSet(ctx, gomock.Any()).Times(cacheCalls)

			updated, err := h.RecordingSetStatusStoppingIfActive(ctx, tt.id)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if updated != tt.expectUpdate {
				t.Errorf("Wrong updated. expect: %v, got: %v", tt.expectUpdate, updated)
			}

			res, err := h.recordingGetFromDB(ctx, tt.id)
			if err != nil {
				t.Fatalf("could not get the recording. err: %v", err)
			}
			if res.Status != tt.expectStatus {
				t.Errorf("Wrong status. expect: %v, got: %v", tt.expectStatus, res.Status)
			}
		})
	}
}

func Test_CallUnsetRecordingIDIfMatch(t *testing.T) {
	resetTestDB(t)

	created := testhelper.TimePtr("2020-04-18T03:22:17.995000Z")
	changed := testhelper.TimePtr("2020-04-18T03:22:19.995000Z")
	recordingID := uuid.FromStringOrNil("d1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e01")
	otherRecordingID := uuid.FromStringOrNil("d1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e02")

	tests := []struct {
		name         string
		id           uuid.UUID
		current      uuid.UUID
		unset        uuid.UUID
		expectUpdate bool
		expectID     uuid.UUID
	}{
		{
			name:         "matching id is cleared",
			id:           uuid.FromStringOrNil("e1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e01"),
			current:      recordingID,
			unset:        recordingID,
			expectUpdate: true,
			expectID:     uuid.Nil,
		},
		{
			name:         "different id is kept",
			id:           uuid.FromStringOrNil("e1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e02"),
			current:      otherRecordingID,
			unset:        recordingID,
			expectUpdate: false,
			expectID:     otherRecordingID,
		},
		{
			name:         "empty id is kept",
			id:           uuid.FromStringOrNil("e1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e03"),
			current:      uuid.Nil,
			unset:        recordingID,
			expectUpdate: false,
			expectID:     uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
			ctx := context.Background()

			c := &call.Call{
				Identity:    commonidentity.Identity{ID: tt.id},
				Type:        call.TypeFlow,
				RecordingID: tt.current,
				TMCreate:    created,
			}
			mockUtil.EXPECT().TimeNow().Return(created)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, c); err != nil {
				t.Fatalf("could not create the call. err: %v", err)
			}

			// the cache is refreshed exactly once when the row changed and never otherwise.
			mockUtil.EXPECT().TimeNow().Return(changed).AnyTimes()
			cacheCalls := 0
			if tt.expectUpdate {
				cacheCalls = 1
			}
			mockCache.EXPECT().CallSet(ctx, gomock.Any()).Times(cacheCalls)

			updated, err := h.CallUnsetRecordingIDIfMatch(ctx, tt.id, tt.unset)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if updated != tt.expectUpdate {
				t.Errorf("Wrong updated. expect: %v, got: %v", tt.expectUpdate, updated)
			}

			res, err := h.callGetFromDB(ctx, tt.id)
			if err != nil {
				t.Fatalf("could not get the call. err: %v", err)
			}
			if res.RecordingID != tt.expectID {
				t.Errorf("Wrong recording id. expect: %v, got: %v", tt.expectID, res.RecordingID)
			}
		})
	}
}
