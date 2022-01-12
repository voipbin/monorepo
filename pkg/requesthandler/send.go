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
// delayed: delay request(ms)
func (r *requestHandler) sendRequest(queue string, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	switch {
	case delayed > 0:
		// send scheduled message.
		// we don't expect the response message here.
		if err := r.sendDelayedRequest(ctx, r.exchangeDelay, queue, resource, delayed, req); err != nil {
			return nil, fmt.Errorf("could not publish the delayed request. err: %v", err)
		}
		return nil, nil

	default:
		res, err := r.sendDirectRequest(ctx, queue, resource, req)
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
func (r *requestHandler) sendDelayedRequest(ctx context.Context, target string, queue string, resource resource, delay int, req *rabbitmqhandler.Request) error {

	start := time.Now()
	err := r.sock.PublishExchangeDelayedRequest(r.exchangeDelay, queue, req, delay)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// SendARIRequest send a request to the Asterisk-proxy and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestAst(asteriskID, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	// create target
	target := fmt.Sprintf("asterisk.%s.request", asteriskID)

	return r.sendRequest(target, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestFM send a request to the flow-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestFM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueFlow, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTTS send a request to the tts-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestTTS(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueTTS, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestWM send a request to the webhook-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestWM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueWebhook, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCM send a request to the call-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestCM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueCall, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestRM send a request to the registrar-manager and return the response
func (r *requestHandler) sendRequestRM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRegistrar, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestNM send a request to the number-manager and return the response
func (r *requestHandler) sendRequestNM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueNumber, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCF send a request to the conference-manager and return the response
func (r *requestHandler) sendRequestCF(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueConference, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTS send a request to the transcribe-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestTS(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueTranscribe, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestSM send a request to the storage-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestSM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueStorage, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestUM send a request to the user-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestUM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueUser, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestAM send a request to the agent-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestAM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueAgent, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestQM send a request to the queue-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestQM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueQueue, uri, method, resource, timeout, delayed, dataType, data)
}
