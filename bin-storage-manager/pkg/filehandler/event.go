package filehandler

import (
	"context"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-storage-manager/models/file"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *fileHandler) EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting customer's all files. customer_id: %s", cu.ID)

	filters := map[file.Field]any{
		file.FieldDeleted:    false,
		file.FieldCustomerID: cu.ID,
	}

	// get files
	files, err := h.Gets(ctx, "", 10000, filters)
	if err != nil {
		log.Errorf("Could not get files. err: %v", err)
		return errors.Wrap(err, "could not get files of the customer")
	}

	for _, f := range files {
		tmp, err := h.Delete(ctx, f.ID)
		if err != nil {
			log.WithField("file", f).Errorf("Could not delete the file. err: %v", err)
			continue
		}
		log.WithField("file", tmp).Debugf("Deleted file. file_id: %s", tmp.ID)
	}

	return nil
}
