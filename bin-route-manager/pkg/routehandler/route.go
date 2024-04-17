package routehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/route"
)

// Get returns route
func (h *routeHandler) Get(ctx context.Context, id uuid.UUID) (*route.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
		"id":   id,
	})

	res, err := h.db.RouteGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get route. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new route
func (h *routeHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	providerID uuid.UUID,
	priority int,
	target string,
) (*route.Route, error) {
	log := logrus.WithField("func", "Create")

	id := uuid.Must(uuid.NewV4())
	r := &route.Route{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		ProviderID: providerID,
		Priority:   priority,

		Target: target,
	}
	log.WithField("route", r).Debug("Creating a new route.")

	if errCreate := h.db.RouteCreate(ctx, r); errCreate != nil {
		log.Errorf("Could not create the route. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.RouteGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created route info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, route.EventTypeRouteCreated, res)

	return res, nil
}

// GetsByCustomerID returns list of routes of the given customerID
func (h *routeHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*route.Route, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCustomerID",
			"customer_id": customerID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting routes.")

	var res []*route.Route
	var err error

	if customerID == uuid.Nil {
		res, err = h.db.RouteGets(ctx, token, limit)
	} else {
		res, err = h.db.RouteGetsByCustomerID(ctx, customerID, token, limit)
	}

	if err != nil {
		log.Errorf("Could not get routes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RouteGetsByCustomerID returns list of routes
func (h *routeHandler) GetsByTarget(ctx context.Context, customerID uuid.UUID, target string) ([]*route.Route, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "RouteGetsByTarget",
			"customer_id": customerID,
			"target":      target,
		})
	log.Debug("Getting routes.")

	routeTargets, err := h.db.RouteGetsByCustomerIDWithTarget(ctx, customerID, target)
	if err != nil {
		log.Errorf("Could not get routes for target. err: %v", err)
		return nil, err
	}

	routeAll, err := h.db.RouteGetsByCustomerIDWithTarget(ctx, customerID, route.TargetAll)
	if err != nil {
		log.Errorf("Could not get routes for all target. err: %v", err)
	}

	res := routeTargets
	for _, r := range routeAll {
		exist := false
		for _, rr := range res {
			if rr.ProviderID == r.ProviderID {
				exist = true
				break
			}

		}
		if !exist {
			res = append(res, r)
		}
	}

	return res, nil
}

// Delete deletes the route
func (h *routeHandler) Delete(ctx context.Context, id uuid.UUID) (*route.Route, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "Delete",
			"route_id": id,
		},
	)
	log.Debug("Deleting the route.")

	err := h.db.RouteDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the route. err: %v", err)
		return nil, err
	}

	res, err := h.db.RouteGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted route. err: %v", err)
		return nil, errors.Wrap(err, "could not get deleted route")
	}
	h.notifyHandler.PublishEvent(ctx, route.EventTypeRouteDeleted, res)

	return res, nil
}

// Update updates the route and return the updated route
func (h *routeHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string, providerID uuid.UUID, priority int, target string) (*route.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Update",
		"route_id": id,
	})

	log.WithFields(logrus.Fields{
		"name":        name,
		"detail":      detail,
		"provider_id": providerID,
		"priority":    priority,
		"target":      target,
	}).Debug("Updating the route.")

	if errUpdate := h.db.RouteUpdate(ctx, id, name, detail, providerID, priority, target); errUpdate != nil {
		log.Errorf("Could not update the route info. err: %v", errUpdate)
		return nil, errors.Wrap(errUpdate, "could not update the route info")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated route info. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated route info")
	}
	h.notifyHandler.PublishEvent(ctx, route.EventTypeRouteUpdated, res)

	return res, nil
}
