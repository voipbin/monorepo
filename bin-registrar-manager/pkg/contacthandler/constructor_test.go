package contacthandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_NewContactHandler_wiring(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAst := dbhandler.NewMockDBHandler(mc)
	mockBin := dbhandler.NewMockDBHandler(mc)
	mockDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

	res := NewContactHandler(mockReq, mockAst, mockBin, mockDomain)

	h, ok := res.(*contactHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *contactHandler, got: %T", res)
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
	if h.customerDomainHandler != mockDomain {
		t.Errorf("Wrong match. the customer domain handler is not the injected one")
	}
}
