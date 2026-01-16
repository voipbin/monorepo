package routehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/route"
)

// DialrouteList returns routes for dialing
func (h *routeHandler) DialrouteList(ctx context.Context, customerID uuid.UUID, target string) ([]*route.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DialrouteGets",
		"customer_id": customerID,
		"target":      target,
	})

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
