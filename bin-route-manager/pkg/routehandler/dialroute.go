package routehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/route"
)

// DialrouteList returns routes for dialing.
//
// When targetProviderIDs is non-empty, bypasses the normal customer/default
// route merging and returns synthetic Route entries — one per provider ID,
// in array order, with Route.ID = ProviderID and Priority = array index.
// When nil/empty, the normal merge behavior is preserved.
func (h *routeHandler) DialrouteList(
	ctx context.Context,
	customerID uuid.UUID,
	target string,
	targetProviderIDs []uuid.UUID,
) ([]*route.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "DialrouteList",
		"customer_id":         customerID,
		"target":              target,
		"target_provider_ids": targetProviderIDs,
	})

	// Override: when targetProviderIDs is set, return synthetic routes in order.
	if len(targetProviderIDs) > 0 {
		// Validate provider existence before constructing synthetic routes.
		// Fail fast so the admin gets a clear error instead of a silent mid-dial hangup.
		for _, pid := range targetProviderIDs {
			if _, err := h.db.ProviderGet(ctx, pid); err != nil {
				log.Errorf("Could not get provider for synthetic dialroute. provider_id: %s, err: %v", pid, err)
				return nil, errors.Wrapf(err, "provider not found: %s", pid)
			}
		}

		// Use ProviderID as the synthetic Route.ID so call-manager's failover
		// tracking uniquely identifies each route (matches by route.ID == c.DialrouteID).
		res := make([]*route.Route, 0, len(targetProviderIDs))
		for i, pid := range targetProviderIDs {
			res = append(res, &route.Route{
				ID:         pid,
				CustomerID: customerID,
				ProviderID: pid,
				Target:     target,
				Priority:   i,
				Name:       "synthetic-route",
				Detail:     "Synthetic route generated for route_provider_ids override. Not persisted.",
			})
		}
		log.WithField("synthetic_routes", res).Info("Returning synthetic dialroutes for provider override")
		return res, nil
	}

	// get customer based route
	customerRoutes, err := h.ListByTarget(ctx, customerID, target)
	if err != nil {
		return nil, errors.Wrap(err, "could not get customer routes")
	}
	log.WithField("customer_route", customerRoutes).Debugf("Found customer routes")

	// get default based route
	defaultRoutes, err := h.ListByTarget(ctx, route.CustomerIDBasicRoute, target)
	if err != nil {
		return nil, errors.Wrap(err, "could not get default routes")
	}
	log.WithField("default_route", defaultRoutes).Debugf("Found default routes")

	res := customerRoutes
	for _, r := range defaultRoutes {
		exist := false
		for _, rr := range customerRoutes {
			if r.ProviderID == rr.ProviderID {
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
