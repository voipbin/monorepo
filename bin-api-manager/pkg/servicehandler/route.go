package servicehandler

import (
	"context"
	"fmt"

	rmroute "monorepo/bin-route-manager/models/route"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// routeGet validates the route's ownership and returns the route info.
func (h *serviceHandler) routeGet(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "routeGet",
		"route_id": routeID,
	})

	// send request
	res, err := h.reqHandler.RouteV1RouteGet(ctx, routeID)
	if err != nil {
		log.Errorf("Could not get the route info. err: %v", err)
		return nil, err
	}
	log.WithField("route", res).Debug("Received result.")

	// create result
	return res, nil
}

// RouteGet sends a request to route-manager
// to getting the route.
func (h *serviceHandler) RouteGet(ctx context.Context, a *amagent.Agent, routeID uuid.UUID) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    routeID,
	})

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.routeGet(ctx, routeID)
	if err != nil {
		log.Errorf("Could not validate the route info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// RouteGets sends a request to route-manager
// to getting a list of routes.
// it returns route info if it succeed.
func (h *serviceHandler) RouteList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "RouteGets",
		"username": a.Username,
		"size":     size,
		"token":    token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get all routes (no filtering for super admin)
	filters := map[rmroute.Field]any{}
	tmps, err := h.reqHandler.RouteV1RouteList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get routes from the route-manager. err: %v", err)
		return nil, err
	}

	res := []*rmroute.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// RouteGetsByCustomerID sends a request to route-manager
// to getting a list of routes.
// it returns route info if it succeed.
func (h *serviceHandler) RouteGetsByCustomerID(ctx context.Context, a *amagent.Agent, customerID uuid.UUID, size uint64, token string) ([]*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteGetsByCustomerID",
		"customer_id": customerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[rmroute.Field]any{
		rmroute.FieldCustomerID: customerID,
	}
	tmps, err := h.reqHandler.RouteV1RouteList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get routes from the route-manager. err: %v", err)
		return nil, err
	}

	res := []*rmroute.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// RouteCreate sends a request to route-manager
// to creating an route.
// it returns created route info if it succeed.
func (h *serviceHandler) RouteCreate(
	ctx context.Context,
	a *amagent.Agent,
	customerID uuid.UUID,
	name string,
	detail string,
	providerID uuid.UUID,
	priority int,
	target string,
) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteCreate",
		"customer_id": customerID,
		"username":    a.Username,
		"provider_id": providerID,
	})

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.RouteV1RouteCreate(
		ctx,
		customerID,
		name,
		detail,
		providerID,
		priority,
		target,
	)
	if err != nil {
		log.Errorf("Could not create the route. err: %v", err)
		return nil, err
	}
	log.WithField("route", tmp).Debug("Created a new route.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// RouteDelete sends a request to route-manager
// to deleting the route.
// it returns error if it failed.
func (h *serviceHandler) RouteDelete(ctx context.Context, a *amagent.Agent, routeID uuid.UUID) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"route_id":    routeID,
	})

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	_, err := h.routeGet(ctx, routeID)
	if err != nil {
		log.Errorf("Could not get route. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.RouteV1RouteDelete(ctx, routeID)
	if err != nil {
		log.Errorf("Could not delete the route. err: %v", err)
		return nil, err
	}
	log.WithField("route", tmp).Debugf("Deleted route. route_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// RouteUpdate sends a request to route-manager
// to updating the route.
// it returns error if it failed.
func (h *serviceHandler) RouteUpdate(
	ctx context.Context,
	a *amagent.Agent,
	routeID uuid.UUID,
	name string,
	detail string,
	providerID uuid.UUID,
	priority int,
	target string,
) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteUpdate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	_, err := h.routeGet(ctx, routeID)
	if err != nil {
		log.Errorf("Could not get route. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.RouteV1RouteUpdate(ctx, routeID, name, detail, providerID, priority, target)
	if err != nil {
		log.Errorf("Could not update the route. err: %v", err)
		return nil, err
	}
	log.WithField("route", tmp).Debugf("Updated route. route_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
