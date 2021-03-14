package notifyhandler

//go:generate mockgen -destination ./mock_notifyhandler_notifyhandler.go -package notifyhandler -source ./main.go NotifyHandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

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

// default stasis application name.
// normally, we don't need to use this, because proxy will set this automatically.
// but, some of Asterisk ARI required application name. this is for that.
const defaultAstStasisApp = "voipbin"

var (
	metricsNamespace = "call_manager"

	promRequestProcessTime = prometheus.NewHistogramVec(
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
)

// eventType
type eventType string

// list of event types
const (
	eventTypeCallCreated eventType = "call_created"
)

// const event publisher
const eventPublisher = "call-manager"

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// NotifyHandler intreface
type NotifyHandler interface {
	// call
	CallCreate(c *call.Call)
}

type notifyHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay  string
	exchangeNotify string
}

// NewNotifyHandler create NotifyHandler
func NewNotifyHandler(sock rabbitmqhandler.Rabbit, exchangeDelay, exchangeEvent string) NotifyHandler {
	h := &notifyHandler{
		sock: sock,

		exchangeDelay:  exchangeDelay,
		exchangeNotify: exchangeEvent,
	}

	if err := sock.ExchangeDeclare(exchangeEvent, "fanout", true, false, false, false, nil); err != nil {
		logrus.Errorf("Could not declare the event exchange. err: %v", err)
		return nil
	}

	return h
}

func uriUnescape(u string) string {
	res, err := url.QueryUnescape(u)
	if err != nil {
		return "could not unescape the url"
	}

	return res
}

// publishNotify publishes a notify message.
func (r *notifyHandler) publishNotify(eventType eventType, dataType string, data json.RawMessage, timeout int) error {

	log.WithFields(log.Fields{
		"type":      eventType,
		"data_type": dataType,
		"data":      data,
	}).Debugf("Publishing the notification. type: %s", eventType)

	// creat a request message
	evt := &rabbitmqhandler.Event{
		Type:      string(eventType),
		Publisher: eventPublisher,
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

		log.WithFields(log.Fields{
			"event": evt,
		}).Debugf("Published event. type: %s", evt.Type)

		return nil
	}
}

// publishDirectEvnt publish the event to the target without delay
func (r *notifyHandler) publishDirectEvnt(ctx context.Context, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := r.sock.PublishExchangeEvent(r.exchangeNotify, "", evt)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// sendDelayedEvent sends the delayed event
// delay unit is millisecond.
func (r *notifyHandler) sendDelayedEvent(ctx context.Context, delay int, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := r.sock.PublishExchangeDelayedEvent(r.exchangeDelay, r.exchangeNotify, evt, delay)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}
