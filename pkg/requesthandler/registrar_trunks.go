package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	rmsipauth "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
	rmtrunk "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
	rmrequest "gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// RegistrarV1TrunkCreate sends a request to registrar-manager
// to creating a trunk.
// it returns created trunk if it succeed.
func (r *requestHandler) RegistrarV1TrunkCreate(ctx context.Context, customerID uuid.UUID, name string, detail string, domainName string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.Trunk, error) {
	uri := "/v1/trunks"

	data := &rmrequest.V1DataTrunksPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
		DomainName: domainName,
		Authtypes:  authTypes,
		Username:   username,
		Password:   password,
		AllowedIPs: allowedIPs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodPost, "registrar/trunks", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmtrunk.Trunk
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RegistrarV1TrunkGets sends a request to registrar-manager
// to getting a list of trunk info.
// it returns detail list of trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	for k, v := range filters {
		uri = fmt.Sprintf("%s&filter_%s=%s", uri, k, v)
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodGet, "registrar/trunks", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []rmtrunk.Trunk
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// RegistrarV1TrunkGet sends a request to registrar-manager
// to getting a detail trunk info.
// it returns detail trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkGet(ctx context.Context, trunkID uuid.UUID) (*rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks/%s", trunkID)

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodGet, "registrar/trunks/<trunk-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmtrunk.Trunk
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RegistrarV1TrunkGetByDomainName sends a request to registrar-manager
// to getting a detail trunk info of the given domain name.
// it returns detail trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkGetByDomainName(ctx context.Context, domainName string) (*rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks/domain_name/%s", domainName)

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodGet, "registrar/trunks/domain_name", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmtrunk.Trunk
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RegistrarV1TrunkDelete sends a request to registrar-manager
// to deleting the trunk.
// it returns deleted trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkDelete(ctx context.Context, trunkID uuid.UUID) (*rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks/%s", trunkID)

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodDelete, "registrar/trunks/<trunk-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmtrunk.Trunk
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RegistrarV1TrunkUpdateBasicInfo sends a request to registrar-manager
// to update the basic trunk info.
// it returns updated trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkUpdateBasicInfo(ctx context.Context, trunkID uuid.UUID, name string, detail string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks/%s", trunkID)

	data := &rmrequest.V1DataTrunksIDPut{
		Name:       name,
		Detail:     detail,
		Authtypes:  authTypes,
		Username:   username,
		Password:   password,
		AllowedIPs: allowedIPs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodPut, "registrar/trunks/<trunk-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmtrunk.Trunk
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
