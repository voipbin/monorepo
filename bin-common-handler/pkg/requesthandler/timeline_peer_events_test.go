package requesthandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

func Test_TimelineV1PeerEventList(t *testing.T) {

	tests := []struct {
		name string

		req *tmpeerevent.PeerEventListRequest

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *tmpeerevent.PeerEventListResponse
	}{
		{
			name: "normal",

			req: &tmpeerevent.PeerEventListRequest{
				CustomerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
				PeerPairs: []tmpeerevent.PeerPair{
					{PeerType: "tel", PeerTarget: "+15551234567"},
				},
				PageToken: "",
				PageSize:  10,
			},

			expectTarget: "bin-manager.timeline-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/peer-events?customer_id=55ecfc4e-2c74-11ee-98fb-0762519529f3&page_token=&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"peer_pairs":[{"peer_type":"tel","peer_target":"+15551234567"}]}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"result":[{"timestamp":"2026-01-15T10:00:00Z","customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","publisher":"call","event_type":"call_hangup","reference_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","direction":"outgoing","peer_type":"tel","peer_target":"+15551234567","local_type":"tel","local_target":"+15559876543","data":{"id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}}],"next_page_token":"token123"}`),
			},
			expectRes: &tmpeerevent.PeerEventListResponse{
				Result: []*tmpeerevent.PeerEvent{
					{
						Timestamp:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
						CustomerID:  uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
						Publisher:   "call",
						EventType:   "call_hangup",
						ReferenceID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
						Direction:   "outgoing",
						PeerType:    "tel",
						PeerTarget:  "+15551234567",
						LocalType:   "tel",
						LocalTarget: "+15559876543",
						Data:        []byte(`{"id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
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

			res, err := reqHandler.TimelineV1PeerEventList(ctx, tt.req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
