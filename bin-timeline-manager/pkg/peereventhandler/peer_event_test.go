package peereventhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

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
		pairs      []PeerPair
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "missing customer_id",
			customerID: uuid.Nil,
			pairs:      []PeerPair{{PeerType: "tel", PeerTarget: "+15551234567"}},
			wantErr:    true,
			errMsg:     "customer_id is required",
		},
		{
			name:       "missing pairs",
			customerID: uuid.Must(uuid.NewV4()),
			pairs:      nil,
			wantErr:    true,
			errMsg:     "at least one peer_type+peer_target pair is required",
		},
		{
			name:       "empty pairs slice",
			customerID: uuid.Must(uuid.NewV4()),
			pairs:      []PeerPair{},
			wantErr:    true,
			errMsg:     "at least one peer_type+peer_target pair is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.List(context.Background(), tt.customerID, tt.pairs, "", 0)
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
	pairs := []PeerPair{{PeerType: "tel", PeerTarget: "+15551234567"}}
	dbPairs := []dbhandler.PeerPairFilter{{PeerType: "tel", PeerTarget: "+15551234567"}}

	ts1 := time.Date(2026, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2026, 1, 15, 10, 29, 0, 123000000, time.UTC)
	expectedRows := []*peerevent.PeerEvent{
		{Timestamp: ts1, EventType: "call_hangup"},
		{Timestamp: ts2, EventType: "call_created"},
	}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, dbPairs, "", 11).
		Return(expectedRows, nil)

	result, err := handler.List(context.Background(), testID, pairs, "", 10)
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
	pairs := []PeerPair{{PeerType: "tel", PeerTarget: "+15551234567"}}
	dbPairs := []dbhandler.PeerPairFilter{{PeerType: "tel", PeerTarget: "+15551234567"}}

	ts1 := time.Date(2026, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2026, 1, 15, 10, 29, 0, 123000000, time.UTC)
	ts3 := time.Date(2026, 1, 15, 10, 28, 0, 123000000, time.UTC)
	expectedRows := []*peerevent.PeerEvent{
		{Timestamp: ts1, EventType: "call_hangup"},
		{Timestamp: ts2, EventType: "call_created"},
		{Timestamp: ts3, EventType: "call_dialing"},
	}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, dbPairs, "", 3).
		Return(expectedRows, nil)

	result, err := handler.List(context.Background(), testID, pairs, "", 2)
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
	pairs := []PeerPair{{PeerType: "tel", PeerTarget: "+15551234567"}}
	dbPairs := []dbhandler.PeerPairFilter{{PeerType: "tel", PeerTarget: "+15551234567"}}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, dbPairs, "", DefaultPageSize+1).
		Return([]*peerevent.PeerEvent{}, nil)
	if _, err := handler.List(context.Background(), testID, pairs, "", 0); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, dbPairs, "", MaxPageSize+1).
		Return([]*peerevent.PeerEvent{}, nil)
	if _, err := handler.List(context.Background(), testID, pairs, "", 5000); err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestList_DatabaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewPeerEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	pairs := []PeerPair{{PeerType: "tel", PeerTarget: "+15551234567"}}
	dbPairs := []dbhandler.PeerPairFilter{{PeerType: "tel", PeerTarget: "+15551234567"}}

	mockDB.EXPECT().
		PeerEventList(gomock.Any(), testID, dbPairs, "", 11).
		Return(nil, errors.New("clickhouse error"))

	_, err := handler.List(context.Background(), testID, pairs, "", 10)
	if err == nil {
		t.Fatal("List() expected error, got nil")
	}
}
