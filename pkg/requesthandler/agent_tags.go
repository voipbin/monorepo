package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	amrequest "gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// AgentV1TagsCreate sends a request to agent-manager
// to creating a tag.
// it returns created call if it succeed.
func (r *requestHandler) AgentV1TagCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
) (*amtag.Tag, error) {
	uri := "/v1/tags"

	data := &amrequest.V1DataTagsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(uri, rabbitmqhandler.RequestMethodPost, resourceAgentTags, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentV1TagGet sends a request to agent-manager
// to getting an tag.
// it returns an tag if it succeed.
func (r *requestHandler) AgentV1TagGet(ctx context.Context, id uuid.UUID) (*amtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags/%s", id)

	tmp, err := r.sendRequestAgent(uri, rabbitmqhandler.RequestMethodGet, resourceAgentTags, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentV1TagGets sends a request to agent-manager
// to getting a list of tag info.
// it returns detail list of tag info if it succeed.
func (r *requestHandler) AgentV1TagGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]amtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestAgent(uri, rabbitmqhandler.RequestMethodGet, resourceAgentTags, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// AgentV1TagUpdate sends a request to agent-manager
// to update teh tag basic info
// it returns error if something went wrong.
func (r *requestHandler) AgentV1TagUpdate(ctx context.Context, id uuid.UUID, name, detail string) (*amtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags/%s", id)

	data := &amrequest.V1DataTagsIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(uri, rabbitmqhandler.RequestMethodPut, resourceAgentTags, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentV1TagDelete sends a request to agent-manager
// to delete the tag.
// it returns error if something went wrong.
func (r *requestHandler) AgentV1TagDelete(ctx context.Context, id uuid.UUID) (*amtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags/%s", id)

	tmp, err := r.sendRequestAgent(uri, rabbitmqhandler.RequestMethodDelete, resourceAgentTags, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
