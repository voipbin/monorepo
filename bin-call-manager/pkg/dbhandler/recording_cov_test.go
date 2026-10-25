package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

func Test_RecordingSetStatus_cov(t *testing.T) {
	created := testhelper.TimePtr("2020-04-18T03:22:17.995000Z")
	changed := testhelper.TimePtr("2020-04-18T03:22:19.995000Z")

	tests := []struct {
		name   string
		id     uuid.UUID
		status recording.Status

		expectErr bool

		expectTMStart bool
		expectTMEnd   bool
	}{
		{
			name:          "recording sets tm_start",
			id:            uuid.FromStringOrNil("a4f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01"),
			status:        recording.StatusRecording,
			expectTMStart: true,
		},
		{
			name:        "ended sets tm_end",
			id:          uuid.FromStringOrNil("a4f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e02"),
			status:      recording.StatusEnded,
			expectTMEnd: true,
		},
		{
			name:   "stopping sets only the status",
			id:     uuid.FromStringOrNil("a4f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e03"),
			status: recording.StatusStopping,
		},
		{
			name:      "unsupported status",
			id:        uuid.FromStringOrNil("a4f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e04"),
			status:    recording.Status("initiating-unsupported"),
			expectErr: true,
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

			r := &recording.Recording{Identity: commonidentity.Identity{ID: tt.id}, Status: recording.StatusInitiating}
			mockUtil.EXPECT().TimeNow().Return(created)
			mockCache.EXPECT().RecordingSet(ctx, gomock.Cond(func(x *recording.Recording) bool { return x != nil && x.ID == tt.id }))
			if err := h.RecordingCreate(ctx, r); err != nil {
				t.Fatalf("could not create the recording. err: %v", err)
			}

			// RecordingUpdate stamps tm_update itself, so the clock is read at least once.
			if !tt.expectErr {
				mockUtil.EXPECT().TimeNow().Return(changed).MinTimes(1)
			}
			if !tt.expectErr {
				mockCache.EXPECT().RecordingSet(ctx, gomock.Cond(func(x *recording.Recording) bool { return x != nil && x.ID == tt.id }))
			}

			err := h.RecordingSetStatus(ctx, tt.id, tt.status)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.recordingGetFromDB(ctx, tt.id)
			if err != nil {
				t.Fatalf("could not get the recording. err: %v", err)
			}
			if res.Status != tt.status {
				t.Errorf("Wrong status. expect: %v, got: %v", tt.status, res.Status)
			}
			if gotStart := res.TMStart != nil && res.TMStart.Equal(*changed); gotStart != tt.expectTMStart {
				t.Errorf("Wrong tm_start. expect set: %v, got: %v", tt.expectTMStart, res.TMStart)
			}
			if gotEnd := res.TMEnd != nil && res.TMEnd.Equal(*changed); gotEnd != tt.expectTMEnd {
				t.Errorf("Wrong tm_end. expect set: %v, got: %v", tt.expectTMEnd, res.TMEnd)
			}
		})
	}
}
