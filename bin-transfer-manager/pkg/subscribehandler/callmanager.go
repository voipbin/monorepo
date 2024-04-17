package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"
)

// processEventCMGroupcallProgressing handles the call-manager's groupcall_progressing event
func (h *subscribeHandler) processEventCMGroupcallProgressing(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupcallProgressing",
		"event": m,
	})

	evt := cmgroupcall.Groupcall{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get transfer
	tr, err := h.transferHandler.GetByGroupcallID(ctx, evt.ID)
	if err != nil {
		// not a transfer's groupcall.
		// nothing to do
		return nil
	}

	if errAnswer := h.transferHandler.TransfereeAnswer(ctx, tr, &evt); errAnswer != nil {
		log.Errorf("Could not handle the answer transferee. err: %v", errAnswer)
	}

	return nil
}

// processEventCMGroupcallHangup handles call-manager's groupcall_hangup event
func (h *subscribeHandler) processEventCMGroupcallHangup(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupcallHangup",
		"event": m,
	})

	evt := cmgroupcall.Groupcall{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get transfer
	tr, err := h.transferHandler.GetByGroupcallID(ctx, evt.ID)
	if err != nil {
		// not a transfer's groupcall.
		// nothing to do
		return nil
	}

	if errAnswer := h.transferHandler.TransfereeHangup(ctx, tr, &evt); errAnswer != nil {
		log.Errorf("Could not handle the answer transferee. err: %v", errAnswer)
	}

	return nil
}

// processEventCMCallHangup handles call-manager's call_hangup event
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallHangup",
		"event": m,
	})

	evt := cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get transferer call
	c, err := h.transferHandler.GetByTransfererCallID(ctx, evt.ID)
	if err != nil {
		// not a transferer call.
		// nothing to do
		return nil
	}

	if errAnswer := h.transferHandler.TransfererHangup(ctx, c, &evt); errAnswer != nil {
		log.Errorf("Could not handle the answer transferee. err: %v", errAnswer)
	}

	return nil
}
