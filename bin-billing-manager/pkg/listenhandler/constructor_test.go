package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/failedeventhandler"
	"monorepo/bin-billing-manager/pkg/paddlehandler"
)

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockBilling := billinghandler.NewMockBillingHandler(mc)
	mockPaddle := paddlehandler.NewMockPaddleHandler(mc)
	mockFailed := failedeventhandler.NewMockFailedEventHandler(mc)

	res := NewListenHandler(mockSock, mockAccount, mockBilling, mockPaddle, mockFailed)
	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *listenHandler, got: %T", res)
	}

	if h.utilHandler == nil || h.sockHandler != mockSock || h.accountHandler != mockAccount || h.billingHandler != mockBilling || h.paddleHandler != mockPaddle || h.failedEventHandler != mockFailed {
		t.Errorf("Wrong match. every dependency must be wired. got: %+v", h)
	}
}
