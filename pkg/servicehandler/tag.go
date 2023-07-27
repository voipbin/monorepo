package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	tmtag "gitlab.com/voipbin/bin-manager/tag-manager.git/models/tag"
)

// tagGet validates the tag's ownership and returns the tag info.
func (h *serviceHandler) tagGet(ctx context.Context, u *cscustomer.Customer, tagID uuid.UUID) (*tmtag.Tag, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "tagGet",
			"customer_id": u.ID,
			"tag_id":      tagID,
		},
	)

	// send request
	res, err := h.reqHandler.TagV1TagGet(ctx, tagID)
	if err != nil {
		log.Errorf("Could not get an tag. err: %v", err)
		return nil, err
	}
	log.WithField("tag", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	return res, nil
}

// TagCreate sends a request to agent-manager
// to creating a tag.
// it returns created tag info if it succeed.
func (h *serviceHandler) TagCreate(ctx context.Context, u *cscustomer.Customer, name string, detail string) (*tmtag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
	})

	// send request
	log.Debug("Creating a new tag.")
	tmp, err := h.reqHandler.TagV1TagCreate(ctx, u.ID, name, detail)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}
	log.WithField("tag", tmp).Debug("Received result.")

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentGet sends a request to agent-manager
// to getting a tag.
func (h *serviceHandler) TagGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*tmtag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"tag_id":      id,
	})

	tmp, err := h.tagGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return nil, err
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TagGets sends a request to agent-manager
// to getting a list of tags.
func (h *serviceHandler) TagGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*tmtag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagGets",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	tmp, err := h.reqHandler.TagV1TagGets(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get tags.. err: %v", err)
		return nil, err
	}

	res := []*tmtag.WebhookMessage{}
	for _, ta := range tmp {
		t := ta.ConvertWebhookMessage()
		res = append(res, t)
	}

	return res, nil
}

// TagDelete sends a request to call-manager
// to delete the tag.
func (h *serviceHandler) TagDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*tmtag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"tag_id":      id,
	})

	_, err := h.tagGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.TagV1TagDelete(ctx, id)
	if err != nil {
		log.Infof("Could not delete the tag info. err: %v", err)
		return nil, err
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil

}

// TagUpdate sends a request to call-manager
// to update the tag.
func (h *serviceHandler) TagUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*tmtag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TagUpdate",
		"customer_id": u.ID,
		"username":    u.Username,
		"tag_id":      id,
	})

	_, err := h.tagGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.TagV1TagUpdate(ctx, id, name, detail)
	if err != nil {
		log.Infof("Could not delete the tag info. err: %v", err)
		return nil, err
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}
