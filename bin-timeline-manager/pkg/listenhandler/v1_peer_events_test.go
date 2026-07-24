package listenhandler

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-timeline-manager/models/peerevent"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
	"monorepo/bin-timeline-manager/pkg/peereventhandler"
)

func TestProcessRequest_V1PeerEventsGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPeerEvent := peereventhandler.NewMockPeerEventHandler(ctrl)

	handler := &listenHandler{
		peerEventHandler: mockPeerEvent,
	}

	testCustomerID := uuid.Must(uuid.NewV4())
	body := &request.V1DataPeerEventsGet{
		PeerAddresses: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}},
	}
	reqData, _ := json.Marshal(body)

	ts := time.Date(2026, 1, 15, 10, 30, 0, 123000000, time.UTC)
	expectedResponse := &peerevent.PeerEventListResponse{
		Result: []*peerevent.PeerEvent{
			{Timestamp: ts, EventType: "call_hangup"},
		},
	}

	mockPeerEvent.EXPECT().
		List(gomock.Any(), testCustomerID, body.PeerAddresses, "", 10).
		Return(expectedResponse, nil)

	sockReq := &sock.Request{
		URI:    "/v1/peer-events?customer_id=" + testCustomerID.String() + "&page_size=10",
		Method: sock.RequestMethodGet,
		Data:   reqData,
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("processRequest() StatusCode = %d, want 200", resp.StatusCode)
	}

	var got response.V1DataPeerEventsGet
	if err := json.Unmarshal(resp.Data, &got); err != nil {
		t.Fatalf("could not unmarshal response body: %v", err)
	}
	if len(got.Result) != 1 {
		t.Fatalf("body Result len = %d, want 1", len(got.Result))
	}
	if got.Result[0].EventType != "call_hangup" {
		t.Errorf("body Result[0].EventType = %q, want call_hangup", got.Result[0].EventType)
	}
}

func TestProcessRequest_V1PeerEventsGet_MissingCustomerID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPeerEvent := peereventhandler.NewMockPeerEventHandler(ctrl)

	handler := &listenHandler{
		peerEventHandler: mockPeerEvent,
	}

	sockReq := &sock.Request{
		URI:    "/v1/peer-events",
		Method: sock.RequestMethodGet,
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestProcessRequest_V1PeerEventsGet_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPeerEvent := peereventhandler.NewMockPeerEventHandler(ctrl)

	handler := &listenHandler{
		peerEventHandler: mockPeerEvent,
	}

	testCustomerID := uuid.Must(uuid.NewV4())
	sockReq := &sock.Request{
		URI:    "/v1/peer-events?customer_id=" + testCustomerID.String(),
		Method: sock.RequestMethodGet,
		Data:   []byte("invalid json"),
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestProcessRequest_V1PeerEventsGet_HandlerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPeerEvent := peereventhandler.NewMockPeerEventHandler(ctrl)

	handler := &listenHandler{
		peerEventHandler: mockPeerEvent,
	}

	testCustomerID := uuid.Must(uuid.NewV4())
	body := &request.V1DataPeerEventsGet{
		PeerAddresses: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}},
	}
	reqData, _ := json.Marshal(body)

	mockPeerEvent.EXPECT().
		List(gomock.Any(), testCustomerID, body.PeerAddresses, "", 10).
		Return(nil, errors.New("handler error"))

	sockReq := &sock.Request{
		URI:    "/v1/peer-events?customer_id=" + testCustomerID.String() + "&page_size=10",
		Method: sock.RequestMethodGet,
		Data:   reqData,
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %d, want 500", resp.StatusCode)
	}
}

func TestProcessRequest_V1PeerEventsGet_NilHandler(t *testing.T) {
	handler := &listenHandler{
		peerEventHandler: nil,
	}

	testCustomerID := uuid.Must(uuid.NewV4())
	sockReq := &sock.Request{
		URI:    "/v1/peer-events?customer_id=" + testCustomerID.String(),
		Method: sock.RequestMethodGet,
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}
	if resp.StatusCode != 503 {
		t.Errorf("processRequest() StatusCode = %d, want 503", resp.StatusCode)
	}
}

func TestRegexPatterns_PeerEvents(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{name: "no query string", uri: "/v1/peer-events", want: true},
		{name: "with query string", uri: "/v1/peer-events?customer_id=abc&page_token=&page_size=50", want: true},
		{name: "different path", uri: "/v1/events", want: false},
		{name: "unrelated suffix", uri: "/v1/peer-events-extra", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := regV1PeerEvents.MatchString(tt.uri)
			if got != tt.want {
				t.Errorf("regV1PeerEvents.MatchString(%q) = %v, want %v", tt.uri, got, tt.want)
			}
		})
	}
}
