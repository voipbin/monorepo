package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	nmnumber "monorepo/bin-number-manager/models/number"
	nmprovider "monorepo/bin-number-manager/models/provider"
	nmrequest "monorepo/bin-number-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// numberProviderURI returns the number-manager uri of the given provider.
func numberProviderURI(name nmnumber.ProviderName) string {
	return fmt.Sprintf("/v1/number_providers/%s", url.PathEscape(string(name)))
}

// NumberV1NumberProviderList sends a request to number-manager
// to get the list of the number providers.
func (r *requestHandler) NumberV1NumberProviderList(ctx context.Context) ([]*nmprovider.Info, error) {
	tmp, err := r.sendRequestNumber(ctx, "/v1/number_providers", sock.RequestMethodGet, "number/number_providers", 15000, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res []*nmprovider.Info
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// NumberV1NumberProviderGet sends a request to number-manager
// to get the given number provider.
func (r *requestHandler) NumberV1NumberProviderGet(ctx context.Context, name nmnumber.ProviderName) (*nmprovider.Info, error) {
	tmp, err := r.sendRequestNumber(ctx, numberProviderURI(name), sock.RequestMethodGet, "number/number_providers", 15000, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res nmprovider.Info
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// NumberV1NumberProviderUpdate sends a request to number-manager
// to update the enabled, priority and config of the given number provider.
// A nil enabled or priority, and an empty config, are left unchanged.
func (r *requestHandler) NumberV1NumberProviderUpdate(ctx context.Context, name nmnumber.ProviderName, actorID uuid.UUID, enabled *bool, priority *int, config map[string]string) (*nmprovider.Info, error) {
	data := &nmrequest.V1DataNumberProvidersIDPut{
		ActorID:  actorID,
		Enabled:  enabled,
		Priority: priority,
		Config:   config,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the request")
	}

	tmp, err := r.sendRequestNumber(ctx, numberProviderURI(name), sock.RequestMethodPut, "number/number_providers", 15000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res nmprovider.Info
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// NumberV1NumberProviderUpdateSecret sends a request to number-manager
// to store the secret of the given number provider.
// The secret is carried only in the request body. It is never part of an uri, a log or an error.
func (r *requestHandler) NumberV1NumberProviderUpdateSecret(ctx context.Context, name nmnumber.ProviderName, actorID uuid.UUID, secret string) (*nmprovider.Info, error) {
	data := &nmrequest.V1DataNumberProvidersIDSecretPut{
		ActorID: actorID,
		Secret:  secret,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the request")
	}

	uri := numberProviderURI(name) + "/secret"
	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPut, "number/number_providers/secret", 15000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res nmprovider.Info
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// NumberV1NumberProviderDeleteSecret sends a request to number-manager
// to delete the stored secret of the given number provider.
// The request body carries the actor, which number-manager requires.
func (r *requestHandler) NumberV1NumberProviderDeleteSecret(ctx context.Context, name nmnumber.ProviderName, actorID uuid.UUID) (*nmprovider.Info, error) {
	data := &nmrequest.V1DataNumberProvidersIDSecretDelete{
		ActorID: actorID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the request")
	}

	uri := numberProviderURI(name) + "/secret"
	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodDelete, "number/number_providers/secret", 15000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res nmprovider.Info
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
