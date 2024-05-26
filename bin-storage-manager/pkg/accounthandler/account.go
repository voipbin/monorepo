package accounthandler

import (
	"context"
	"fmt"
	"monorepo/bin-storage-manager/models/account"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Create Creates the account and returns the created account
func (h *accountHandler) Create(ctx context.Context, customerID uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	tmp, err := h.getByCustomerID(ctx, customerID)
	if err == nil || tmp != nil {
		log.WithField("account", tmp).Errorf("The customer already has an account. account_id: %v", tmp.ID)
		return nil, fmt.Errorf("the customer already has an account")
	}

	id := h.utilHandler.UUIDCreate()
	f := &account.Account{
		ID:             id,
		CustomerID:     customerID,
		TotalFileCount: 0,
		TotalFileSize:  0,
	}

	if errCreate := h.db.AccountCreate(ctx, f); errCreate != nil {
		log.Errorf("Could not create account. err: %v", errCreate)
		return nil, errCreate
	}

	// get created resource
	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created account info. err: %v", err)
		return nil, err
	}
	log.WithField("account", res).Debugf("Created account info. id: %s", res.ID)

	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountCreated, res)

	return res, nil
}

// Get returns account info
func (h *accountHandler) Get(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get file info")
	}

	return res, nil
}

// getByCustomerID returns given customer's account
func (h *accountHandler) getByCustomerID(ctx context.Context, customerID uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getByCustomerID",
		"customer_id": customerID,
	})

	// check the account exists
	filters := map[string]string{
		"deleted":     "false",
		"customer_id": customerID.String(),
	}
	tmps, err := h.Gets(ctx, "", 1, filters)
	if err != nil {
		log.Errorf("Could not check the accounts. err: %v", err)
		return nil, err
	}
	if len(tmps) == 0 {
		return nil, fmt.Errorf("no account found")
	}

	res := tmps[0]
	return res, nil
}

// Gets returns list of account
func (h *accountHandler) Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Gets",
		"token": token,
		"size":  size,
		"limit": size,
	})

	res, err := h.db.AccountGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get accounts. err: %v", err)
		return nil, err
	}

	return res, nil
}

// DeleteForce deletes the given account from the bucket
func (h *accountHandler) Delete(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
		"id":   id,
	})

	if errDelete := h.db.AccountDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the account. err: %v", errDelete)
		return nil, errDelete
	}

	// get deleted info
	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted account info. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountDeleted, res)

	return res, nil
}

// IncreaseFileInfo increases the account's file info and returns the updated account
func (h *accountHandler) IncreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "IncreaseFileInfo",
		"id":        id,
		"filecount": filecount,
		"filesize":  filesize,
	})

	if errIncrease := h.db.AccountIncreaseFileInfo(ctx, id, filecount, filesize); errIncrease != nil {
		log.Errorf("Could not increase the account file info. err: %v", errIncrease)
		return nil, errIncrease
	}

	// get updated info
	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted account info. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountUpdated, res)

	return res, nil
}

// DecreaseFileInfo decreases the account's file info and returns the updated account
func (h *accountHandler) DecreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "DecreaseFileInfo",
		"id":        id,
		"filecount": filecount,
		"filesize":  filesize,
	})

	if errDecrease := h.db.AccountDecreaseFileInfo(ctx, id, filecount, filesize); errDecrease != nil {
		log.Errorf("Could not decrease the account file info. err: %v", errDecrease)
		return nil, errDecrease
	}

	// get updated info
	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted account info. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountUpdated, res)

	return res, nil
}

// ValidateFileInfoByCustomerID validates the given file info of the given customer id
// it returns account if it is vliad
func (h *accountHandler) ValidateFileInfoByCustomerID(ctx context.Context, customerID uuid.UUID, filecount int64, filesize int64) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateFileInfoByCustomerID",
		"customer_id": customerID,
		"filecount":   filecount,
		"filesize":    filesize,
	})

	// get account
	filters := map[string]string{
		"deleted":     "false",
		"customer_id": customerID.String(),
	}

	tmps, err := h.Gets(ctx, "", 1, filters)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return nil, err
	}

	if len(tmps) == 0 {
		log.Errorf("Could not find account info. err: %v", err)
		return nil, fmt.Errorf("could not find account info")
	}

	res := tmps[0]
	if res.TotalFileSize+filesize > maxFileSize {
		log.Debugf("Exceeded max total file size limit. total_file_size: %d", res.TotalFileSize)
		return nil, fmt.Errorf("exceeded max total file size limit")
	}

	return res, nil
}
