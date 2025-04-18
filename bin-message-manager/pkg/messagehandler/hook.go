package messagehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/telnyx"
)

// Hook handles hook message
func (h *messageHandler) Hook(ctx context.Context, uri string, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Hook",
		"uri":  uri,
		"data": data,
	})

	var m *message.Message
	var num *nmnumber.Number
	var err error

	if strings.HasSuffix(uri, hookTelnyx) {
		m, num, err = h.hookTelnyx(ctx, data)
	}
	if err != nil {
		log.Errorf("Could not handle the hook message correctly. err: %v", err)
		return err
	}

	// execute messageflow
	af, errExecute := h.executeMessageFlow(ctx, m, num)
	if errExecute != nil {
		log.Errorf("Could not execute the messageflow correctly. err: %v", errExecute)
		return errExecute
	} else if af == nil {
		// no activeflow created
		return nil
	}
	log.WithField("activeflow", af).Debugf("Executed messageflow. activeflow_id: %s", af.ID)

	return nil
}

// hookTelnyx telnyx type hook message.
func (h *messageHandler) hookTelnyx(ctx context.Context, data []byte) (*message.Message, *nmnumber.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "hookTelnyx",
		"data": data,
	})

	hm := telnyx.MessageEvent{}
	if errUnmarshal := json.Unmarshal(data, &hm); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the data. err: %v", errUnmarshal)
		return nil, nil, errUnmarshal
	}

	if len(hm.Data.Payload.To) == 0 {
		log.Errorf("Destination address is empty.")
		return nil, nil, fmt.Errorf("destination address is empty")
	}

	destinationNumber := hm.Data.Payload.To[0].PhoneNumber
	log.Debugf("Parsed destination number. destination_number: %s", destinationNumber)

	// get number info
	filters := map[string]string{
		"number":  destinationNumber,
		"deleted": "false",
	}
	numbs, err := h.reqHandler.NumberV1NumberGets(ctx, "", 1, filters)
	if err != nil {
		log.Errorf("Could not get numbers info. err: %v", err)
		return nil, nil, err
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
		log.Errorf("Could not create a message record. err: %v", err)
		return nil, nil, err
	}

	log.WithField("message", res).Debugf("Created message. message_id: %s", res.ID)

	return res, &num, nil
}

// executeMessageFlow executes the given number's messageflow with message.
func (h *messageHandler) executeMessageFlow(ctx context.Context, m *message.Message, num *nmnumber.Number) (*fmactiveflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "executeMessageFlow",
		"message": m,
		"number":  num,
	})

	if num.MessageFlowID == uuid.Nil {
		// nothing to do. has no message flow id
		return nil, nil
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		m.CustomerID,
		num.MessageFlowID,
		fmactiveflow.ReferenceTypeMessage,
		m.ID,
		uuid.Nil,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create an activeflow. message_id: %s, number_id: %s", m.ID, num.ID)
	}
	log.WithField("activeflow", af).Debugf("Created activeflow. activeflow_id: %s", af.ID)

	// set variables
	if errVariable := h.setVariables(ctx, af.ID, m); errVariable != nil {
		return nil, errors.Wrapf(errVariable, "Could not set the variables. activeflow_id: %s", af.ID)
	}

	// execute the activeflow
	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
		return nil, errors.Wrapf(errExecute, "Could not execute the activeflow. activeflow_id: %s", af.ID)
	}
	log.Debugf("Executed activeflow. activeflow_id: %s", af.ID)

	return af, nil
}
