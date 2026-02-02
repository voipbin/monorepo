package eventhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

func TestList_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	tests := []struct {
		name    string
		req     *event.EventListRequest
		wantErr bool
	}{
		{
			name: "missing publisher",
			req: &event.EventListRequest{
				ID:     uuid.Must(uuid.NewV4()),
				Events: []string{"activeflow_*"},
			},
			wantErr: true,
		},
		{
			name: "missing id",
			req: &event.EventListRequest{
				Publisher: commonoutline.ServiceNameFlowManager,
				Events:    []string{"activeflow_*"},
			},
			wantErr: true,
		},
		{
			name: "missing events",
			req: &event.EventListRequest{
				Publisher: commonoutline.ServiceNameFlowManager,
				ID:        uuid.Must(uuid.NewV4()),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.List(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
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
	req := &event.EventListRequest{
		Publisher: commonoutline.ServiceNameFlowManager,
		ID:        testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}

	expectedEvents := []*event.Event{
		{Timestamp: "2024-01-15T10:30:00.123Z", EventType: "activeflow_created"},
		{Timestamp: "2024-01-15T10:29:00.123Z", EventType: "activeflow_started"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), req.Publisher, testID, req.Events, "", 11).
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
