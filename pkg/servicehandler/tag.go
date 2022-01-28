package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/tag"
)

// tagGet validates the tag's ownership and returns the tag info.
func (h *serviceHandler) tagGet(ctx context.Context, u *cscustomer.Customer, tagID uuid.UUID) (*tag.Tag, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "tagGet",
			"customer_id": u.ID,
			"tag_id":      tagID,
		},
	)

	// send request
	tmp, err := h.reqHandler.AMV1TagGet(ctx, tagID)
	if err != nil {
		log.Errorf("Could not get an tag. err: %v", err)
		return nil, err
	}
	log.WithField("tag", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tag.ConvertToTag(tmp)
	return res, nil
}

// TagCreate sends a request to agent-manager
// to creating a tag.
// it returns created tag info if it succeed.
func (h *serviceHandler) TagCreate(u *cscustomer.Customer, name string, detail string) (*tag.Tag, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
	})

	// send request
	log.Debug("Creating a new tag.")
	tmp, err := h.reqHandler.AMV1TagCreate(ctx, u.ID, name, detail)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}
	log.WithField("tag", tmp).Debug("Received result.")

	// create result
	res := tag.ConvertToTag(tmp)

	return res, nil
}

// AgentGet sends a request to agent-manager
// to getting a tag.
func (h *serviceHandler) TagGet(u *cscustomer.Customer, id uuid.UUID) (*tag.Tag, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"tag_id":      id,
	})

	res, err := h.tagGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// TagGets sends a request to agent-manager
// to getting a list of tags.
func (h *serviceHandler) TagGets(u *cscustomer.Customer, size uint64, token string) ([]*tag.Tag, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagGets",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	tmp, err := h.reqHandler.AMV1TagGets(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get tags.. err: %v", err)
		return nil, err
	}

	res := []*tag.Tag{}
	for _, ta := range tmp {
		t := tag.ConvertToTag(&ta)
		res = append(res, t)
	}

	return res, nil
}

// TagDelete sends a request to call-manager
// to delete the tag.
func (h *serviceHandler) TagDelete(u *cscustomer.Customer, id uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"tag_id":      id,
	})

	_, err := h.tagGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.AMV1TagDelete(ctx, id); err != nil {
		log.Infof("Could not delete the tag info. err: %v", err)
		return err
	}

	return nil
}

// TagUpdate sends a request to call-manager
// to update the tag.
func (h *serviceHandler) TagUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagUpdate",
		"customer_id": u.ID,
		"username":    u.Username,
		"tag_id":      id,
	})

	_, err := h.tagGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.AMV1TagUpdate(ctx, id, name, detail); err != nil {
		log.Infof("Could not delete the tag info. err: %v", err)
		return err
	}

	return nil
}
