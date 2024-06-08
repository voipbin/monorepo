package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	amresource "monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"net/url"

	"github.com/gofrs/uuid"
)

// AgentV1ResourceGets sends a request to agent-manager to getting a list of agent info.
// It returns detail list of agent info if it succeed.
//
// Parameters:
// ctx (context.Context): The context for the request.
// pageToken (string): The token for the next page.
// pageSize (uint64): The number of items to return per page.
// filters (map[string]string): The filters to apply to the request.
//
// Returns:
// ([]amresource.Resource, error): A list of resources on success, or an error if something went wrong.
//
// The function constructs a GET request to the agent-manager service at the specified endpoint.
// It then sends the request using the sendRequestAgent method, which handles the communication with the service.
// The function checks the response status code and unmarshals the response data into a slice of amresource.Resource objects.
// If an error occurs during the request or response handling, the function returns an error.
// If the resource is not found, the function returns a 404 error.
// If the request is successful, the function returns a slice of resources.
func (r *requestHandler) AgentV1ResourceGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]amresource.Resource, error) {
	uri := fmt.Sprintf("/v1/resources?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestAgent(ctx, uri, rabbitmqhandler.RequestMethodGet, "agent/resources", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amresource.Resource
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// AgentV1ResourceGet sends a request to agent-manager to getting an agent.
// It returns an agent if it succeeds.
//
// Parameters:
// ctx (context.Context): The context for the request.
// resourceID (uuid.UUID): The unique identifier of the resource to get.
//
// Returns:
// (*amresource.Resource, error): A pointer to the requested resource on success, or an error if something went wrong.
//
// The function constructs a GET request to the agent-manager service at the specified resource ID.
// It then sends the request using the sendRequestAgent method, which handles the communication with the service.
// The function checks the response status code and unmarshals the response data into an amresource.Resource object.
// If an error occurs during the request or response handling, the function returns an error.
// If the resource is not found, the function returns a 404 error.
// If the request is successful, the function returns a pointer to the requested resource.
func (r *requestHandler) AgentV1ResourceGet(ctx context.Context, resourceID uuid.UUID) (*amresource.Resource, error) {
	uri := fmt.Sprintf("/v1/resources/%s", resourceID)

	tmp, err := r.sendRequestAgent(ctx, uri, rabbitmqhandler.RequestMethodGet, "agent/resources/<resource-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amresource.Resource
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentV1ResourceDelete sends a request to agent-manager
// to delete a resource.
//
// Parameters:
// ctx (context.Context): The context for the request.
// resourceID (uuid.UUID): The unique identifier of the resource to delete.
//
// Returns:
// (*amresource.Resource, error): A pointer to the deleted resource on success, or an error if something went wrong.
//
// The function constructs a DELETE request to the agent-manager service at the specified resource ID.
// It then sends the request using the sendRequestAgent method, which handles the communication with the service.
// The function checks the response status code and unmarshals the response data into an amresource.Resource object.
// If an error occurs during the request or response handling, the function returns an error.
// If the resource is not found, the function returns a 404 error.
// If the request is successful, the function returns a pointer to the deleted resource.
func (r *requestHandler) AgentV1ResourceDelete(ctx context.Context, resourceID uuid.UUID) (*amresource.Resource, error) {
	uri := fmt.Sprintf("/v1/resources/%s", resourceID)

	tmp, err := r.sendRequestAgent(ctx, uri, rabbitmqhandler.RequestMethodDelete, "agent/resources/<resource-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amresource.Resource
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
