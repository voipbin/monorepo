package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"
)

// routeGet validates the route's ownership and returns the route info.
func (h *serviceHandler) routeGet(ctx context.Context, u *cscustomer.Customer, routeID uuid.UUID) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "routeGet",
			"customer_id": u.ID,
			"route_id":    routeID,
		},
	)

	// send request
	tmp, err := h.reqHandler.RouteV1RouteGet(ctx, routeID)
	if err != nil {
		log.Errorf("Could not get the route info. err: %v", err)
		return nil, err
	}
	log.WithField("route", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission for this route.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// RouteGet sends a request to route-manager
// to getting the route.
func (h *serviceHandler) RouteGet(ctx context.Context, u *cscustomer.Customer, routeID uuid.UUID) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    routeID,
	})

	res, err := h.routeGet(ctx, u, routeID)
	if err != nil {
		log.Errorf("Could not validate the route info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RouteGets sends a request to route-manager
// to getting a list of routes.
// it returns route info if it succeed.
func (h *serviceHandler) RouteGets(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, size uint64, token string) ([]*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteGets",
		"customer_id": customerID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.GetCurTime()
	}

	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission for this route.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmps, err := h.reqHandler.RouteV1RouteGets(ctx, customerID, token, size)
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
	u *cscustomer.Customer,
	customerID uuid.UUID,
	providerID uuid.UUID,
	priority int,
	target string,
) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteCreate",
		"customer_id": customerID,
		"username":    u.Username,
		"provider_id": providerID,
	})

	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission for this route.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.RouteV1RouteCreate(
		ctx,
		customerID,
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
func (h *serviceHandler) RouteDelete(ctx context.Context, u *cscustomer.Customer, routeID uuid.UUID) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"route_id":    routeID,
	})

	_, err := h.routeGet(ctx, u, routeID)
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
func (h *serviceHandler) RouteUpdate(ctx context.Context, u *cscustomer.Customer, routeID, providerID uuid.UUID, priority int, target string) (*rmroute.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RouteUpdate",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	_, err := h.routeGet(ctx, u, routeID)
	if err != nil {
		log.Errorf("Could not get route. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.RouteV1RouteUpdate(ctx, routeID, providerID, priority, target)
	if err != nil {
		log.Errorf("Could not update the route. err: %v", err)
		return nil, err
	}
	log.WithField("route", tmp).Debugf("Updated route. route_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
