package transcripthandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

func Test_Create_dbErrors(t *testing.T) {
	id := uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000011")

	tests := []struct {
		name string

		createErr error
		getErr    error

		expectGet bool
	}{
		{
			name:      "create failure stops before the lookup and the webhook",
			createErr: fmt.Errorf("insert failed"),
			expectGet: false,
		},
		{
			name:      "lookup failure after create skips the webhook",
			getErr:    fmt.Errorf("select failed"),
			expectGet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transcriptHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(id)
			mockDB.EXPECT().TranscriptCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.expectGet {
				mockDB.EXPECT().TranscriptGet(ctx, id).Return(nil, tt.getErr)
			}

			res, err := h.Create(ctx, uuid.Nil, uuid.Nil, transcript.DirectionIn, "msg", 0)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_List_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &transcriptHandler{db: mockDB}
	ctx := context.Background()

	filters := map[transcript.Field]any{
		transcript.FieldTranscribeID: uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000021"),
	}
	mockDB.EXPECT().TranscriptList(ctx, uint64(10), "token", filters).Return(nil, fmt.Errorf("list failed"))

	res, err := h.List(ctx, 10, "token", filters)
	if err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_Delete_paths(t *testing.T) {
	id := uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000031")
	deleted := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		// first TranscriptGet (dbGet) response
		getRes *transcript.Transcript
		getErr error

		// dbDelete: TranscriptDelete error
		deleteErr error

		expectDelete bool
		expectErr    bool
		expectSame   bool
	}{
		{
			name:      "the lookup failure is returned",
			getErr:    fmt.Errorf("select failed"),
			expectErr: true,
		},
		{
			name: "an already deleted transcript is returned without deleting again",
			getRes: &transcript.Transcript{
				Identity: commonidentity.Identity{ID: id},
				TMDelete: &deleted,
			},
			expectSame: true,
		},
		{
			name: "the delete failure is returned",
			getRes: &transcript.Transcript{
				Identity: commonidentity.Identity{ID: id},
			},
			deleteErr:    fmt.Errorf("delete failed"),
			expectDelete: true,
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &transcriptHandler{db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			mockDB.EXPECT().TranscriptGet(ctx, id).Return(tt.getRes, tt.getErr)
			if tt.expectDelete {
				mockDB.EXPECT().TranscriptDelete(ctx, id).Return(tt.deleteErr)
			}

			res, err := h.Delete(ctx, id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.expectSame && res != tt.getRes {
				t.Errorf("Wrong match. expect the stored transcript pointer, got: %v", res)
			}
		})
	}
}

func Test_dbGet_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &transcriptHandler{db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000041")

	mockDB.EXPECT().TranscriptGet(ctx, id).Return(nil, fmt.Errorf("select failed"))

	res, err := h.dbGet(ctx, id)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
	}
}

func Test_dbDelete_errorBranches(t *testing.T) {
	id := uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000051")

	tests := []struct {
		name string

		deleteErr error
		getErr    error

		expectGet bool
	}{
		{
			name:      "delete failure",
			deleteErr: fmt.Errorf("delete failed"),
		},
		{
			name:      "lookup failure after delete",
			getErr:    fmt.Errorf("select failed"),
			expectGet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &transcriptHandler{db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			mockDB.EXPECT().TranscriptDelete(ctx, id).Return(tt.deleteErr)
			if tt.expectGet {
				mockDB.EXPECT().TranscriptGet(ctx, id).Return(nil, tt.getErr)
			}

			res, err := h.dbDelete(ctx, id)
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
			}
		})
	}
}
