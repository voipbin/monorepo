package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

// processEventCUCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCUCustomerDeleted(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCUCustomerDeleted",
		"event": m,
	})

	cu := &cucustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.Debugf("Executing the event handler.")

	if errTranscribe := h.transcribeHandler.EventCUCustomerDeleted(ctx, cu); errTranscribe != nil {
		log.Errorf("Could not handle the event correctly. the transcribe handler returned an error. err: %v", errTranscribe)
	}

	return nil
}
