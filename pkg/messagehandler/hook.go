package messagehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/hooktelnyx"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
)

// Hook handles hook message
func (h *messageHandler) Hook(ctx context.Context, uri string, data []byte) error {
	if strings.HasSuffix(uri, hookTelnyx) {
		return h.hookTelnyx(ctx, data)
	}

	return nil
}

// hookTelnyx telnyx type hook message.
func (h *messageHandler) hookTelnyx(ctx context.Context, data []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "hookTelnyx",
		},
	)

	hm := hooktelnyx.Message{}
	if errUnmarshal := json.Unmarshal(data, &hm); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the data. err: %v", errUnmarshal)
		return errUnmarshal
	}

	if len(hm.Data.Payload.To) == 0 {
		log.Errorf("Destination address is empty.")
		return fmt.Errorf("destination address is empty")
	}

	toNum := hm.Data.Payload.To[0].PhoneNumber
	log = log.WithField("number", toNum)

	// get number info
	num, err := h.reqHandler.NMV1NumberGetByNumber(ctx, toNum)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return err
	}

	// convert to Message.
	id := uuid.Must(uuid.NewV4())
	msg := hm.ConvertMessage(id, num.CustomerID)
	msg.TMCreate = dbhandler.GetCurTime()
	msg.TMUpdate = dbhandler.DefaultTimeStamp
	msg.TMDelete = dbhandler.DefaultTimeStamp
	log = log.WithFields(logrus.Fields{
		"message_id":  id,
		"customer_id": num.CustomerID,
	})

	res, err := h.Create(ctx, msg)
	if err != nil {
		log.Errorf("Could not create a message record. err: %v", err)
		return err
	}
	log.WithField("message", res).Debugf("Created message. message_id: %s", res.ID)

	return nil
}
