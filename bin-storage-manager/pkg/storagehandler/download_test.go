package storagehandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-storage-manager/pkg/filehandler"
)

func Test_FileDownloadURIRefresh(t *testing.T) {
	id := uuid.FromStringOrNil("6c1f5a1e-0000-0000-0000-000000000001")
	errRefresh := errors.New("simulated refresh failure")

	tests := []struct {
		name string

		responseURI string
		responseErr error

		expectRes string
		expectErr error
	}{
		{
			name:        "normal",
			responseURI: "https://example.invalid/signed",
			expectRes:   "https://example.invalid/signed",
		},
		{
			name:        "refresh failed",
			responseErr: errRefresh,
			expectErr:   errRefresh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockFile := filehandler.NewMockFileHandler(mc)
			h := &storageHandler{fileHandler: mockFile}
			ctx := context.Background()

			mockFile.EXPECT().DownloadURIRefresh(ctx, id).Return(tt.responseURI, tt.responseErr)

			res, err := h.FileDownloadURIRefresh(ctx, id)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, res)
			}
		})
	}
}
