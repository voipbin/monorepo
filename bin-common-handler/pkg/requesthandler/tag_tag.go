package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	tmtag "gitlab.com/voipbin/bin-manager/tag-manager.git/models/tag"
	tmrequest "gitlab.com/voipbin/bin-manager/tag-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// TagV1TagCreate sends a request to tag-manager
// to creating a tag.
// it returns created tag if it succeed.
func (r *requestHandler) TagV1TagCreate(ctx context.Context, customerID uuid.UUID, name string, detail string) (*tmtag.Tag, error) {
	uri := "/v1/tags"

	data := &tmrequest.V1DataTagsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTag(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceTagTags, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TagV1TagUpdate sends a request to tag-manager
// to update the tag info.
// it returns updated tag if it succeed.
func (r *requestHandler) TagV1TagUpdate(ctx context.Context, tagID uuid.UUID, name string, detail string) (*tmtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags/%s", tagID)

	data := &tmrequest.V1DataTagsIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTag(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceTagTags, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TagV1TagDelete sends a request to tag-manager
// to deleting the tag.
func (r *requestHandler) TagV1TagDelete(ctx context.Context, tagID uuid.UUID) (*tmtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags/%s", tagID)

	tmp, err := r.sendRequestTag(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceTagTags, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TagV1TagGet sends a request to tag-manager
// to getting the tag.
func (r *requestHandler) TagV1TagGet(ctx context.Context, tagID uuid.UUID) (*tmtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags/%s", tagID)

	tmp, err := r.sendRequestTag(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceTagTags, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TagV1TagGets sends a request to tag-manager
// to getting the list of tags.
func (r *requestHandler) TagV1TagGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]tmtag.Tag, error) {
	uri := fmt.Sprintf("/v1/tags?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestTag(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceTagTags, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []tmtag.Tag
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}
