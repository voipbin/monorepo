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
// All fields are pointers: nil means "leave unchanged", a non-nil
// pointer means "set to this value" (including a pointer to an empty
// codecs string, which explicitly resets codecs to server-default
// negotiation).
func (h *providerHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	providerType *provider.Type,
	hostname *string,
	techPrefix *string,
	techPostfix *string,
	techHeaders *map[string]string,
	name *string,
	detail *string,
	codecs *string,
) (*provider.Provider, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Update",
			"id":   id,
		})
	log.Debug("Updating the provider.")

	// Fetch current provider first: needed both for the hostname-change
	// comparison below and as the response for an all-omitted no-op PUT.
	current, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get current provider. err: %v", err)
		return nil, errors.Wrap(err, "could not get current provider")
	}

	fields := map[provider.Field]any{}
	if providerType != nil {
		fields[provider.FieldType] = *providerType
	}
	if hostname != nil {
		fields[provider.FieldHostname] = *hostname
	}
	if techPrefix != nil {
		fields[provider.FieldTechPrefix] = *techPrefix
	}
	if techPostfix != nil {
		fields[provider.FieldTechPostfix] = *techPostfix
	}
	if techHeaders != nil {
		fields[provider.FieldTechHeaders] = *techHeaders
	}
	if name != nil {
		fields[provider.FieldName] = *name
	}
	if detail != nil {
		fields[provider.FieldDetail] = *detail
	}
	if codecs != nil {
		normalizedCodecs, errValidate := validateCodecs(*codecs)
		if errValidate != nil {
			return nil, cerrors.InvalidArgument(
				commonoutline.ServiceNameRouteManager,
				"INVALID_CODECS",
				fmt.Sprintf("Invalid codecs value: %v", errValidate),
			)
		}
		fields[provider.FieldCodecs] = normalizedCodecs
	}

	// Reset health status only when hostname actually changes.
	if hostname != nil && current.Hostname != *hostname {
		fields[provider.FieldHealthStatus] = provider.HealthStatusUnknown
		fields[provider.FieldHealthCheckedAt] = nil
	}

	if len(fields) == 0 {
		// nothing to update; no PublishEvent for a true no-op PUT.
		return current, nil
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
