package peereventhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-timeline-manager/models/peerevent"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

func TestNewPeerEventHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewPeerEventHandler(mockDB)

	if handler == nil {
		t.Error("NewPeerEventHandler() returned nil")
	}
}

func TestPeerEventHandler_Interface(t *testing.T) {
	// Ensure peerEventHandler implements PeerEventHandler interface
	var _ PeerEventHandler = (*peerEventHandler)(nil)
}

func TestList_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewPeerEventHandler(mockDB)

	tests := []struct {
		name       string
		customerID uuid.UUID
		addrs      []commonaddress.Address
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "missing customer_id",
			customerID: uuid.Nil,
			addrs:      []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}},
			wantErr:    true,
			errMsg:     "customer_id is required",
		},
		{
			name:       "missing addresses",
			customerID: uuid.Must(uuid.NewV4()),
			addrs:      nil,
			wantErr:    true,
			errMsg:     "at least one peer address is required",
		},
		{
			name:       "empty addresses slice",
			customerID: uuid.Must(uuid.NewV4()),
			addrs:      []commonaddress.Address{},
			wantErr:    true,
			errMsg:     "at least one peer address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.List(context.Background(), tt.customerID, tt.addrs, "", 0)
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
	handler := NewPeerEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}}

	ts1 := time.Date(2026, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2026, 1, 15, 10, 29, 0, 123000000, time.UTC)
	expectedRows := []*peerevent.PeerEvent{
		{Timestamp: ts1, EventType: "call_hangup"},
		{Timestamp: ts2, EventType: "call_created"},
	}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, addrs, "", 11).
		Return(expectedRows, nil)

	result, err := handler.List(context.Background(), testID, addrs, "", 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("List() returned %d rows, want 2", len(result.Result))
	}
	if result.NextPageToken != "" {
		t.Errorf("List() NextPageToken = %q, want empty", result.NextPageToken)
	}
}

func TestList_Pagination_HasMore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewPeerEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}}

	ts1 := time.Date(2026, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2026, 1, 15, 10, 29, 0, 123000000, time.UTC)
	ts3 := time.Date(2026, 1, 15, 10, 28, 0, 123000000, time.UTC)
	expectedRows := []*peerevent.PeerEvent{
		{Timestamp: ts1, EventType: "call_hangup"},
		{Timestamp: ts2, EventType: "call_created"},
		{Timestamp: ts3, EventType: "call_dialing"},
	}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, addrs, "", 3).
		Return(expectedRows, nil)

	result, err := handler.List(context.Background(), testID, addrs, "", 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("List() returned %d rows, want 2", len(result.Result))
	}
	if result.NextPageToken != "2026-01-15T10:29:00.123000Z" {
		t.Errorf("List() NextPageToken = %q, want %q", result.NextPageToken, "2026-01-15T10:29:00.123000Z")
	}
}

func TestList_DefaultAndMaxPageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewPeerEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, addrs, "", DefaultPageSize+1).
		Return([]*peerevent.PeerEvent{}, nil)
	if _, err := handler.List(context.Background(), testID, addrs, "", 0); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, addrs, "", MaxPageSize+1).
		Return([]*peerevent.PeerEvent{}, nil)
	if _, err := handler.List(context.Background(), testID, addrs, "", 5000); err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestList_DatabaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewPeerEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, addrs, "", 11).
		Return(nil, errors.New("clickhouse error"))

	_, err := handler.List(context.Background(), testID, addrs, "", 10)
	if err == nil {
		t.Fatal("List() expected error, got nil")
	}
}
