package taghandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/pkg/dbhandler"
)

func Test_Get_NotFoundWrapping(t *testing.T) {
	tests := []struct {
		name string

		dbErr error

		expectTyped    bool
		expectNotFound bool
	}{
		{
			name:           "not_found_is_wrapped_as_typed_error",
			dbErr:          fmt.Errorf("lookup failed: %w", dbhandler.ErrNotFound),
			expectTyped:    true,
			expectNotFound: true,
		},
		{
			name:           "other_error_passes_through",
			dbErr:          fmt.Errorf("connection lost"),
			expectTyped:    false,
			expectNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := tagHandler{
				reqHandler:    mockReq,
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()
			id := uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211")

			mockDB.EXPECT().TagGet(ctx, id).Return(nil, tt.dbErr)

			res, err := h.Get(ctx, id)
			if err == nil {
				t.Fatalf("Expected an error, got nil. res: %v", res)
			}
			if res != nil {
				t.Errorf("Expected a nil result on error. got: %v", res)
			}

			var ve *cerrors.VoipbinError
			if got := stderrors.As(err, &ve); got != tt.expectTyped {
				t.Errorf("Wrong typed error match. expect: %v, got: %v, err: %v", tt.expectTyped, got, err)
			}
			if tt.expectTyped && (ve.Status != cerrors.StatusNotFound || ve.Reason != "TAG_NOT_FOUND") {
				t.Errorf("Wrong typed error. status: %v, reason: %s", ve.Status, ve.Reason)
			}
			if got := stderrors.Is(err, dbhandler.ErrNotFound); got != tt.expectNotFound {
				t.Errorf("Wrong ErrNotFound match. expect: %v, got: %v", tt.expectNotFound, got)
			}
		})
	}
}
