package eventhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
)

func TestNewEventHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	if handler == nil {
		t.Error("NewEventHandler() returned nil")
	}
}

func TestEventHandler_Interface(t *testing.T) {
	// Ensure eventHandler implements EventHandler interface
	var _ EventHandler = (*eventHandler)(nil)
}

func TestList_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	tests := []struct {
		name    string
		req     *request.V1DataEventsPost
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing publisher",
			req: &request.V1DataEventsPost{
				ResourceID: uuid.Must(uuid.NewV4()),
				Events: []string{"activeflow_*"},
			},
			wantErr: true,
			errMsg:  "publisher is required",
		},
		{
			name: "missing id",
			req: &request.V1DataEventsPost{
				Publisher: commonoutline.ServiceName("flow-manager"),
				Events:    []string{"activeflow_*"},
			},
			wantErr: true,
			errMsg:  "resource_id is required",
		},
		{
			name: "missing events",
			req: &request.V1DataEventsPost{
				Publisher: commonoutline.ServiceName("flow-manager"),
				ResourceID: uuid.Must(uuid.NewV4()),
			},
			wantErr: true,
			errMsg:  "events filter is required",
		},
		{
			name: "empty events slice",
			req: &request.V1DataEventsPost{
				Publisher: commonoutline.ServiceName("flow-manager"),
				ResourceID: uuid.Must(uuid.NewV4()),
				Events:    []string{},
			},
			wantErr: true,
			errMsg:  "events filter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.List(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("List() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestList_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
		{Timestamp: ts2, EventType: "activeflow_started"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", 11).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("List() returned %d events, want 2", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("List() NextPageToken = %q, want empty", result.NextPageToken)
	}
}

func TestList_Pagination_HasMore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  2,
	}

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	ts3 := time.Date(2024, 1, 15, 10, 28, 0, 123000000, time.UTC)
	// Return 3 events when requesting pageSize+1 (3), indicating more results
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
		{Timestamp: ts2, EventType: "activeflow_started"},
		{Timestamp: ts3, EventType: "activeflow_finished"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", 3).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("List() returned %d events, want 2", len(result.Result))
	}

	if result.NextPageToken != "2024-01-15T10:29:00.123000Z" {
		t.Errorf("List() NextPageToken = %q, want %q", result.NextPageToken, "2024-01-15T10:29:00.123000Z")
	}
}

func TestList_Pagination_WithPageToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	pageToken := "2024-01-15T10:29:00.123000Z"
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
		PageToken: pageToken,
	}

	ts := time.Date(2024, 1, 15, 10, 28, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts, EventType: "activeflow_finished"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, pageToken, 11).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 1 {
		t.Errorf("List() returned %d events, want 1", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("List() NextPageToken = %q, want empty", result.NextPageToken)
	}
}

func TestList_DefaultPageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		// PageSize not set, should use default (100)
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", request.DefaultPageSize+1).
		Return([]*event.Event{}, nil)

	_, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestList_NegativePageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  -5, // Negative, should use default
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", request.DefaultPageSize+1).
		Return([]*event.Event{}, nil)

	_, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestList_MaxPageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  5000, // Over max, should be capped to MaxPageSize (1000)
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", request.MaxPageSize+1).
		Return([]*event.Event{}, nil)

	_, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestList_DatabaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", 11).
		Return(nil, errors.New("database connection failed"))

	_, err := handler.List(context.Background(), req)
	if err == nil {
		t.Fatal("List() expected error, got nil")
	}
}

func TestList_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", 11).
		Return([]*event.Event{}, nil)

	result, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 0 {
		t.Errorf("List() returned %d events, want 0", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("List() NextPageToken = %q, want empty", result.NextPageToken)
	}
}

func TestList_NilResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", 11).
		Return(nil, nil)

	result, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 0 {
		t.Errorf("List() returned %d events, want 0", len(result.Result))
	}
}

func TestList_MultipleEventFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_created", "activeflow_started", "flow_*"},
		PageSize:  10,
	}

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
		{Timestamp: ts2, EventType: "flow_updated"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(req.Publisher), testID, req.Events, "", 11).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("List() returned %d events, want 2", len(result.Result))
	}
}
