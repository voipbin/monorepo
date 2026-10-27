package listenhandler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/models/correlation"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

// Test_processRequest_response_encoding_failure drives the only way a route returns an error: the
// handler result cannot be encoded. The default error handling must answer with status 400 and no
// body, and must not return the error to the transport layer.
func Test_processRequest_response_encoding_failure(t *testing.T) {
	resourceID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	activeflowID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	invalidRaw := json.RawMessage("{")
	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) // years above 9999 cannot be encoded

	tests := []struct {
		name string

		request *sock.Request
		setup   func(mockEvent *eventhandler.MockEventHandler)
	}{
		{
			name: "events with a raw data field that is not JSON",
			request: &sock.Request{
				URI:    "/v1/events",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"publisher":"flow-manager","resource_id":"11111111-1111-1111-1111-111111111111","events":["activeflow_*"]}`),
			},
			setup: func(mockEvent *eventhandler.MockEventHandler) {
				mockEvent.EXPECT().List(gomock.Any(), gomock.Any(), resourceID, []string{"activeflow_*"}, "", 0).Return(
					&event.EventListResponse{Result: []*event.Event{{EventType: "activeflow_created", Data: invalidRaw}}}, nil,
				)
			},
		},
		{
			name: "aggregated events with a raw data field that is not JSON",
			request: &sock.Request{
				URI:    "/v1/aggregated-events",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"activeflow_id":"22222222-2222-2222-2222-222222222222"}`),
			},
			setup: func(mockEvent *eventhandler.MockEventHandler) {
				mockEvent.EXPECT().AggregatedList(gomock.Any(), activeflowID, "", 0).Return(
					&event.AggregatedEventListResponse{Result: []*event.Event{{EventType: "call_created", Data: invalidRaw}}}, nil,
				)
			},
		},
		{
			name: "correlation with a time outside the encodable range",
			request: &sock.Request{
				URI:    "/v1/correlations/11111111-1111-1111-1111-111111111111",
				Method: sock.RequestMethodGet,
			},
			setup: func(mockEvent *eventhandler.MockEventHandler) {
				mockEvent.EXPECT().CorrelationGet(gomock.Any(), resourceID).Return(
					&correlation.Correlation{
						ResourceID: resourceID,
						Resources: []*correlation.PublisherGroup{
							{Publisher: "call-manager", Resources: []*correlation.CorrelatedResource{{ID: resourceID, FirstSeen: outOfRange}}},
						},
					}, nil,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := logtest.NewGlobal()
			t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEvent := eventhandler.NewMockEventHandler(mc)
			tt.setup(mockEvent)

			h := &listenHandler{sockHandler: mockSock, eventHandler: mockEvent}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: nil error, got: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Fatalf("Wrong match. expect: status 400, got: %+v", res)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: empty body, got: %s", res.Data)
			}

			found := false
			for _, entry := range hook.AllEntries() {
				if entry.Level == logrus.ErrorLevel && strings.Contains(entry.Message, "Could not process the request correctly") {
					found = true
				}
			}
			if !found {
				t.Errorf("Wrong match. expect the failure to be logged as an error, got: none")
			}
		})
	}
}

func Test_marshalAnalysis_encoding_failure(t *testing.T) {
	res, err := marshalAnalysis(json.RawMessage("{"))
	if err != nil {
		t.Fatalf("Wrong match. expect: nil error, got: %v", err)
	}
	if res == nil || res.StatusCode != 500 {
		t.Errorf("Wrong match. expect: status 500, got: %+v", res)
	}
	if res != nil && len(res.Data) != 0 {
		t.Errorf("Wrong match. expect: empty body, got: %s", res.Data)
	}
}

func Test_v1CorrelationsGet_short_uri(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		sockHandler:  sockhandler.NewMockSockHandler(mc),
		eventHandler: eventhandler.NewMockEventHandler(mc),
	}

	res, err := h.v1CorrelationsGet(t.Context(), &sock.Request{URI: "/v1/correlations", Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("Wrong match. expect: nil error, got: %v", err)
	}
	if res == nil || res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: status 400, got: %+v", res)
	}
}
