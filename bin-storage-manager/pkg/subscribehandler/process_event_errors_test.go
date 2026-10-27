package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

var testCustomerID = uuid.FromStringOrNil("17ce05c9-0000-0000-0000-000000000002")

// Test_processEvent_handlerFailureStopsTheCascade checks that a failing file cleanup on
// customer_deleted is only logged by processEvent and that the account cleanup, which
// follows it, is not attempted. The account handler mock has no expectation, so a call
// to it fails the test.
func Test_processEvent_handlerFailureStopsTheCascade(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockFile := filehandler.NewMockFileHandler(mc)
	h := &subscribeHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		accountHandler: accounthandler.NewMockAccountHandler(mc),
		fileHandler:    mockFile,
	}

	mockFile.EXPECT().EventCustomerDeleted(gomock.Any(), &cmcustomer.Customer{ID: testCustomerID}).Return(errTest)

	h.processEvent(&sock.Event{
		Publisher: string(commonoutline.ServiceNameCustomerManager),
		Type:      cmcustomer.EventTypeCustomerDeleted,
		Data:      []byte(`{"id":"` + testCustomerID.String() + `"}`),
	})
}
