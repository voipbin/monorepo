package arirequest

//go:generate mockgen -destination ./mock_arirequest_requesthandler.go -package arirequest gitlab.com/voipbin/bin-manager/call-manager/pkg/arirequest RequestHandler

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"

	log "github.com/sirupsen/logrus"
)

// reqMethod type
type reqMethod string

// List of reqMethod
const (
	reqMethodPost   reqMethod = "POST"
	reqMethodGet    reqMethod = "GET"
	reqMethodPut    reqMethod = "PUT"
	reqMethodDelete reqMethod = "DELETE"
)

var requestTargetPrefix = "asterisk_ari_request"

// contents type
var (
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

const requestTimeout int64 = 3 // default request timeout

// RequestHandler intreface for ARI request handler
type RequestHandler interface {
	ChannelAnswer(asteriskID, channelID string) error
	ChannelContinue(asteriskID, channelID, context, ext string, pri int, label string) error
	ChannelHangup(asteriskID, channelID string, code ari.ChannelCause) error
	ChannelVariableSet(asteriskID, channelID, variable, value string) error
}

type requestHandler struct {
	sock rabbitmq.Rabbit
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmq.Rabbit) RequestHandler {
	h := &requestHandler{
		sock: sock,
	}

	return h
}

// SendARIRequest send a request and return the response
func (r *requestHandler) sendRequest(asteriskID, uri string, method reqMethod, timeout int64, dataType, data string) (*rabbitmq.Response, error) {
	log.WithFields(log.Fields{
		"asterisk_id": asteriskID,
		"method":      method,
		"url":         uri,
		"data_type":   dataType,
	}).Debugf("Sending ARI request. data: %s", data)

	// get method
	mapMethod := map[reqMethod]rabbitmq.RequestMethod{
		reqMethodGet:    rabbitmq.RequestMethodGet,
		reqMethodPost:   rabbitmq.RequestMethodPost,
		reqMethodPut:    rabbitmq.RequestMethodPut,
		reqMethodDelete: rabbitmq.RequestMethodDelete,
	}
	rabbitMethod := mapMethod[method]
	if rabbitMethod == "" {
		return nil, fmt.Errorf("Unsupported method type. method: %s", method)
	}

	// creat a request message
	m := &rabbitmq.Request{
		URI:      uri,
		Method:   rabbitMethod,
		DataType: dataType,
		Data:     data,
	}

	// create target
	target := fmt.Sprintf("%s-%s", requestTargetPrefix, asteriskID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	res, err := r.sock.PublishRPC(ctx, target, m)
	if err != nil {
		return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
	}

	log.WithFields(log.Fields{
		"asterisk_id": asteriskID,
		"method":      method,
		"url":         uri,
		"status_code": res.StatusCode,
	}).Debugf("Received result. data: %s", res.Data)

	return res, nil
}
