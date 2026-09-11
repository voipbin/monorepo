package routehandler

import (
	"context"
	stderrors "errors"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/dbhandler"
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
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameRouteManager,
				"ROUTE_NOT_FOUND",
				"The route was not found.",
			).Wrap(err)
		}
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

// ListByCustomerID returns list of routes of the given customerID
func (h *routeHandler) ListByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*route.Route, error) {
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

	filters := map[route.Field]any{}
	if customerID != uuid.Nil {
		filters[route.FieldCustomerID] = customerID
	}

	res, err = h.db.RouteList(ctx, token, limit, filters)
	if err != nil {
		log.Errorf("Could not get routes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ListByTarget returns list of routes
func (h *routeHandler) ListByTarget(ctx context.Context, customerID uuid.UUID, target string) ([]*route.Route, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByTarget",
			"customer_id": customerID,
			"target":      target,
		})
	log.Debug("Getting routes.")

	// Get routes for specific target
	filtersTarget := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     target,
	}
	routeTargets, err := h.db.RouteList(ctx, "", 1000, filtersTarget)
	if err != nil {
		log.Errorf("Could not get routes for target. err: %v", err)
		return nil, err
	}

	// Get routes for "all" target
	filtersAll := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}
	routeAll, err := h.db.RouteList(ctx, "", 1000, filtersAll)
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
//
// Every field is a pointer: nil means "leave the existing value
// untouched", a non-nil pointer (including one pointing at the zero
// value, e.g. priority: 0) means "set to exactly this value". See
// docs/plans/2026-09-12-route-put-partial-update-phase3-design.md.
func (h *routeHandler) Update(ctx context.Context, id uuid.UUID, name *string, detail *string, providerID *uuid.UUID, priority *int, target *string) (*route.Route, error) {
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

	fields := map[route.Field]any{}
	if name != nil {
		fields[route.FieldName] = *name
	}
	if detail != nil {
		fields[route.FieldDetail] = *detail
	}
	if providerID != nil {
		fields[route.FieldProviderID] = *providerID
	}
	if priority != nil {
		fields[route.FieldPriority] = *priority
	}
	if target != nil {
		fields[route.FieldTarget] = *target
	}

	if len(fields) == 0 {
		// A PUT with every field omitted is a client no-op, not a server
		// error. Reuse Get's existing ErrNotFound -> cerrors.NotFound
		// mapping instead of duplicating it here.
		return h.Get(ctx, id)
	}

	if errUpdate := h.db.RouteUpdate(ctx, id, fields); errUpdate != nil {
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
