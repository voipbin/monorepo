package requesthandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
)

// ErrConfigServiceUnknown is returned for a service that has no config RPC.
var ErrConfigServiceUnknown = errors.New("the service has no config RPC")

// names of the services in the config API.
const (
	configServiceEmail      = "email"
	configServiceAI         = "ai"
	configServiceMessage    = "message"
	configServiceTTS        = "tts"
	configServiceTranscribe = "transcribe"
	configServiceBilling    = "billing"
	configServicePipecat    = "pipecat"
)

// configQueue returns the request queue of the service of the config API, and the label of the
// metrics. The label is a fixed text per service, never a key name. A service is added here with
// one line, together with the mount of configrpc in its listener.
func configQueue(service string) (commonoutline.QueueName, string, error) {
	switch service {
	case configServiceEmail:
		return commonoutline.QueueNameEmailRequest, "email/configs", nil
	case configServiceAI:
		return commonoutline.QueueNameAIRequest, "ai/configs", nil
	case configServiceMessage:
		return commonoutline.QueueNameMessageRequest, "message/configs", nil
	case configServiceTTS:
		return commonoutline.QueueNameTTSRequest, "tts/configs", nil
	case configServiceTranscribe:
		return commonoutline.QueueNameTranscribeRequest, "transcribe/configs", nil
	case configServiceBilling:
		return commonoutline.QueueNameBillingRequest, "billing/configs", nil
	case configServicePipecat:
		return commonoutline.QueueNamePipecatRequest, "pipecat/configs", nil
	}

	return "", "", ErrConfigServiceUnknown
}

func configKeyURI(key string) string {
	return "/v1/configs/" + url.PathEscape(key)
}

func (r *requestHandler) sendRequestConfig(ctx context.Context, service, uri string, method sock.RequestMethod, labelSuffix string, data json.RawMessage) (*sock.Response, error) {
	queue, label, err := configQueue(service)
	if err != nil {
		return nil, err
	}

	return r.sendRequest(ctx, queue, uri, method, label+labelSuffix, 15000, 0, ContentTypeJSON, data)
}

// ConfigV1ServiceGet sends a request to the manager of the service
// to get its config keys and whether its secrets can be written.
func (r *requestHandler) ConfigV1ServiceGet(ctx context.Context, service string) (*configrpc.Service, error) {
	tmp, err := r.sendRequestConfig(ctx, service, "/v1/configs", sock.RequestMethodGet, "", nil)
	if err != nil {
		return nil, err
	}

	var res configrpc.Service
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}
	res.Service = service

	return &res, nil
}

// ConfigV1KeySecretSet sends a request to the manager of the service
// to store the secret of the given key, if its stored revision is the given one.
// The secret is carried only in the request body. It is never part of an uri, a log or an error.
func (r *requestHandler) ConfigV1KeySecretSet(ctx context.Context, service, key string, actorID uuid.UUID, secret string, revision uint32) (*configrpc.Key, error) {
	data := &configrpc.SecretPutRequest{
		ActorID:  actorID,
		Secret:   secret,
		Revision: &revision,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the request")
	}

	tmp, err := r.sendRequestConfig(ctx, service, configKeyURI(key)+"/secret", sock.RequestMethodPut, "/secret", m)
	if err != nil {
		return nil, err
	}

	var res configrpc.Key
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConfigV1KeyClear sends a request to the manager of the service
// to remove the stored value of the given key, if its stored revision is the given one.
func (r *requestHandler) ConfigV1KeyClear(ctx context.Context, service, key string, actorID uuid.UUID, revision uint32) (*configrpc.Key, error) {
	data := &configrpc.KeyDeleteRequest{
		ActorID:  actorID,
		Revision: &revision,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the request")
	}

	tmp, err := r.sendRequestConfig(ctx, service, configKeyURI(key), sock.RequestMethodDelete, "", m)
	if err != nil {
		return nil, err
	}

	var res configrpc.Key
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConfigV1KeyHistoryList sends a request to the manager of the service
// to get a page of the change history of the given key, the newest first.
func (r *requestHandler) ConfigV1KeyHistoryList(ctx context.Context, service, key, pageToken string, pageSize int) (*configrpc.HistoryList, error) {
	uri := configKeyURI(key) + "/histories"

	query := url.Values{}
	if pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		query.Set("page_token", pageToken)
	}
	if len(query) > 0 {
		uri += "?" + query.Encode()
	}

	tmp, err := r.sendRequestConfig(ctx, service, uri, sock.RequestMethodGet, "/histories", nil)
	if err != nil {
		return nil, err
	}

	var res configrpc.HistoryList
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
