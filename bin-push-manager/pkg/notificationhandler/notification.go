package notificationhandler

import (
	"context"
	"errors"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/pkg/dbhandler"
	"monorepo/bin-push-manager/pkg/pushhandler"
)

// Send sends the notification to the live devices of the agent, or to the one device when deviceID is set.
// A device of another agent or customer is not addressed: the result has no device then.
func (h *notificationHandler) Send(ctx context.Context, customerID, agentID, deviceID uuid.UUID, kind notification.Kind, data map[string]string) (*notification.SendResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Send",
		"customer_id": customerID,
		"agent_id":    agentID,
		"device_id":   deviceID,
		"kind":        kind,
	})

	if customerID == uuid.Nil || agentID == uuid.Nil {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_OWNER", "The customer id and the agent id are required.")
	}
	if !notification.IsValidKind(kind) {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_KIND", "The notification kind is not supported.")
	}
	if _, ok := data[notification.DataKeyKind]; ok {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_DATA", "The data must not contain the key kind.")
	}

	devices, err := h.targets(ctx, customerID, agentID, deviceID)
	if err != nil {
		log.Errorf("Could not get the devices. err: %v", err)
		return nil, err
	}

	res := &notification.SendResult{DeviceCount: len(devices)}

	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, d := range devices {
		wg.Add(1)
		go func(d *device.Device) {
			defer wg.Done()

			if h.deliver(ctx, d, kind, data) {
				mu.Lock()
				res.SentCount++
				mu.Unlock()
			}
		}(d)
	}
	wg.Wait()

	return res, nil
}

// targets returns the live devices the notification is addressed to.
func (h *notificationHandler) targets(ctx context.Context, customerID, agentID, deviceID uuid.UUID) ([]*device.Device, error) {
	now := h.now()

	if deviceID != uuid.Nil {
		d, err := h.db.DeviceGet(ctx, deviceID)
		if err != nil {
			if errors.Is(err, dbhandler.ErrNotFound) {
				return nil, nil
			}
			return nil, err
		}

		if d.CustomerID != customerID || d.AgentID != agentID || !d.IsLive(now) {
			return nil, nil
		}

		return []*device.Device{d}, nil
	}

	filters := map[device.Field]any{
		device.FieldCustomerID: customerID,
		device.FieldAgentID:    agentID,
	}
	list, err := h.db.DeviceList(ctx, listSize, listPageToken, filters)
	if err != nil {
		return nil, err
	}

	res := []*device.Device{}
	for _, d := range list {
		if d.IsLive(now) {
			res = append(res, d)
		}
	}

	return res, nil
}

// deliver sends the notification to the device and returns true if the provider accepted it. A token that
// the provider reports as unregistered deletes the device.
func (h *notificationHandler) deliver(ctx context.Context, d *device.Device, kind notification.Kind, data map[string]string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":      "deliver",
		"device_id": d.ID,
		"kind":      kind,
	})

	cctx, cancel := context.WithTimeout(ctx, deliveryTimeout)
	defer cancel()

	err := h.sender.Send(cctx, d.Token, kind, data)
	switch {
	case err == nil:
		promNotificationTotal.WithLabelValues(string(kind), resultSent).Inc()
		return true

	case errors.Is(err, pushhandler.ErrNotConfigured):
		promNotificationTotal.WithLabelValues(string(kind), resultNotConfigured).Inc()

	case errors.Is(err, pushhandler.ErrUnregistered):
		promNotificationTotal.WithLabelValues(string(kind), resultUnregistered).Inc()
		if errDelete := h.db.DeviceDelete(ctx, d.ID); errDelete != nil {
			log.Errorf("Could not delete the unregistered device. err: %v", errDelete)
		}

	default:
		promNotificationTotal.WithLabelValues(string(kind), resultFailed).Inc()
		log.Errorf("Could not deliver the notification. err: %v", err)
	}

	return false
}
