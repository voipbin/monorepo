package schedulehandler

import (
	"context"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-scheduler-manager/models/schedule"
)

// eventListLimit bounds the number of schedules fetched for an event-driven
// bulk operation.
const eventListLimit = 1000

// EventCustomerDeleted handles the customer-manager's customer_deleted event:
// it soft-deletes every active schedule owned by the deleted customer.
// Nil-customer platform schedules are never touched (design §6): a nil
// customer id in the event is refused instead of matching the platform rows.
func (h *scheduleHandler) EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": c,
	})

	if c == nil || c.ID == uuid.Nil {
		return errors.New("customer id is empty — refusing to delete nil-customer platform schedules")
	}

	filters := map[schedule.Field]any{
		schedule.FieldCustomerID: c.ID,
		schedule.FieldDeleted:    false,
	}

	schedules, err := h.dbList(ctx, eventListLimit, h.utilHandler.TimeGetCurTime(), filters)
	if err != nil {
		log.Errorf("Could not get the customer's schedules. err: %v", err)
		return errors.Wrap(err, "could not get the customer's schedules")
	}

	for _, s := range schedules {
		tmp, errDelete := h.Delete(ctx, s.ID)
		if errDelete != nil {
			log.Errorf("Could not delete the schedule. schedule_id: %s, err: %v", s.ID, errDelete)
			continue
		}

		log.WithField("schedule", tmp).Debugf("Deleted schedule. schedule_id: %s", tmp.ID)
	}

	return nil
}
