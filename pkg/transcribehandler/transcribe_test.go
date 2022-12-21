package transcribehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/sttgoogle"
)

func TestGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockGoogle := sttgoogle.NewMockSTTGoogle(mc)

	h := &transcribeHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		sttGoogle:     mockGoogle,
	}

	tests := []struct {
		name string

		id uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("5d0166e6-877f-11ec-b42f-4f6a59ece023"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().TranscribeGet(gomock.Any(), tt.id).Return(&transcribe.Transcribe{}, nil)
			_, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockGoogle := sttgoogle.NewMockSTTGoogle(mc)

	h := &transcribeHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		sttGoogle:     mockGoogle,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("5d0166e6-877f-11ec-b42f-4f6a59ece023"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().TranscribeCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), gomock.Any()).Return(&transcribe.Transcribe{CustomerID: tt.customerID}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.customerID, transcribe.EventTypeTranscribeCreated, gomock.Any())
			_, err := h.Create(ctx, tt.customerID, uuid.Nil, transcribe.TypeCall, "en-US", common.DirectionBoth, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockGoogle := sttgoogle.NewMockSTTGoogle(mc)

	h := &transcribeHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		sttGoogle:     mockGoogle,
	}

	tests := []struct {
		name string

		id                 uuid.UUID
		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
			},

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().TranscribeDelete(gomock.Any(), tt.id).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), tt.id).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), transcribe.EventTypeTranscribeDeleted, gomock.Any())

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
