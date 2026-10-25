package contacthandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

var errInjectedContact = errors.New("injected failure")

func Test_ContactGetsByExtension_contactQueryFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	h := &contactHandler{dbAst: mockDBAst, customerDomainHandler: mockCustomerDomain}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aa15c2c0-5652-11ee-b3a1-0b960b5e92e8")
	endpoint := "1001@case.reg.example.test"

	mockCustomerDomain.EXPECT().EndpointGet(ctx, customerID, "1001").Return(endpoint, nil)
	mockDBAst.EXPECT().AstContactGetsByEndpoint(ctx, endpoint).Return(nil, errInjectedContact)

	res, err := h.ContactGetsByExtension(ctx, customerID, "1001")
	if !errors.Is(err, errInjectedContact) {
		t.Errorf("Wrong match. expect: the query failure, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: no contacts on failure, got: %v", res)
	}
}

func Test_ContactRefreshByEndpoint_cacheDeleteFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	h := &contactHandler{dbAst: mockDBAst, customerDomainHandler: mockCustomerDomain}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aa15c2c0-5652-11ee-b3a1-0b960b5e92e8")
	endpoint := "1001@case.reg.example.test"

	mockCustomerDomain.EXPECT().EndpointGet(ctx, customerID, "1001").Return(endpoint, nil)
	mockDBAst.EXPECT().AstContactDeleteFromCache(ctx, endpoint).Return(errInjectedContact)

	err := h.ContactRefreshByEndpoint(ctx, customerID, "1001")
	if !errors.Is(err, errInjectedContact) {
		t.Errorf("Wrong match. expect: the cache failure to be returned wrapped, got: %v", err)
	}
}
