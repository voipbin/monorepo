package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

func TestRegV1Correlations(t *testing.T) {
	validID := uuid.Must(uuid.NewV4()).String()

	tests := []struct {
		name    string
		uri     string
		matches bool
	}{
		{name: "valid lowercase uuid", uri: "/v1/correlations/" + validID, matches: true},
		{name: "uppercase uuid", uri: "/v1/correlations/AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE", matches: true},
		{name: "no id", uri: "/v1/correlations/", matches: false},
		{name: "no id no slash", uri: "/v1/correlations", matches: false},
		{name: "trailing path", uri: "/v1/correlations/" + validID + "/extra", matches: false},
		{name: "non-uuid", uri: "/v1/correlations/not-a-uuid", matches: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := regV1Correlations.MatchString(tt.uri)
			if result != tt.matches {
				t.Errorf("regV1Correlations.MatchString(%q) = %v, want %v", tt.uri, result, tt.matches)
			}
		})
	}
}

func TestV1CorrelationsGet_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	resourceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	expected := &response.V1DataResourceCorrelationGet{
		ResourceID:    resourceID,
		ResourceFound: true,
		ActiveflowID:  activeflowID,
		Resources:     []*event.PublisherGroup{},
	}

	mockEvent.EXPECT().ResourceCorrelationGet(gomock.Any(), resourceID).Return(expected, nil)

	sockReq := &sock.Request{
		URI:    "/v1/correlations/" + resourceID.String(),
		Method: sock.RequestMethodGet,
	}

	resp, err := handler.v1CorrelationsGet(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1CorrelationsGet() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if resp.DataType != "application/json" {
		t.Errorf("DataType = %q, want application/json", resp.DataType)
	}

	var got response.V1DataResourceCorrelationGet
	if err := json.Unmarshal(resp.Data, &got); err != nil {
		t.Fatalf("could not unmarshal response body: %v", err)
	}
	if got.ResourceID != resourceID {
		t.Errorf("body ResourceID = %v, want %v", got.ResourceID, resourceID)
	}
	if got.ActiveflowID != activeflowID {
		t.Errorf("body ActiveflowID = %v, want %v", got.ActiveflowID, activeflowID)
	}
	if !got.ResourceFound {
		t.Error("body ResourceFound = false, want true")
	}
}

func TestV1CorrelationsGet_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	// URI with a non-uuid id; handler should reject with 400 without calling eventHandler.
	sockReq := &sock.Request{
		URI:    "/v1/correlations/not-a-uuid",
		Method: sock.RequestMethodGet,
	}

	resp, err := handler.v1CorrelationsGet(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1CorrelationsGet() error = %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1CorrelationsGet_HandlerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	resourceID := uuid.Must(uuid.NewV4())

	mockEvent.EXPECT().ResourceCorrelationGet(gomock.Any(), resourceID).Return(nil, errors.New("clickhouse down"))

	sockReq := &sock.Request{
		URI:    "/v1/correlations/" + resourceID.String(),
		Method: sock.RequestMethodGet,
	}

	resp, err := handler.v1CorrelationsGet(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1CorrelationsGet() error = %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", resp.StatusCode)
	}
}

func TestProcessRequest_V1CorrelationsGet_Routing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	resourceID := uuid.Must(uuid.NewV4())
	expected := &response.V1DataResourceCorrelationGet{
		ResourceID:    resourceID,
		ResourceFound: true,
		Resources:     []*event.PublisherGroup{},
	}
	mockEvent.EXPECT().ResourceCorrelationGet(gomock.Any(), resourceID).Return(expected, nil)

	sockReq := &sock.Request{
		URI:    "/v1/correlations/" + resourceID.String(),
		Method: sock.RequestMethodGet,
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}
