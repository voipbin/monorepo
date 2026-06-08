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
	tmcorrelation "monorepo/bin-timeline-manager/models/correlation"
)

func Test_TimelineV1CorrelationGet(t *testing.T) {

	tests := []struct {
		name string

		resourceID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *tmcorrelation.CorrelationResponse
	}{
		{
			name: "normal",

			resourceID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),

			expectTarget: "bin-manager.timeline-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/correlations/55ecfc4e-2c74-11ee-98fb-0762519529f3",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"resource_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","resource_found":true,"activeflow_id":"a8d3b3e2-2c74-11ee-98fb-0762519529f3","truncated":false,"resources":[{"publisher":"call-manager","resources":[{"id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","data_type":"call","event_types":["call_created"],"first_seen":"2024-01-15T10:00:00Z","last_seen":"2024-01-15T10:05:00Z"}]}]}`),
			},
			expectRes: &tmcorrelation.CorrelationResponse{
				ResourceID:    uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
				ResourceFound: true,
				ActiveflowID:  uuid.FromStringOrNil("a8d3b3e2-2c74-11ee-98fb-0762519529f3"),
				Truncated:     false,
				Resources: []*tmcorrelation.PublisherGroup{
					{
						Publisher: "call-manager",
						Resources: []*tmcorrelation.CorrelatedResource{
							{
								ID:         uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
								DataType:   "call",
								EventTypes: []string{"call_created"},
								FirstSeen:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
								LastSeen:   time.Date(2024, 1, 15, 10, 5, 0, 0, time.UTC),
							},
						},
					},
				},
			},
		},
		{
			name: "resource not found",

			resourceID: uuid.FromStringOrNil("66ecfc4e-2c74-11ee-98fb-0762519529f3"),

			expectTarget: "bin-manager.timeline-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/correlations/66ecfc4e-2c74-11ee-98fb-0762519529f3",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"resource_id":"66ecfc4e-2c74-11ee-98fb-0762519529f3","resource_found":false}`),
			},
			expectRes: &tmcorrelation.CorrelationResponse{
				ResourceID:    uuid.FromStringOrNil("66ecfc4e-2c74-11ee-98fb-0762519529f3"),
				ResourceFound: false,
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

			res, err := reqHandler.TimelineV1CorrelationGet(ctx, tt.resourceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
