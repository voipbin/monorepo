package trunkhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

// Create creates a new trunk and returns a created trunk info
func (h *trunkHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	domainName string,
	authTypes []trunk.AuthType,
	username string,
	password string,
	allowedIPs []string,
) (*trunk.Trunk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"domain_name": domainName,
	})
	log.Debugf("Creating trunk. domain_name: %s", domainName)

	if !isValidDomainName(domainName) {
		log.Errorf("Invalid domain name. domain_name: %s", domainName)
		return nil, fmt.Errorf("invalid domain name")
	}

	// check duplicated trunk
	tmp, err := h.GetByDomainName(ctx, domainName)
	if err == nil {
		if tmp.TMDelete < dbhandler.DefaultTimeStamp {
			log.Errorf("The given trunk is already existed. err: %v", err)
			return nil, fmt.Errorf("already exists")
		}
	}

	// create new trunk
	id := h.utilHandler.UUIDCreate()
	realm := fmt.Sprintf("%s.%s", domainName, basicDomainName)
	t := &trunk.Trunk{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		DomainName: domainName,
		AuthTypes:  authTypes,

		Realm:    realm,
		Username: username,
		Password: password,

		AllowedIPs: allowedIPs,
	}

	if err := h.db.TrunkCreate(ctx, t); err != nil {
		log.Errorf("Could not create a trunk info. err: %v", err)
		return nil, err
	}
	log.Debugf("Created new trunk. id: %s, domain_name: %s", t.ID, t.DomainName)

	res, err := h.Get(ctx, t.ID)
	if err != nil {
		log.Errorf("Could not get created trunk info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, trunk.EventTypeTrunkCreated, res)
	promTrunkCreateTotal.Inc()

	return res, nil
}

// Get returns trunk
func (h *trunkHandler) Get(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {
	res, err := h.db.TrunkGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetByDomainName returns trunk of the given domain name
func (h *trunkHandler) GetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error) {
	res, err := h.db.TrunkGetByDomainName(ctx, domainName)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Gets returns list of trunks
func (h *trunkHandler) Gets(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*trunk.Trunk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
	})

	res, err := h.db.TrunkGetsByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get trunks. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Update updates the trunk info
func (h *trunkHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string, authTypes []trunk.AuthType, username string, password string, allowedIPs []string) (*trunk.Trunk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Update",
		"trunk_id":    id,
		"name":        name,
		"detail":      detail,
		"auth_type":   authTypes,
		"username":    username,
		"password":    password,
		"allowed_ips": allowedIPs,
	})

	// update
	if errUpdate := h.db.TrunkUpdateBasicInfo(ctx, id, name, detail, authTypes, username, password, allowedIPs); errUpdate != nil {
		log.Errorf("Could not update the trunk. err: %v", errUpdate)
		return nil, errUpdate
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, trunk.EventTypeTrunkUpdated, res)

	return res, nil
}

// Delete deletes the trunk info
func (h *trunkHandler) Delete(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"trunk_id": id,
	})

	// delete trunk
	if err := h.db.TrunkDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the trunk. err: %v", err)
		return nil, err
	}

	res, err := h.db.TrunkGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted trunk. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, trunk.EventTypeTrunkDeleted, res)
	promTrunkDeleteTotal.Inc()

	return res, nil
}
