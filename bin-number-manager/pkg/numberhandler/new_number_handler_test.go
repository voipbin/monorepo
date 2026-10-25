package numberhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
	"monorepo/bin-number-manager/pkg/numberhandlertwilio"
)

func Test_NewNumberHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockTwilio := numberhandlertwilio.NewMockNumberHandlerTwilio(mc)

	res := NewNumberHandler(mockReq, mockDB, mockNotify, mockTelnyx, mockTwilio)

	h, ok := res.(*numberHandler)
	if !ok {
		t.Fatalf("wrong type. got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("the util handler is not set")
	}
	if h.reqHandler != mockReq || h.db != mockDB || h.notifyHandler != mockNotify {
		t.Errorf("the injected handlers are not kept")
	}
	if h.numberHandlerTelnyx != mockTelnyx || h.numberHandlerTwilio != mockTwilio {
		t.Errorf("the injected provider handlers are not kept")
	}
}
