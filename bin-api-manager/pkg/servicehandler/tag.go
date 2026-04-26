package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	tmtag "monorepo/bin-tag-manager/models/tag"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// tagGet validates the tag's ownership and returns the tag info.
func (h *serviceHandler) tagGet(ctx context.Context, tagID uuid.UUID) (*tmtag.Tag, error) {
	res, err := h.reqHandler.TagV1TagGet(ctx, tagID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TagCreate sends a request to agent-manager
// to creating a tag.
// it returns created tag info if it succeed.
func (h *serviceHandler) TagCreate(ctx context.Context, a *auth.AuthIdentity, name string, detail string) (*tmtag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "TagCreate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// send request
	log.Debug("Creating a new tag.")
	tmp, err := h.reqHandler.TagV1TagCreate(ctx, a.CustomerID, name, detail)
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
func (h *serviceHandler) TagGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*tmtag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "TagGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"tag_id":      id,
	})

	t, err := h.tagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// create result
	res := t.ConvertWebhookMessage()
	return res, nil
}

// TagGets sends a request to agent-manager
// to getting a list of tags.
func (h *serviceHandler) TagList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*tmtag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "TagGets",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[tmtag.Field]any{
		tmtag.FieldCustomerID: a.CustomerID,
	}
	tmp, err := h.reqHandler.TagV1TagList(ctx, token, size, filters)
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
func (h *serviceHandler) TagDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*tmtag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "TagDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"tag_id":      id,
	})

	t, err := h.tagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) TagUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name, detail string) (*tmtag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "TagUpdate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"tag_id":      id,
	})

	t, err := h.tagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not validate the tag info. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
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
