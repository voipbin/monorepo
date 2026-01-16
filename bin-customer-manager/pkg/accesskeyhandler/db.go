package accesskeyhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

// List returns list of accesskeys
func (h *accesskeyHandler) List(ctx context.Context, size uint64, token string, filters map[accesskey.Field]any) ([]*accesskey.Accesskey, error) {
	log := logrus.WithField("func", "List")

	res, err := h.db.AccesskeyList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get accesskey info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ListByCustomerID returns list of accesskeys by the customer id
func (h *accesskeyHandler) GetsByCustomerID(ctx context.Context, size uint64, token string, customerID uuid.UUID) ([]*accesskey.Accesskey, error) {
	log := logrus.WithField("func", "List")

	filter := map[accesskey.Field]any{
		accesskey.FieldCustomerID: customerID,
	}

	res, err := h.db.AccesskeyList(ctx, size, token, filter)
	if err != nil {
		log.Errorf("Could not get accesskey info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns accesskey info.
func (h *accesskeyHandler) Get(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.AccesskeyGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get accesskey info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByToken returns accesskey info.
func (h *accesskeyHandler) GetByToken(ctx context.Context, token string) (*accesskey.Accesskey, error) {
	log := logrus.WithField("func", "GetByToken")

	filter := map[accesskey.Field]any{
		accesskey.FieldToken: token,
	}

	tmp, err := h.db.AccesskeyList(ctx, 100, "", filter)
	if err != nil || len(tmp) == 0 || len(tmp) > 1 {
		log.Errorf("Could not get accesskeys info. err: %v", err)
		return nil, err
	}

	res := tmp[0]
	return res, nil
}

// Create creates a new accesskey.
func (h *accesskeyHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	expire time.Duration,
) (*accesskey.Accesskey, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Create",
		"name":   name,
		"detail": detail,
		"expire": expire,
	})
	log.Debug("Creating a new accesskey.")

	id := h.utilHandler.UUIDCreate()
	tmExpire := h.utilHandler.TimeGetCurTimeAdd(expire)

	token, err := h.utilHandler.StringGenerateRandom(defaultLenToken)
	if err != nil {
		log.Errorf("Could not generate the token. err: %v", err)
	}

	a := &accesskey.Accesskey{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		Token: token,

		TMExpire: tmExpire,
	}

	if err := h.db.AccesskeyCreate(ctx, a); err != nil {
		log.Errorf("Could not create a new customer. err: %v", err)
		return nil, err
	}

	res, err := h.db.AccesskeyGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created customer info. err: %v", err)
		return nil, err
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, accesskey.EventTypeAccesskeyCreated, res)

	return res, nil
}

// Delete deletes the accesskey.
func (h *accesskeyHandler) Delete(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Delete",
		"accesskey_id": id,
	})
	log.Debug("Deleteing the accesskey.")

	// get accesskey info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get accesskey info. err: %v", err)
		return nil, err
	}

	if c.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		log.Infof("The accesskey already deleted. accesskey_id: %s", c.ID)
		return c, nil
	}

	if err := h.db.AccesskeyDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the accesskey. err: %v", err)
		return nil, err
	}

	// get deleted accesskey
	res, err := h.db.AccesskeyGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted accesskey. err: %v", err)
		return nil, fmt.Errorf("could not get deleted accesskey. err: %v", err)
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, accesskey.EventTypeAccesskeyDeleted, res)

	return res, nil
}

// UpdateBasicInfo updates the accesskey's basic info.
func (h *accesskeyHandler) UpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
) (*accesskey.Accesskey, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateBasicInfo",
		"accesskey_id": id,
	})
	log.Debug("Updating the accesskey's basic info.")

	fields := map[accesskey.Field]any{
		accesskey.FieldName:   name,
		accesskey.FieldDetail: detail,
	}

	if err := h.db.AccesskeyUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the basic info. err: %v", err)
		return nil, err
	}

	// get updated accesskey
	res, err := h.db.AccesskeyGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated accesskey. err: %v", err)
		return nil, fmt.Errorf("could not get updated accesskey")
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, accesskey.EventTypeAccesskeyUpdated, res)

	return res, nil
}
