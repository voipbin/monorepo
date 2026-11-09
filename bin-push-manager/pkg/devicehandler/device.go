package devicehandler

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/pkg/dbhandler"
)

// newInvalidArgument returns the typed 400 error.
func newInvalidArgument(reason, message string) error {
	return cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, reason, message)
}

// newNotFound returns the typed 404 error.
func newNotFound() error {
	return cerrors.NotFound(commonoutline.ServiceNamePushManager, "DEVICE_NOT_FOUND", "The device was not found.")
}

// Register registers the token for the agent.
//
// Three idempotent steps without a transaction, arbitrated by the unique key of the token: (1) delete the
// row of this token when its owner is not this agent, (2) when the agent has the token already refresh it,
// (3) insert. A duplicate key error on the insert means a concurrent registration, so it starts again from
// step 1, at most maxRegisterTries times.
func (h *deviceHandler) Register(ctx context.Context, customerID, agentID uuid.UUID, platform device.Platform, token string) (*device.Device, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Register",
		"customer_id": customerID,
		"agent_id":    agentID,
	})

	if customerID == uuid.Nil || agentID == uuid.Nil {
		return nil, newInvalidArgument("INVALID_OWNER", "The customer id and the agent id are required.")
	}
	if !device.IsValidPlatform(platform) {
		return nil, newInvalidArgument("INVALID_PLATFORM", "The platform is not supported.")
	}
	if token == "" || len(token) > device.MaxTokenLength {
		return nil, newInvalidArgument("INVALID_TOKEN", "The token is empty or too long.")
	}

	for i := 0; i < maxRegisterTries; i++ {
		res, err := h.registerOnce(ctx, customerID, agentID, platform, token)
		if err == nil {
			h.trimOldest(ctx, customerID, agentID)
			return res, nil
		}
		if !errors.Is(err, dbhandler.ErrDuplicate) {
			log.Errorf("Could not register the device. err: %v", err)
			return nil, err
		}
		log.Debugf("Lost the registration race. try: %d", i+1)
	}

	return nil, fmt.Errorf("could not register the device after %d tries", maxRegisterTries)
}

// registerOnce runs the three steps of Register once.
func (h *deviceHandler) registerOnce(ctx context.Context, customerID, agentID uuid.UUID, platform device.Platform, token string) (*device.Device, error) {
	// 1. a token has one owner
	if err := h.db.DeviceDeleteByTokenNotOwner(ctx, token, customerID, agentID); err != nil {
		return nil, fmt.Errorf("could not delete the previous owner of the token. err: %w", err)
	}

	// 2. refresh
	cur, err := h.db.DeviceGetByToken(ctx, token)
	switch {
	case err == nil:
		// a racing registration of another owner inserted the token after step 1: start again, the
		// last registration wins
		if cur.CustomerID != customerID || cur.AgentID != agentID {
			return nil, dbhandler.ErrDuplicate
		}
		if errTouch := h.db.DeviceTouch(ctx, cur.ID); errTouch != nil {
			return nil, fmt.Errorf("could not refresh the device. err: %w", errTouch)
		}
		res, errGet := h.db.DeviceGet(ctx, cur.ID)
		if errors.Is(errGet, dbhandler.ErrNotFound) {
			// the row was removed by a racing registration of another owner: start again
			return nil, dbhandler.ErrDuplicate
		}
		return res, errGet
	case !errors.Is(err, dbhandler.ErrNotFound):
		return nil, fmt.Errorf("could not get the device by the token. err: %w", err)
	}

	// 3. insert
	d := &device.Device{}
	d.ID = h.utilHandler.UUIDCreate()
	d.CustomerID = customerID
	d.AgentID = agentID
	d.Platform = platform
	d.Token = token
	if errCreate := h.db.DeviceCreate(ctx, d); errCreate != nil {
		return nil, errCreate
	}

	return h.db.DeviceGet(ctx, d.ID)
}

// trimOldest deletes the oldest devices of the agent above the cap. It is best effort: a failure is logged.
func (h *deviceHandler) trimOldest(ctx context.Context, customerID, agentID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "trimOldest",
		"agent_id": agentID,
	})

	filters := map[device.Field]any{
		device.FieldCustomerID: customerID,
		device.FieldAgentID:    agentID,
	}
	list, err := h.db.DeviceList(ctx, 1000, listPageToken, filters)
	if err != nil {
		log.Errorf("Could not list the devices. err: %v", err)
		return
	}
	if len(list) <= maxDevicesPerAgent {
		return
	}

	// the oldest refresh first
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].TMUpdate.Before(*list[j].TMUpdate)
	})
	for _, d := range list[:len(list)-maxDevicesPerAgent] {
		if errDelete := h.db.DeviceDelete(ctx, d.ID); errDelete != nil {
			log.Errorf("Could not delete the old device. device_id: %s, err: %v", d.ID, errDelete)
		}
	}
}

// Get returns the device.
func (h *deviceHandler) Get(ctx context.Context, id uuid.UUID) (*device.Device, error) {
	res, err := h.db.DeviceGet(ctx, id)
	if err != nil {
		if errors.Is(err, dbhandler.ErrNotFound) {
			return nil, newNotFound()
		}
		return nil, err
	}

	return res, nil
}

// Delete deletes the device and returns it.
func (h *deviceHandler) Delete(ctx context.Context, id uuid.UUID) (*device.Device, error) {
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if errDelete := h.db.DeviceDelete(ctx, id); errDelete != nil {
		return nil, fmt.Errorf("could not delete the device. err: %w", errDelete)
	}

	return res, nil
}

// List returns the live devices that match the filters.
func (h *deviceHandler) List(ctx context.Context, size uint64, token string, filters map[device.Field]any) ([]*device.Device, error) {
	list, err := h.db.DeviceList(ctx, size, token, filters)
	if err != nil {
		return nil, err
	}

	now := h.now()
	res := []*device.Device{}
	for _, d := range list {
		if d.IsLive(now) {
			res = append(res, d)
		}
	}

	return res, nil
}
