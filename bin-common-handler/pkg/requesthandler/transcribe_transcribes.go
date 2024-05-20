package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmrequest "monorepo/bin-transcribe-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// TranscribeV1TranscribeGet sends a request to transcribe-manager
// to getting a call.
// it returns given transcribe id's transcribe if it succeed.
func (r *requestHandler) TranscribeV1TranscribeGet(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/transcribes/%s", transcribeID)

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodGet, "transcribe/transcribes/<transcribe-id>", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TranscribeV1TranscribeGets sends a request to transcribe-manager
// to getting a list of transcribe info.
// it returns detail list of transcribe info if it succeed.
func (r *requestHandler) TranscribeV1TranscribeGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]tmtranscribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/transcribes?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodGet, "transcribe/transcribes", 30000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// TranscribeV1TranscribeStart sends a request to transcribe-manager
// to create and starts a transcribe.
// it returns created transcribe info if it succeed.
func (r *requestHandler) TranscribeV1TranscribeStart(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType tmtranscribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction tmtranscribe.Direction,
) (*tmtranscribe.Transcribe, error) {
	uri := "/v1/transcribes"

	data := &tmrequest.V1DataTranscribesPost{
		CustomerID:    customerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Language:      language,
		Direction:     direction,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodPost, "transcribe/transcribes", 30000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TranscribeV1TranscribeStop sends a request to transcribe-manager
// to stops a live transcribe.
// it returns stopped transcribe info if it succeed.
func (r *requestHandler) TranscribeV1TranscribeStop(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/transcribes/%s/stop", transcribeID)

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodPost, "transcribe/transcribes", 30000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TranscribeV1TranscribeDelete sends a request to transcribe-manager
// to deleting the transcribe.
func (r *requestHandler) TranscribeV1TranscribeDelete(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/transcribes/%s", transcribeID)

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodDelete, "transcribe/transcribes/<transcribe-id>", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TranscribeV1TranscribeHealthCheck sends the request to the transcribe-manager for transcribe health-check
//
// delay: milliseconds
func (r *requestHandler) TranscribeV1TranscribeHealthCheck(ctx context.Context, id uuid.UUID, delay int, retryCount int) error {
	uri := fmt.Sprintf("/v1/transcribes/%s/health-check", id)

	type Data struct {
		RetryCount int `json:"retry_count,omitempty"`
	}

	m, err := json.Marshal(Data{
		RetryCount: retryCount,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodPost, "transcribe/transcribes/<transcribe-id>/health-check", requestTimeoutDefault, delay, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
