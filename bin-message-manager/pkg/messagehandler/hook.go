package messagehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	nmnumber "monorepo/bin-number-manager/models/number"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/telnyx"
)

// Hook handles hook message
func (h *messageHandler) Hook(ctx context.Context, uri string, data []byte) error {
	var m *message.Message
	var num *nmnumber.Number
	var err error
	switch {
	case strings.HasSuffix(uri, hookTelnyx):
		m, num, err = h.hookTelnyx(ctx, data)
		if err != nil {
			return errors.Wrapf(err, "Could not handle the hook message. uri: %s", uri)
		}

	default:
		return fmt.Errorf("unknown hook uri. uri: %s", uri)
	}

	if m == nil || num == nil {
		// nothing to do.
		return nil
	}

	return nil
}

// hookTelnyx telnyx type hook message.
func (h *messageHandler) hookTelnyx(ctx context.Context, data []byte) (*message.Message, *nmnumber.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "hookTelnyx",
	})

	hm := telnyx.MessageEvent{}
	if errUnmarshal := json.Unmarshal(data, &hm); errUnmarshal != nil {
		// the body is not part of the error: it carries the numbers, the text and the media urls.
		return nil, nil, errors.Wrapf(errUnmarshal, "Could not unmarshal the data. data_size: %d", len(data))
	}
	log = log.WithFields(logrus.Fields{
		"event_type": hm.Data.EventType,
		"event_id":   hm.Data.ID,
	})
	log.Debugf("Unmarshalled hook message. event_type: %s", hm.Data.EventType)

	if len(hm.Data.Payload.To) == 0 {
		return nil, nil, fmt.Errorf("destination address is empty")
	}

	if hm.Data.EventType == "message.sent" || hm.Data.EventType == "message.finalized" {
		log.Debugf("Received message sent event. event_type: %s", hm.Data.EventType)
		return nil, nil, nil
	}

	destinationNumber := hm.Data.Payload.To[0].PhoneNumber
	log.Debugf("Parsed destination number. destination_number: %s", destinationNumber)

	// get number info
	filters := map[nmnumber.Field]any{
		nmnumber.FieldNumber:  destinationNumber,
		nmnumber.FieldDeleted: false,
	}
	numbs, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Could not get number info. number: %s", destinationNumber)
	}

	if len(numbs) == 0 {
		return nil, nil, fmt.Errorf("no number info found. len: %d", len(numbs))
	}

	num := numbs[0]
	log.WithField("number", num).Infof("Found number info. number_id: %s", num.ID)

	// get information
	source := hm.GetSource()
	targets := hm.GetTargets()
	text := hm.GetText()

	// Canonicalize inbound source/targets through the shared address
	// normalization authority. NormalizeTarget is loss-proof, so the error is
	// discarded. Nil-guard the source pointer; normalize targets by index.
	if source != nil {
		source.Target, _ = commonaddress.NormalizeTarget(source.Type, source.Target)
	}
	for i := range targets {
		targets[i].Destination.Target, _ = commonaddress.NormalizeTarget(targets[i].Destination.Type, targets[i].Destination.Target)
	}

	// the media is not downloaded here: the hook has to answer within the rpc timeout and the provider delivers the
	// event again when it does not. The message is created at once and the media is stored in the background.
	medias, skipped := hm.GetMedias()
	if skipped > 0 {
		promInboundMediaTotal.WithLabelValues(string(FetchResultParse)).Add(float64(skipped))
	}

	msgType := message.TypeSMS
	if hm.IsMMS() {
		msgType = message.TypeMMS
	}

	res, err := h.create(ctx, uuid.Nil, num.CustomerID, source, targets, message.ProviderNameTelnyx, text, message.DirectionInbound, msgType)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Could not create a message. number_id: %s, number: %s", num.ID, num.Number)
	}
	log.Debugf("Created message. message_id: %s, type: %s", res.ID, res.Type)

	if len(medias) > 0 {
		h.startInboundMediaIngest(ctx, res, medias)
	}

	return res, &num, nil
}
