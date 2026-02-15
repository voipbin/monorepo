package accesskeyhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"go.uber.org/mock/gomock"
)

func TestNewAccesskeyHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewAccesskeyHandler(mockReq, mockDB, mockNotify)
	if h == nil {
		t.Error("NewAccesskeyHandler returned nil")
	}
}

func TestDefaultLenToken(t *testing.T) {
	if defaultLenToken != 16 {
		t.Errorf("defaultLenToken = %d, expected 16", defaultLenToken)
	}
}
