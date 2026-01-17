package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

// SendRequest sends a request to the given destination.
//
// timeout: timeout(ms)
// delayed: delay request(ms)
func (r *requestHandler) SendRequest(ctx context.Context, queue commonoutline.QueueName, uri string, method sock.RequestMethod, timeout int, delay int, dataType string, data json.RawMessage) (*sock.Response, error) {
	return r.sendRequest(ctx, queue, uri, method, "", timeout, delay, dataType, data)
}

// sendRequest sends a request to the given destination.
//
// timeout: timeout(ms)
// delayed: delay request(ms)
func (r *requestHandler) sendRequest(ctx context.Context, queue commonoutline.QueueName, uri string, method sock.RequestMethod, resource string, timeout int, delay int, dataType string, data json.RawMessage) (*sock.Response, error) {
	// creat a request message
	req := &sock.Request{
		URI:       uri,
		Method:    method,
		Publisher: string(r.publisher),
		DataType:  dataType,
		Data:      data,
	}

	cctx, cancel := context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
	defer cancel()

	switch {
	case delay > 0:
		// send scheduled message.
		// we don't expect the response message here.
		if err := r.sendDelayedRequest(string(queue), resource, delay, req); err != nil {
			return nil, errors.Wrapf(err, "could not send the delayed request. queue: %s, method: %s, uri: %s", queue, method, uri)
		}
		return nil, nil

	default:
		res, err := r.sendDirectRequest(cctx, string(queue), resource, req)
		if err != nil {
			return nil, errors.Wrapf(err, "could not send the request. queue: %s, method: %s, uri: %s", queue, method, uri)
		}

		return res, nil
	}
}

// sendDirectRequest sends the request to the target without delay
func (r *requestHandler) sendDirectRequest(ctx context.Context, target string, resource string, req *sock.Request) (*sock.Response, error) {

	start := time.Now()
	res, err := r.sock.RequestPublish(ctx, target, req)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return res, err
}

// sendDelayedRequest sends the delayed request to the target
// delay: delay time(millisecond).
func (r *requestHandler) sendDelayedRequest(queue string, resource string, delay int, req *sock.Request) error {

	start := time.Now()
	err := r.sock.RequestPublishWithDelay(queue, req, delay)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(string(commonoutline.QueueNameDelay), string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// SendARIRequest send a request to the Asterisk-proxy and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestAst(ctx context.Context, asteriskID, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	// create target
	target := fmt.Sprintf("asterisk.%s.request", asteriskID)

	return r.sendRequest(ctx, commonoutline.QueueName(target), uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestFlow send a request to the flow-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestFlow(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameFlowRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTTS send a request to the tts-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTTS(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameTTSRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestWebhook send a request to the webhook-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestWebhook(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameWebhookRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCall send a request to the call-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestCall(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameCallRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestRegistrar send a request to the registrar-manager and return the response
func (r *requestHandler) sendRequestRegistrar(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameRegistrarRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestNumber send a request to the number-manager and return the response
func (r *requestHandler) sendRequestNumber(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameNumberRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestConference send a request to the conference-manager and return the response
func (r *requestHandler) sendRequestConference(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameConferenceRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTranscribe send a request to the transcribe-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTranscribe(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameTranscribeRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestStorage send a request to the storage-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestStorage(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameStorageRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestAgent send a request to the agent-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestAgent(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameAgentRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestQueue send a request to the queue-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestQueue(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameQueueRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCustomer send a request to the customer-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestCustomer(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameCustomerRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestEmail send a request to the email-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestEmail(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameEmailRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestMessage send a request to the message-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestMessage(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameMessageRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestOutdial send a request to the outdial-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestOutdial(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameOutdialRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCampaign send a request to the campaign-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestCampaign(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameCampaignRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestConversation send a request to the conversation-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestConversation(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameConversationRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestChat send a request to the chat-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestChat(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameChatRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestRoute send a request to the route-manager and return the response
func (r *requestHandler) sendRequestRoute(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameRouteRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestAI send a request to the ai-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestAI(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameAIRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTransfer send a request to the transfer-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTransfer(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameTransferRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestBilling send a request to the billing-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestBilling(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameBillingRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTag send a request to the tag-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTag(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameTagRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestPipecat send a request to the pipecat-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestPipecat(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data json.RawMessage) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNamePipecatRequest, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTalk send a request to the talk-manager and return the response
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestTalk(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout, delayed int, dataType string, data []byte) (*sock.Response, error) {

	return r.sendRequest(ctx, commonoutline.QueueNameTalkRequest, uri, method, resource, timeout, delayed, dataType, data)
}
