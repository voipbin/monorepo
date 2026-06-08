package eventhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-timeline-manager/models/correlation"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

func TestResourceCorrelationGet_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	_, err := handler.ResourceCorrelationGet(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("expected error for nil resource_id")
	}
	if err.Error() != "resource_id is required" {
		t.Errorf("ResourceCorrelationGet() error = %q, want %q", err.Error(), "resource_id is required")
	}
}

func TestResourceCorrelationGet_NoActiveflow_ResourceFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return("", nil)
	mockDB.EXPECT().ResourceExists(gomock.Any(), resourceID.String()).Return(true, nil)

	res, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.ResourceFound {
		t.Error("expected ResourceFound=true")
	}
	if res.ActiveflowID != uuid.Nil {
		t.Errorf("expected ActiveflowID=nil, got %v", res.ActiveflowID)
	}
	if len(res.Resources) != 0 {
		t.Errorf("expected empty resources, got %d", len(res.Resources))
	}
	if res.Truncated {
		t.Error("expected Truncated=false")
	}
}

func TestResourceCorrelationGet_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return("", nil)
	mockDB.EXPECT().ResourceExists(gomock.Any(), resourceID.String()).Return(false, nil)

	res, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ResourceFound {
		t.Error("expected ResourceFound=false")
	}
	if len(res.Resources) != 0 {
		t.Errorf("expected empty resources, got %d", len(res.Resources))
	}
}

func TestResourceCorrelationGet_Success_GroupedAndSorted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	callID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())

	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Minute)

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return(activeflowID.String(), nil)
	// ai-manager row first to verify publisher sorting puts call-manager before ai... no, alphabetical: ai-manager < call-manager
	mockDB.EXPECT().CorrelatedResourceList(gomock.Any(), activeflowID.String(), maxCorrelationResources+1).Return([]*correlation.CorrelatedRow{
		{
			Publisher:  "call-manager",
			ResourceID: callID.String(),
			DataType:   "call",
			EventTypes: []string{"call_progressing", "call_created"},
			FirstSeen:  t0,
			LastSeen:   t1,
		},
		{
			Publisher:  "ai-manager",
			ResourceID: aicallID.String(),
			DataType:   "aicall",
			EventTypes: []string{"aicall_initiating"},
			FirstSeen:  t0,
			LastSeen:   t1,
		},
	}, nil)

	res, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.ResourceFound {
		t.Error("expected ResourceFound=true")
	}
	if res.ActiveflowID != activeflowID {
		t.Errorf("expected ActiveflowID=%v, got %v", activeflowID, res.ActiveflowID)
	}
	if res.Truncated {
		t.Error("expected Truncated=false")
	}
	if len(res.Resources) != 2 {
		t.Fatalf("expected 2 publisher groups, got %d", len(res.Resources))
	}
	// publishers sorted alphabetically: ai-manager before call-manager
	if res.Resources[0].Publisher != "ai-manager" {
		t.Errorf("expected first group ai-manager, got %s", res.Resources[0].Publisher)
	}
	if res.Resources[1].Publisher != "call-manager" {
		t.Errorf("expected second group call-manager, got %s", res.Resources[1].Publisher)
	}
	// event_types sorted within resource
	callRes := res.Resources[1].Resources[0]
	if callRes.EventTypes[0] != "call_created" || callRes.EventTypes[1] != "call_progressing" {
		t.Errorf("expected sorted event_types, got %v", callRes.EventTypes)
	}
}

func TestResourceCorrelationGet_Truncation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	// Return maxCorrelationResources+1 rows to trigger truncation.
	rows := make([]*correlation.CorrelatedRow, maxCorrelationResources+1)
	base := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	for i := range rows {
		rows[i] = &correlation.CorrelatedRow{
			Publisher:  "call-manager",
			ResourceID: uuid.Must(uuid.NewV4()).String(),
			DataType:   "call",
			EventTypes: []string{"call_created"},
			FirstSeen:  base.Add(time.Duration(i) * time.Second),
			LastSeen:   base.Add(time.Duration(i) * time.Second),
		}
	}

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return(activeflowID.String(), nil)
	mockDB.EXPECT().CorrelatedResourceList(gomock.Any(), activeflowID.String(), maxCorrelationResources+1).Return(rows, nil)

	res, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Truncated {
		t.Error("expected Truncated=true")
	}
	total := 0
	for _, g := range res.Resources {
		total += len(g.Resources)
	}
	if total != maxCorrelationResources {
		t.Errorf("expected %d resources after truncation, got %d", maxCorrelationResources, total)
	}
}

func TestResourceCorrelationGet_SkipsUnparseableResourceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	goodID := uuid.Must(uuid.NewV4())
	base := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return(activeflowID.String(), nil)
	mockDB.EXPECT().CorrelatedResourceList(gomock.Any(), activeflowID.String(), maxCorrelationResources+1).Return([]*correlation.CorrelatedRow{
		{Publisher: "call-manager", ResourceID: "not-a-uuid", DataType: "call", EventTypes: []string{"call_created"}, FirstSeen: base, LastSeen: base},
		{Publisher: "call-manager", ResourceID: goodID.String(), DataType: "call", EventTypes: []string{"call_created"}, FirstSeen: base, LastSeen: base},
	}, nil)

	res, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	total := 0
	for _, g := range res.Resources {
		total += len(g.Resources)
	}
	if total != 1 {
		t.Errorf("expected 1 valid resource (1 skipped), got %d", total)
	}
}

func TestResourceCorrelationGet_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return("", errors.New("clickhouse down"))

	_, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err == nil {
		t.Fatal("expected error from db failure")
	}
}

func TestResourceCorrelationGet_ResourceExistsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return("", nil)
	mockDB.EXPECT().ResourceExists(gomock.Any(), resourceID.String()).Return(false, errors.New("clickhouse down"))

	_, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err == nil {
		t.Fatal("expected error from ResourceExists failure")
	}
}

func TestResourceCorrelationGet_CorrelatedListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return(activeflowID.String(), nil)
	mockDB.EXPECT().CorrelatedResourceList(gomock.Any(), activeflowID.String(), maxCorrelationResources+1).Return(nil, errors.New("clickhouse down"))

	_, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err == nil {
		t.Fatal("expected error from CorrelatedResourceList failure")
	}
}

func TestResourceCorrelationGet_UnparseableActiveflowID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return("garbage-not-a-uuid", nil)

	_, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err == nil {
		t.Fatal("expected error from unparseable activeflow_id")
	}
}

func TestResourceCorrelationGet_ActiveflowFoundButEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	resourceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().ResourceActiveflowIDGet(gomock.Any(), resourceID.String()).Return(activeflowID.String(), nil)
	mockDB.EXPECT().CorrelatedResourceList(gomock.Any(), activeflowID.String(), maxCorrelationResources+1).Return([]*correlation.CorrelatedRow{}, nil)

	res, err := handler.ResourceCorrelationGet(context.Background(), resourceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.ResourceFound {
		t.Error("expected ResourceFound=true")
	}
	if res.ActiveflowID != activeflowID {
		t.Errorf("expected ActiveflowID=%v, got %v", activeflowID, res.ActiveflowID)
	}
	if res.Truncated {
		t.Error("expected Truncated=false")
	}
	if len(res.Resources) != 0 {
		t.Errorf("expected empty resources, got %d", len(res.Resources))
	}
}
