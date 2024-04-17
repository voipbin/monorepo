package providerhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/provider"
)

// Get returns provider
func (h *providerHandler) Get(ctx context.Context, id uuid.UUID) (*provider.Provider, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
		"id":   id,
	})

	res, err := h.db.ProviderGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get provider. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new provider
func (h *providerHandler) Create(
	ctx context.Context,
	providerType provider.Type,
	hostname string,
	techPrefix string,
	techPostfix string,
	techHeaders map[string]string,
	name string,
	detail string,
) (*provider.Provider, error) {
	log := logrus.WithField("func", "Create")

	id := uuid.Must(uuid.NewV4())
	p := &provider.Provider{
		ID:          id,
		Type:        providerType,
		Hostname:    hostname,
		TechPrefix:  techPrefix,
		TechPostfix: techPostfix,
		TechHeaders: techHeaders,
		Name:        name,
		Detail:      detail,
	}
	log.WithField("provider", p).Debugf("Creating a new provider. id: %s", id)

	if errCreate := h.db.ProviderCreate(ctx, p); errCreate != nil {
		log.Errorf("Could not create the provider. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.ProviderGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created provider info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, provider.EventTypeProviderCreated, res)

	return res, nil
}

// Gets returns list of providers
func (h *providerHandler) Gets(ctx context.Context, token string, limit uint64) ([]*provider.Provider, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "Gets",
			"token": token,
			"limit": limit,
		})
	log.Debug("Getting providers.")

	res, err := h.db.ProviderGets(ctx, token, limit)
	if err != nil {
		log.Errorf("Could not get providers. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the provider
func (h *providerHandler) Delete(ctx context.Context, id uuid.UUID) (*provider.Provider, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Delete",
			"provider_id": id,
		},
	)
	log.Debug("Deleting the provider.")

	err := h.db.ProviderDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the provider. err: %v", err)
		return nil, err
	}

	res, err := h.db.ProviderGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted provider. err: %v", err)
		return nil, errors.Wrap(err, "could not get deleted provider")
	}
	h.notifyHandler.PublishEvent(ctx, provider.EventTypeProviderDeleted, res)

	return res, nil
}

// Update updates the provider and return the updated provider
func (h *providerHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	providerType provider.Type,
	hostname string,
	techPrefix string,
	techPostfix string,
	techHeaders map[string]string,
	name string,
	detail string,
) (*provider.Provider, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Update",
			"id":   id,
		})
	log.WithFields(
		logrus.Fields{
			"id":          id,
			"type":        providerType,
			"hostname":    hostname,
			"techPrefix":  techPrefix,
			"techPostfix": techPostfix,
			"techHeaders": techHeaders,
			"name":        name,
			"detail":      detail,
		},
	).Debug("Updating the provider.")

	tmp := &provider.Provider{
		ID:          id,
		Type:        providerType,
		Hostname:    hostname,
		TechPrefix:  techPrefix,
		TechPostfix: techPostfix,
		TechHeaders: techHeaders,
		Name:        name,
		Detail:      detail,
	}
	log.WithField("update_provider", tmp).Debugf("Created update provider info.")

	if errUpdate := h.db.ProviderUpdate(ctx, tmp); errUpdate != nil {
		log.Errorf("Could not update the provider info. err: %v", errUpdate)
		return nil, errors.Wrap(errUpdate, "could not update the provider info")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated provider info. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated provider info")
	}
	h.notifyHandler.PublishEvent(ctx, provider.EventTypeProviderUpdated, res)

	return res, nil
}
