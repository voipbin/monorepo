package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// sendRequest sends a request to the given destination.
//
// timeout: timeout(ms)
// delayed: delay request(ms)
func (r *requestHandler) sendRequest(ctx context.Context, queue string, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	// creat a request message
	req := &rabbitmqhandler.Request{
		URI:       uri,
		Method:    method,
		Publisher: r.publisher,
		DataType:  dataType,
		Data:      data,
	}

	log := logrus.WithFields(logrus.Fields{
		"queue":   queue,
		"delayed": delayed,
		"request": req,
	})
	log.Debugf("Sending a request. queue: %s, method: %s, uri: %s", queue, method, uri)

	cctx, cancel := context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
	defer cancel()

	switch {
	case delayed > 0:
		// send scheduled message.
		// we don't expect the response message here.
		if err := r.sendDelayedRequest(cctx, queue, resource, delayed, req); err != nil {
			return nil, fmt.Errorf("could not publish the delayed request. err: %v", err)
		}
		return nil, nil

	default:
		res, err := r.sendDirectRequest(cctx, queue, resource, req)
		if err != nil {
			return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
		}

		log.WithFields(logrus.Fields{
			"response": res,
		}).Debugf("Received result. queue: %s, method: %s, uri: %s, status_code: %d", queue, method, uri, res.StatusCode)
		return res, nil
	}
}

// sendDirectRequest sends the request to the target without delay
func (r *requestHandler) sendDirectRequest(ctx context.Context, target string, resource resource, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	start := time.Now()
	res, err := r.sock.PublishRPC(ctx, target, req)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return res, err
}

// sendDelayedRequest sends the delayed request to the target
// delay: delay time(millisecond).
func (r *requestHandler) sendDelayedRequest(ctx context.Context, queue string, resource resource, delay int, req *rabbitmqhandler.Request) error {

	start := time.Now()
	err := r.sock.PublishExchangeDelayedRequest(exchangeDelay, queue, req, delay)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(exchangeDelay, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// SendARIRequest send a request to the Asterisk-proxy and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestAst(ctx context.Context, asteriskID, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	// create target
	target := fmt.Sprintf("asterisk.%s.request", asteriskID)

	return r.sendRequest(ctx, target, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestFlow send a request to the flow-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestFlow(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueFlow, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTTS send a request to the tts-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTTS(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueTTS, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestWebhook send a request to the webhook-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestWebhook(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueWebhook, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCall send a request to the call-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestCall(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueCall, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestRegistrar send a request to the registrar-manager and return the response
func (r *requestHandler) sendRequestRegistrar(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueRegistrar, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestNumber send a request to the number-manager and return the response
func (r *requestHandler) sendRequestNumber(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueNumber, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestConference send a request to the conference-manager and return the response
func (r *requestHandler) sendRequestConference(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueConference, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTranscribe send a request to the transcribe-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTranscribe(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueTranscribe, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestStorage send a request to the storage-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestStorage(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueStorage, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestUser send a request to the user-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestUser(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueUser, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestAgent send a request to the agent-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestAgent(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueAgent, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestQueue send a request to the queue-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestQueue(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueQueue, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCustomer send a request to the customer-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestCustomer(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueCustomer, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestMessage send a request to the message-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestMessage(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueMessage, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestOutdial send a request to the outdial-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestOutdial(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueOutdial, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCampaign send a request to the campaign-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestCampaign(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueCampaign, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestConversation send a request to the conversation-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestConversation(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueConversation, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestChat send a request to the chat-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestChat(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueChat, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestRoute send a request to the route-manager and return the response
func (r *requestHandler) sendRequestRoute(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueRoute, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestChatbot send a request to the chatbot-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestChatbot(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueChatbot, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTransfer send a request to the transfer-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTransfer(ctx context.Context, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(ctx, queueTransfer, uri, method, resource, timeout, delayed, dataType, data)
}
