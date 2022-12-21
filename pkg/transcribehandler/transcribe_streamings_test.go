package transcribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcirpthandler"
)

func TestTranscribeStreamingsHandle(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockGoogle := transcirpthandler.NewMockTranscriptHandler(mc)

	h := &transcribeHandler{
		reqHandler:        mockReq,
		db:                mockDB,
		notifyHandler:     mockNotify,
		transcriptHandler: mockGoogle,

		transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
	}

	tests := []struct {
		name string

		loopCount int
	}{
		{
			"normal",

			1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for i := 0; i < tt.loopCount; i++ {
				go func() {
					id := uuid.Must(uuid.NewV4())
					h.addTranscribeStreamings(id, []*streaming.Streaming{})
					tmp := h.getTranscribeStreamings(id)
					if tmp == nil {
						t.Errorf("Wrong match. expect: not nil, got: nil")
					}
					h.deleteTranscribeStreamings(id)
				}()
			}
		})
	}
}
