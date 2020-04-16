package arihandler

//go:generate mockgen -destination ./mock_arihandler_requester.go -package arihandler gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler Requester

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"

	log "github.com/sirupsen/logrus"
)

var requestTargetPrefix = "asterisk_ari_request"

// Requester is interface for send ARI request
type Requester interface {
	sendARIRequest(sock rabbitmq.Rabbit, asteriskID, url, method string, timeout int64, dataType, data string) (Response, error)
}

type requester struct{}

// Request struct for ARI request
type Request struct {
	URL      string `json:"url"`
	Method   string `json:"method"`
	DataType string `json:"data_type"`
	Data     string `json:"data"`
}

// Response defines RPC message response
type Response struct {
	StatusCode int    `json:"status_code"`
	Data       string `json:"data"`
}

// sendARIRequest send a request and return the response
func (r *requester) sendARIRequest(sock rabbitmq.Rabbit, asteriskID, url, method string, timeout int64, dataType, data string) (Response, error) {
	log.WithFields(log.Fields{
		"asterisk_id": asteriskID,
		"method":      method,
		"url":         url,
		"data_type":   dataType,
	}).Debugf("Sending ARI request. data: %s", data)

	// create a request message
	m, err := json.Marshal(Request{
		url,
		method,
		dataType,
		data,
	})
	if err != nil {
		return Response{}, fmt.Errorf("could not create a request message. err: %v", err)
	}

	// create target
	target := fmt.Sprintf("%s-%s", requestTargetPrefix, asteriskID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	res, err := sock.PublishRPC(ctx, target, string(m))
	if err != nil {
		return Response{}, fmt.Errorf("could not publish the RPC. err: %v", err)
	}

	// Response
	resp := Response{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return Response{}, err
	}

	log.WithFields(log.Fields{
		"status_code": resp.StatusCode,
	}).Debugf("Received result. data: %s", resp.Data)

	return resp, nil
}
