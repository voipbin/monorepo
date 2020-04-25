package arirequest

//go:generate mockgen -destination ./mock_arirequest_requesthandler.go -package arirequest gitlab.com/voipbin/bin-manager/call-manager/pkg/arirequest RequestHandler

import (
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmqari"
)

// methods
var (
	reqPost   = "POST"
	reqGet    = "GET"
	reqPut    = "PUT"
	reqDelete = "DELETE"
)

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
	requester rabbitmqari.Requester
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmq.Rabbit) RequestHandler {
	requester := rabbitmqari.NewRabbitMQARI(sock)

	reqHandler := &requestHandler{
		requester: requester,
	}

	return reqHandler
}
