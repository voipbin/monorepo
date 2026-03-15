package subscribehandler

import (
	"encoding/json"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

func Test_processEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "normal",
			event: &sock.Event{
				Type:      "call_created",
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      json.RawMessage(`{"id":"test-id"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &subscribeHandler{
				dbHandler: mockDB,
			}

			mockDB.EXPECT().EventInsert(
				gomock.Any(),
				gomock.Any(), // timestamp
				tt.event.Type,
				tt.event.Publisher,
				tt.event.DataType,
				string(tt.event.Data),
			).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
