package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"
	rmrequest "monorepo/bin-registrar-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
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

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodPost, "registrar/trunks", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmtrunk.Trunk
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1TrunkGets sends a request to registrar-manager
// to getting a list of trunk info.
// it returns detail list of trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/trunks", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []rmtrunk.Trunk
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// RegistrarV1TrunkGet sends a request to registrar-manager
// to getting a detail trunk info.
// it returns detail trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkGet(ctx context.Context, trunkID uuid.UUID) (*rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks/%s", trunkID)

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/trunks/<trunk-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmtrunk.Trunk
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1TrunkGetByDomainName sends a request to registrar-manager
// to getting a detail trunk info of the given domain name.
// it returns detail trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkGetByDomainName(ctx context.Context, domainName string) (*rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks/domain_name/%s", domainName)

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/trunks/domain_name", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmtrunk.Trunk
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1TrunkDelete sends a request to registrar-manager
// to deleting the trunk.
// it returns deleted trunk info if it succeed.
func (r *requestHandler) RegistrarV1TrunkDelete(ctx context.Context, trunkID uuid.UUID) (*rmtrunk.Trunk, error) {
	uri := fmt.Sprintf("/v1/trunks/%s", trunkID)

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodDelete, "registrar/trunks/<trunk-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmtrunk.Trunk
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodPut, "registrar/trunks/<trunk-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmtrunk.Trunk
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
