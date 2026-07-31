package schedulehandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

func TestNewScheduleHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewScheduleHandler(mockReq, mockDB, mockNotify)

	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}
