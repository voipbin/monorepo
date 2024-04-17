package requesthandler

import (
	"context"
	"encoding/json"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"
)

// publishEvent sends a event to the given destination.
func (r *requestHandler) publishEvent(ctx context.Context, queue commonoutline.QueueName, eventType string, publisher string, dataType string, data json.RawMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "sendEvent",
		"queue":      queue,
		"event_type": eventType,
		"publisher":  publisher,
		"data_type":  dataType,
		"data":       data,
	})

	evt := &rabbitmqhandler.Event{
		Type:      eventType,
		Publisher: publisher,
		DataType:  dataType,
		Data:      data,
	}

	if errPublish := r.sock.PublishEvent(string(queue), evt); errPublish != nil {
		log.Errorf("Could not publish event: %v", errPublish)
		return errPublish
	}
	promEventCount.WithLabelValues(eventType).Inc()

	return nil
}

// CallPublishEvent publish the event to the call-manager.
func (r *requestHandler) CallPublishEvent(ctx context.Context, eventType string, publisher string, dataType string, data []byte) error {

	return r.publishEvent(ctx, commonoutline.QueueNameCallSubscribe, eventType, publisher, dataType, data)
}
