package arihandler

//go:generate mockgen -destination ./mock_arihandler_requesthandler.go -package arihandler gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler RequestHandler

import (
	"encoding/json"
	"fmt"

	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// methods
var (
	reqPost = "POST"
	reqGet  = "GET"
	reqPut  = "PUT"
)

// contents type
var (
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

const requestTimeout int64 = 3 // default request timeout

// RequestHandler intreface for ARI request handler
type RequestHandler interface {
	SetSock(rabbitmq.Rabbit)

	ChannelAnswer(asteriskID, channelID string) error
	ChannelContinue(asteriskID, channelID, context, ext string, pri int, label string) error
	ChannelVariableSet(asteriskID, channelID, variable, value string) error
}

type requestHandler struct {
	rabbitSock rabbitmq.Rabbit
	requester  Requester
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler() RequestHandler {
	requestHandler := &requestHandler{}
	requestHandler.requester = &requester{}

	return requestHandler
}

// setSock sets amqp sock
func (r *requestHandler) SetSock(sock rabbitmq.Rabbit) {
	r.rabbitSock = sock
}

// ChannelAnswer sends the channel answer request
func (r *requestHandler) ChannelAnswer(asteriskID, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/answer", channelID)

	res, err := r.requester.sendARIRequest(r.rabbitSock, asteriskID, url, reqPost, requestTimeout, "", "")
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// ChannelContinue sends the continue request
func (r *requestHandler) ChannelContinue(asteriskID, channelID, context, ext string, pri int, label string) error {
	url := fmt.Sprintf("/ari/channels/%s/continue", channelID)

	type Data struct {
		Context   string `json:"context"`
		Extension string `json:"extension"`
		Priority  int    `json:"priority"`
		Label     string `json:"label"`
	}

	m, err := json.Marshal(Data{
		context,
		ext,
		pri,
		label,
	})
	if err != nil {
		return err
	}

	res, err := r.requester.sendARIRequest(r.rabbitSock, asteriskID, url, reqPost, requestTimeout, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// ChannelVariableSet sends the variable set request
func (r *requestHandler) ChannelVariableSet(asteriskID, channelID, variable, value string) error {
	url := fmt.Sprintf("/ari/channels/%s/variable", channelID)

	type Data struct {
		Variable string `json:"variable"`
		Value    string `json:"value"`
	}

	m, err := json.Marshal(Data{
		variable,
		value,
	})
	if err != nil {
		return err
	}

	res, err := r.requester.sendARIRequest(r.rabbitSock, asteriskID, url, reqPost, requestTimeout, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
