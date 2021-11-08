package notifyhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package notifyhandler -destination ./mock_notifyhandler_notifyhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

// WebhookMessage defines
type WebhookMessage interface {
	CreateWebhookEvent(t string) ([]byte, error)
}

// EventType string
type EventType string

// list of event types
const (
	// call
	EventTypeCallCreated EventType = "call_created"
	EventTypeCallUpdated EventType = "call_updated"
	EventTypeCallHungup  EventType = "call_hungup"

	// confbridge
	EventTypeConfbridgeCreated EventType = "confbridge_created"
	EventTypeConfbridgeDeleted EventType = "confbridge_deleted"
	EventTypeConfbridgeJoined  EventType = "confbridge_joined"
	EventTypeConfbridgeLeaved  EventType = "confbridge_leaved"

	// conference
	EventTypeConferenceCreated EventType = "conference_created"
	EventTypeConferenceDeleted EventType = "conference_deleted"
	EventTypeConferenceJoined  EventType = "conference_joined"
	EventTypeConferenceLeaved  EventType = "conference_leaved"

	// recording
	EventTypeRecordingStarted  EventType = "recording_started"
	EventTypeRecordingFinished EventType = "recording_finished"
)

// EventPublisher type
const EventPublisher = "call-manager"

// Data types
var (
	dataTypeText = "text/plain"
	dataTypeJSON = "application/json"
)

const requestTimeoutDefault int = 3 // default request timeout

// delay units
const (
	DelayNow    int = 0
	DelaySecond int = 1000
	DelayMinute int = DelaySecond * 60
	DelayHour   int = DelayMinute * 60
)

var (
	metricsNamespace = "call_manager"

	promNotifyProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "notify_process_time",
			Help:      "Process time of send notification",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"type"},
	)

	promNotifyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "notify_total",
			Help:      "Total number of sent notification.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promNotifyProcessTime,
		promNotifyTotal,
	)
}

// NotifyHandler intreface
type NotifyHandler interface {
	NotifyEvent(eventType EventType, webhookURI string, message WebhookMessage)
	PublishEvent(t EventType, c interface{})
}

type notifyHandler struct {
	sock       rabbitmqhandler.Rabbit
	reqHandler requesthandler.RequestHandler

	exchangeDelay  string
	exchangeNotify string
}

// NewNotifyHandler create NotifyHandler
func NewNotifyHandler(sock rabbitmqhandler.Rabbit, reqHandler requesthandler.RequestHandler, exchangeDelay, exchangeEvent string) NotifyHandler {
	h := &notifyHandler{
		sock:       sock,
		reqHandler: reqHandler,

		exchangeDelay:  exchangeDelay,
		exchangeNotify: exchangeEvent,
	}

	if err := sock.ExchangeDeclare(exchangeEvent, "fanout", true, false, false, false, nil); err != nil {
		logrus.Errorf("Could not declare the event exchange. err: %v", err)
		return nil
	}

	return h
}

//nolint:deadcode,unused // this is ok.
func uriUnescape(u string) string {
	res, err := url.QueryUnescape(u)
	if err != nil {
		return "could not unescape the url"
	}

	return res
}

// PublishNotify publishes a notify message.
func (r *notifyHandler) PublishNotify(eventType string, data json.RawMessage) error {

	log.WithFields(log.Fields{
		"type": eventType,
		"data": data,
	}).Debugf("Publishing the notification. type: %s", eventType)

	return r.publishNotify(eventType, dataTypeText, data, requestTimeoutDefault)
}

// publishNotify publishes a notify message.
func (r *notifyHandler) publishNotify(eventType string, dataType string, data json.RawMessage, timeout int) error {

	log.WithFields(log.Fields{
		"type":      eventType,
		"data_type": dataType,
		"data":      data,
	}).Debugf("Publishing the notification. type: %s", eventType)

	// creat a request message
	evt := &rabbitmqhandler.Event{
		Type:      eventType,
		Publisher: EventPublisher,
		DataType:  dataType,
		Data:      data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	switch {
	// case delayed > 0:
	// 	// send scheduled message.
	// 	// we don't expect the response message here.
	// 	if err := r.sendDelayedRequest(ctx, r.exchangeDelay, queue, resource, delayed, req); err != nil {
	// 		return nil, fmt.Errorf("could not publish the delayed request. err: %v", err)
	// 	}
	// 	return nil, nil

	default:
		err := r.publishDirectEvnt(ctx, evt)
		if err != nil {
			return fmt.Errorf("could not publish the event. err: %v", err)
		}
	}

	promNotifyTotal.WithLabelValues(evt.Type).Inc()
	log.WithFields(log.Fields{
		"event": evt,
	}).Debugf("Published event. type: %s", evt.Type)

	return nil

}

// publishDirectEvnt publish the event to the target without delay
func (r *notifyHandler) publishDirectEvnt(ctx context.Context, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := r.sock.PublishExchangeEvent(r.exchangeNotify, "", evt)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// sendDelayedEvent sends the delayed event
// delay unit is millisecond.
//nolint:unused // this is ok
func (r *notifyHandler) sendDelayedEvent(ctx context.Context, delay int, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := r.sock.PublishExchangeDelayedEvent(r.exchangeDelay, r.exchangeNotify, evt, delay)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}
