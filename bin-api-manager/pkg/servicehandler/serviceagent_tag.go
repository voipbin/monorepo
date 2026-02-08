package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmtag "monorepo/bin-tag-manager/models/tag"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentTagList returns a list of tags for the service agent's customer.
func (h *serviceHandler) ServiceAgentTagList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmtag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentTagList",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[tmtag.Field]any{
		tmtag.FieldCustomerID: a.CustomerID,
	}
	tmps, err := h.reqHandler.TagV1TagList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get tags. err: %v", err)
		return nil, err
	}

	res := []*tmtag.WebhookMessage{}
	for _, ta := range tmps {
		t := ta.ConvertWebhookMessage()
		res = append(res, t)
	}

	return res, nil
}

// ServiceAgentTagGet returns the given tag info.
func (h *serviceHandler) ServiceAgentTagGet(ctx context.Context, a *amagent.Agent, tagID uuid.UUID) (*tmtag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentTagGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"tag_id":      tagID,
	})

	t, err := h.tagGet(ctx, tagID)
	if err != nil {
		log.Errorf("Could not get tag info. err: %v", err)
		return nil, err
	}
	log.WithField("tag", t).Debugf("Retrieved tag info. tag_id: %s", t.ID)

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := t.ConvertWebhookMessage()
	return res, nil
}
