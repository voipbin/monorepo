package subscribehandler

import (
	"fmt"
	"strings"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/numberhandler"
)

// Test_processEvent_handlerErrorIsLogged checks that processEvent reports a handler failure in
// the log with the publisher and the type, and that it does not panic or return it.
func Test_processEvent_handlerErrorIsLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNumber := numberhandler.NewMockNumberHandler(mc)
	h := subscribeHandler{
		sockHandler:   sockhandler.NewMockSockHandler(mc),
		numberHandler: mockNumber,
	}

	std := logrus.StandardLogger()
	hook := logrustest.NewLocal(logrus.New())
	origHooks := std.ReplaceHooks(make(logrus.LevelHooks))
	std.AddHook(hook)
	t.Cleanup(func() { std.ReplaceHooks(origHooks) })

	customer := &cmcustomer.Customer{ID: uuid.FromStringOrNil("12345678-1234-1234-1234-123456789012")}
	mockNumber.EXPECT().EventCustomerDeleted(gomock.Any(), customer).Return(fmt.Errorf("handler failure"))

	h.processEvent(&sock.Event{
		Publisher: string(commonoutline.ServiceNameCustomerManager),
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		DataType:  "application/json",
		Data:      []byte(`{"id":"12345678-1234-1234-1234-123456789012"}`),
	})

	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel && strings.Contains(e.Message, "Could not process the event correctly") &&
			strings.Contains(e.Message, "customer_deleted") && strings.Contains(e.Message, "handler failure") {
			found = true
		}
	}
	if !found {
		t.Errorf("Wrong match. expect an error log with the type and the cause, got: %v", hook.AllEntries())
	}
}
