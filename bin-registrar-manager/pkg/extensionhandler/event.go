package extensionhandler

import (
	"context"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/models/extension"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *extensionHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all extension of the customer. customer_id: %s", cu.ID)

	// get all extensions in customer
	filters := map[extension.Field]any{
		extension.FieldCustomerID: cu.ID,
		extension.FieldDeleted:    false,
	}
	extensions, err := h.List(ctx, "", 1000, filters)
	if err != nil {
		log.Errorf("Could not gets extensions list. err: %v", err)
		return errors.Wrap(err, "could not get extensions list")
	}

	// delete all extensions
	for _, e := range extensions {
		log.Debugf("Deleting extension info. extension_id: %s", e.ID)
		tmp, err := h.Delete(ctx, e.ID)
		if err != nil {
			log.Errorf("Could not delete extension info. err: %v", err)
			continue
		}
		log.WithField("extension", tmp).Debugf("Deleted extension info. extension_id: %s", tmp.ID)
	}

	return nil
}
