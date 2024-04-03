package subscribehandler

import (
	"context"
	"encoding/json"

	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCUCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCUCustomerDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerDeleted",
		"event": m,
	})

	cu := &cucustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.Debugf("Executing the event handler.")

	if errCall := h.callHandler.EventCUCustomerDeleted(ctx, cu); errCall != nil {
		log.Errorf("Could not handle the event correctly. The call handler returned an error. err: %v", errCall)
	}

	if errGroup := h.groupcallHandler.EventCUCustomerDeleted(ctx, cu); errGroup != nil {
		log.Errorf("Could not handle the event correctly. The groupcall handler returned an error. err: %v", errGroup)
	}

	if errConfbridge := h.confbridgeHandler.EventCUCustomerDeleted(ctx, cu); errConfbridge != nil {
		log.Errorf("Could not handle the event correctly. The confbridge handler returned an error. err: %v", errConfbridge)
	}
	return nil
}
