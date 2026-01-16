package messagehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	nmnumber "monorepo/bin-number-manager/models/number"

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
		return nil, nil, errors.Wrapf(errUnmarshal, "Could not unmarshal the data. data: %s", string(data))
	}
	log = log.WithField("hook_message", hm)
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

	// get informations
	source := hm.GetSource()
	targets := hm.GetTargets()
	text := hm.GetText()
	res, err := h.Create(ctx, uuid.Nil, num.CustomerID, source, targets, message.ProviderNameTelnyx, text, message.DirectionInbound)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Could not create a message. number_id: %s, number: %s", num.ID, num.Number)
	}

	log.WithField("message", res).Debugf("Created message. message_id: %s", res.ID)
	return res, &num, nil
}
