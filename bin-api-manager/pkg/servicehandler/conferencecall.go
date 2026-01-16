package servicehandler

import (
	"context"
	"fmt"

	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// conferencecallGet vaildates the customer's ownership and returns the conferencecall info.
func (h *serviceHandler) conferencecallGet(ctx context.Context, id uuid.UUID) (*cfconferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "conferencecallGet",
		"conferencecall_id": id,
	})

	// send request
	res, err := h.reqHandler.ConferenceV1ConferencecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the conferencecall. err: %v", err)
		return nil, err
	}
	log.WithField("conferencecall", res).Debug("Received result.")

	return res, nil
}

// ConferencecallGet vaildates the customer's ownership and returns the conferencecall info.
func (h *serviceHandler) ConferencecallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cfconferencecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ConferencecallGet",
		"customer_id":       a.CustomerID,
		"username":          a.Username,
		"conferencecall_id": id,
	})
	log.Debugf("Get conferencecall. conferencecall_id: %s", id)

	// get conference
	tmp, err := h.conferencecallGet(ctx, id)
	if err != nil {
		log.Infof("Could not get conference info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferencecallGets gets the list of conferencecall.
// It returns list of conferencecalls if it succeed.
func (h *serviceHandler) ConferencecallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cfconferencecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConferencecallGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}

	// get conferences
	// Convert string filters to typed filters
	typedFilters, err := h.convertConferencecallFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.ConferenceV1ConferencecallList(ctx, token, size, typedFilters)
	if err != nil {
		log.Infof("Could not get conferences info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*cfconferencecall.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// ConferencecallKick is a service handler for kick the conferencecall from the conference.
func (h *serviceHandler) ConferencecallKick(ctx context.Context, a *amagent.Agent, conferencecallID uuid.UUID) (*cfconferencecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ConferencecallKick",
		"customer_id":       a.CustomerID,
		"username":          a.Username,
		"conferencecall_id": conferencecallID,
	})

	// get conference for ownership check
	c, err := h.conferencecallGet(ctx, conferencecallID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// kick the conferencecall from the conference
	tmp, err := h.reqHandler.ConferenceV1ConferencecallKick(ctx, conferencecallID)
	if err != nil {
		log.Errorf("Could not kick the call from the conference. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// convertConferencecallFilters converts map[string]string to map[cfconferencecall.Field]any
func (h *serviceHandler) convertConferencecallFilters(filters map[string]string) (map[cfconferencecall.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, cfconferencecall.Conferencecall{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[cfconferencecall.Field]any, len(typed))
	for k, v := range typed {
		result[cfconferencecall.Field(k)] = v
	}

	return result, nil
}
