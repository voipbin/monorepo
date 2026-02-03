package requesthandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	tmevent "monorepo/bin-timeline-manager/models/event"
)

func Test_TimelineV1EventList(t *testing.T) {

	tests := []struct {
		name string

		req *tmevent.EventListRequest

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *tmevent.EventListResponse
	}{
		{
			name: "normal",

			req: &tmevent.EventListRequest{
				Publisher:  commonoutline.ServiceNameCallManager,
				ResourceID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
				Events:     []string{"call_created", "call_progressing"},
				PageSize:   10,
			},

			expectTarget: "bin-manager.timeline-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/events",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"publisher":"call-manager","resource_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","events":["call_created","call_progressing"],"page_size":10}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"result":[{"timestamp":"2024-01-15T10:00:00Z","event_type":"call_created","publisher":"call-manager","data_type":"application/json","data":{"id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}}],"next_page_token":"token123"}`),
			},
			expectRes: &tmevent.EventListResponse{
				Result: []*tmevent.Event{
					{
						Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
						EventType: "call_created",
						Publisher: commonoutline.ServiceNameCallManager,
						DataType:  "application/json",
						Data:      []byte(`{"id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
					},
				},
				NextPageToken: "token123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TimelineV1EventList(ctx, tt.req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
