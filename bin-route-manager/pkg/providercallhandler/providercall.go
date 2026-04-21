package providercallhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/providercall"
)

// Create creates a new providercall record capturing the admin's request info
// (customer, provider, flow, source, destinations, anonymous) plus the IDs
// of the calls/groupcalls that were already created by the caller via
// CallV1CallsCreate. The caller is bin-api-manager's provider-call handler.
func (h *providerCallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	providerID uuid.UUID,
	flowID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	anonymous string,
	callIDs []uuid.UUID,
	groupcallIDs []uuid.UUID,
) (*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"provider_id": providerID,
	})

	id := uuid.Must(uuid.NewV4())
	p := &providercall.ProviderCall{
		ID:           id,
		CustomerID:   customerID,
		ProviderID:   providerID,
		FlowID:       flowID,
		Source:       source,
		Destinations: destinations,
		Anonymous:    anonymous,
		CallIDs:      callIDs,
		GroupcallIDs: groupcallIDs,
	}
	log.WithField("providercall", p).Debugf("Creating a new providercall. id: %s", id)

	if errCreate := h.db.ProviderCallCreate(ctx, p); errCreate != nil {
		log.Errorf("Could not create the providercall. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "could not create the providercall")
	}

	res, err := h.db.ProviderCallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created providercall info. err: %v", err)
		return nil, errors.Wrap(err, "could not get created providercall info")
	}
	h.notifyHandler.PublishEvent(ctx, providercall.EventTypeProviderCallCreated, res)

	return res, nil
}

// Get returns a single providercall by id.
func (h *providerCallHandler) Get(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
		"id":   id,
	})

	res, err := h.db.ProviderCallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get providercall. err: %v", err)
		return nil, err
	}
	log.WithField("providercall", res).Debugf("Retrieved providercall. id: %s", res.ID)

	return res, nil
}

// List returns a paginated list of providercalls.
func (h *providerCallHandler) List(ctx context.Context, token string, limit uint64, filters map[providercall.Field]any) ([]*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"token":   token,
		"limit":   limit,
		"filters": filters,
	})

	res, err := h.db.ProviderCallList(ctx, token, limit, filters)
	if err != nil {
		log.Errorf("Could not get providercalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete soft-deletes the providercall and returns the deleted record.
func (h *providerCallHandler) Delete(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "Delete",
		"providercall_id": id,
	})

	if err := h.db.ProviderCallDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the providercall. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the providercall")
	}

	res, err := h.db.ProviderCallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted providercall. err: %v", err)
		return nil, errors.Wrap(err, "could not get deleted providercall")
	}
	h.notifyHandler.PublishEvent(ctx, providercall.EventTypeProviderCallDeleted, res)

	return res, nil
}
