package extensionhandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_NewExtensionHandler_wiring(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAst := dbhandler.NewMockDBHandler(mc)
	mockBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

	res := NewExtensionHandler(mockReq, mockAst, mockBin, mockNotify, mockDomain)

	h, ok := res.(*extensionHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *extensionHandler, got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. the util handler must be created")
	}
	if h.reqHandler != mockReq {
		t.Errorf("Wrong match. the request handler is not the injected one")
	}
	if h.dbAst != mockAst {
		t.Errorf("Wrong match. the asterisk database handler is not the injected one")
	}
	if h.dbBin != mockBin {
		t.Errorf("Wrong match. the bin database handler is not the injected one")
	}
	if h.notifyHandler != mockNotify {
		t.Errorf("Wrong match. the notify handler is not the injected one")
	}
	if h.customerDomainHandler != mockDomain {
		t.Errorf("Wrong match. the customer domain handler is not the injected one")
	}
}
