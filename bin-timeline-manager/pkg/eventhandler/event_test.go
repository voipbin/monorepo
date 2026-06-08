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
		name       string
		publisher  commonoutline.ServiceName
		resourceID uuid.UUID
		events     []string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "missing publisher",
			resourceID: uuid.Must(uuid.NewV4()),
			events:     []string{"activeflow_*"},
			wantErr:    true,
			errMsg:     "publisher is required",
		},
		{
			name:      "missing id",
			publisher: commonoutline.ServiceName("flow-manager"),
			events:    []string{"activeflow_*"},
			wantErr:   true,
			errMsg:    "resource_id is required",
		},
		{
			name:       "missing events",
			publisher:  commonoutline.ServiceName("flow-manager"),
			resourceID: uuid.Must(uuid.NewV4()),
			wantErr:    true,
			errMsg:     "events filter is required",
		},
		{
			name:       "empty events slice",
			publisher:  commonoutline.ServiceName("flow-manager"),
			resourceID: uuid.Must(uuid.NewV4()),
			events:     []string{},
			wantErr:    true,
			errMsg:     "events filter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.List(context.Background(), tt.publisher, tt.resourceID, tt.events, "", 0)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
		{Timestamp: ts2, EventType: "activeflow_started"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", 11).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), publisher, testID, events, "", 10)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

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
		EventList(gomock.Any(), string(publisher), testID, events, "", 3).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), publisher, testID, events, "", 2)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}
	pageToken := "2024-01-15T10:29:00.123000Z"

	ts := time.Date(2024, 1, 15, 10, 28, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts, EventType: "activeflow_finished"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, pageToken, 11).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), publisher, testID, events, pageToken, 10)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", DefaultPageSize+1).
		Return([]*event.Event{}, nil)

	// PageSize 0, should use default (100)
	_, err := handler.List(context.Background(), publisher, testID, events, "", 0)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", DefaultPageSize+1).
		Return([]*event.Event{}, nil)

	// Negative page size, should use default
	_, err := handler.List(context.Background(), publisher, testID, events, "", -5)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", MaxPageSize+1).
		Return([]*event.Event{}, nil)

	// Over max, should be capped to MaxPageSize (1000)
	_, err := handler.List(context.Background(), publisher, testID, events, "", 5000)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", 11).
		Return(nil, errors.New("database connection failed"))

	_, err := handler.List(context.Background(), publisher, testID, events, "", 10)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", 11).
		Return([]*event.Event{}, nil)

	result, err := handler.List(context.Background(), publisher, testID, events, "", 10)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_*"}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", 11).
		Return(nil, nil)

	result, err := handler.List(context.Background(), publisher, testID, events, "", 10)
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
	publisher := commonoutline.ServiceName("flow-manager")
	events := []string{"activeflow_created", "activeflow_started", "flow_*"}

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
		{Timestamp: ts2, EventType: "flow_updated"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), string(publisher), testID, events, "", 11).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), publisher, testID, events, "", 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("List() returned %d events, want 2", len(result.Result))
	}
}

func TestAggregatedList_Validation_MissingActiveflowID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	// ActiveflowID is zero value (uuid.Nil)
	_, err := handler.AggregatedList(context.Background(), uuid.Nil, "", 0)
	if err == nil {
		t.Fatal("AggregatedList() expected error, got nil")
	}
	if err.Error() != "activeflow_id is required" {
		t.Errorf("AggregatedList() error = %q, want %q", err.Error(), "activeflow_id is required")
	}
}

func TestAggregatedList_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
		{Timestamp: ts2, EventType: "call_hangup"},
	}

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", 11).
		Return(expectedEvents, nil)

	result, err := handler.AggregatedList(context.Background(), activeflowID, "", 10)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("AggregatedList() returned %d events, want 2", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("AggregatedList() NextPageToken = %q, want empty", result.NextPageToken)
	}
}

func TestAggregatedList_Pagination_HasMore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	ts3 := time.Date(2024, 1, 15, 10, 28, 0, 123000000, time.UTC)
	// Return 3 events when requesting pageSize+1 (3), indicating more results
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
		{Timestamp: ts2, EventType: "call_hangup"},
		{Timestamp: ts3, EventType: "activeflow_finished"},
	}

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", 3).
		Return(expectedEvents, nil)

	result, err := handler.AggregatedList(context.Background(), activeflowID, "", 2)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("AggregatedList() returned %d events, want 2", len(result.Result))
	}

	if result.NextPageToken != "2024-01-15T10:29:00.123000Z" {
		t.Errorf("AggregatedList() NextPageToken = %q, want %q", result.NextPageToken, "2024-01-15T10:29:00.123000Z")
	}
}

func TestAggregatedList_Pagination_NoMore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts1, EventType: "activeflow_created"},
	}

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", 11).
		Return(expectedEvents, nil)

	result, err := handler.AggregatedList(context.Background(), activeflowID, "", 10)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}

	if len(result.Result) != 1 {
		t.Errorf("AggregatedList() returned %d events, want 1", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("AggregatedList() NextPageToken = %q, want empty", result.NextPageToken)
	}
}

func TestAggregatedList_DefaultPageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", DefaultPageSize+1).
		Return([]*event.Event{}, nil)

	// PageSize 0, should use default (100)
	_, err := handler.AggregatedList(context.Background(), activeflowID, "", 0)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}
}

func TestAggregatedList_NegativePageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", DefaultPageSize+1).
		Return([]*event.Event{}, nil)

	// Negative page size, should use default
	_, err := handler.AggregatedList(context.Background(), activeflowID, "", -5)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}
}

func TestAggregatedList_MaxPageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", MaxPageSize+1).
		Return([]*event.Event{}, nil)

	// Over max, should be capped to MaxPageSize (1000)
	_, err := handler.AggregatedList(context.Background(), activeflowID, "", 5000)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}
}

func TestAggregatedList_DatabaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", 11).
		Return(nil, errors.New("database connection failed"))

	_, err := handler.AggregatedList(context.Background(), activeflowID, "", 10)
	if err == nil {
		t.Fatal("AggregatedList() expected error, got nil")
	}
}

func TestAggregatedList_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), "", 11).
		Return([]*event.Event{}, nil)

	result, err := handler.AggregatedList(context.Background(), activeflowID, "", 10)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}

	if len(result.Result) != 0 {
		t.Errorf("AggregatedList() returned %d events, want 0", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("AggregatedList() NextPageToken = %q, want empty", result.NextPageToken)
	}
}

func TestAggregatedList_WithPageToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	activeflowID := uuid.Must(uuid.NewV4())
	pageToken := "2024-01-15T10:29:00.123000Z"

	ts := time.Date(2024, 1, 15, 10, 28, 0, 123000000, time.UTC)
	expectedEvents := []*event.Event{
		{Timestamp: ts, EventType: "activeflow_finished"},
	}

	mockDB.EXPECT().
		AggregatedEventList(gomock.Any(), activeflowID.String(), pageToken, 11).
		Return(expectedEvents, nil)

	result, err := handler.AggregatedList(context.Background(), activeflowID, pageToken, 10)
	if err != nil {
		t.Fatalf("AggregatedList() error = %v", err)
	}

	if len(result.Result) != 1 {
		t.Errorf("AggregatedList() returned %d events, want 1", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("AggregatedList() NextPageToken = %q, want empty", result.NextPageToken)
	}
}
