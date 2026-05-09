package providerhandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
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
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameRouteManager,
				"PROVIDER_NOT_FOUND",
				"The provider was not found.",
			).Wrap(err)
		}
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
	codecs string,
) (*provider.Provider, error) {
	log := logrus.WithField("func", "Create")

	normalizedCodecs, err := validateCodecs(codecs)
	if err != nil {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameRouteManager,
			"INVALID_CODECS",
			fmt.Sprintf("Invalid codecs value: %v", err),
		)
	}

	id := uuid.Must(uuid.NewV4())
	p := &provider.Provider{
		ID:           id,
		Type:         providerType,
		Hostname:     hostname,
		TechPrefix:   techPrefix,
		TechPostfix:  techPostfix,
		TechHeaders:  techHeaders,
		Name:         name,
		Detail:       detail,
		Codecs:       normalizedCodecs,
		HealthStatus: provider.HealthStatusUnknown,
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

// List returns list of providers
func (h *providerHandler) List(ctx context.Context, token string, limit uint64) ([]*provider.Provider, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "List",
			"token": token,
			"limit": limit,
		})
	log.Debug("Getting providers.")

	filters := map[provider.Field]any{}

	res, err := h.db.ProviderList(ctx, token, limit, filters)
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
	codecs string,
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
			"codecs":      codecs,
		},
	).Debug("Updating the provider.")

	normalizedCodecs, err := validateCodecs(codecs)
	if err != nil {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameRouteManager,
			"INVALID_CODECS",
			fmt.Sprintf("Invalid codecs value: %v", err),
		)
	}

	// Fetch current provider to determine if hostname is changing.
	current, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get current provider. err: %v", err)
		return nil, errors.Wrap(err, "could not get current provider")
	}

	fields := map[provider.Field]any{
		provider.FieldType:        providerType,
		provider.FieldHostname:    hostname,
		provider.FieldTechPrefix:  techPrefix,
		provider.FieldTechPostfix: techPostfix,
		provider.FieldTechHeaders: techHeaders,
		provider.FieldName:        name,
		provider.FieldDetail:      detail,
		provider.FieldCodecs:      normalizedCodecs,
	}

	// Reset health status only when hostname actually changes.
	if current.Hostname != hostname {
		fields[provider.FieldHealthStatus] = provider.HealthStatusUnknown
		fields[provider.FieldHealthCheckedAt] = nil
	}

	if errUpdate := h.db.ProviderUpdate(ctx, id, fields); errUpdate != nil {
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
