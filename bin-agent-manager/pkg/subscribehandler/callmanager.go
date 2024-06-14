package subscribehandler

import (
	"context"
	"encoding/json"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCMGroupcallCreated handles the call-manager's groupcall_created event.
func (h *subscribeHandler) processEventCMGroupcallCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupcallCreated",
		"event": m,
	})

	groupcall := &cmgroupcall.Groupcall{}
	if err := json.Unmarshal([]byte(m.Data), &groupcall); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.agentHandler.EventGroupcallCreated(ctx, groupcall); errEvent != nil {
		log.Errorf("Could not handle the groupcall created event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the groupcall created event.")
	}

	return nil
}

// processEventCMGroupcallProgressing handles the call-manager's groupcall_answered event.
func (h *subscribeHandler) processEventCMGroupcallProgressing(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupcallAnswered",
		"event": m,
	})

	groupcall := &cmgroupcall.Groupcall{}
	if err := json.Unmarshal([]byte(m.Data), &groupcall); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.agentHandler.EventGroupcallProgressing(ctx, groupcall); errEvent != nil {
		log.Errorf("Could not handle the groupcall answered event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the groupcall answered event.")
	}

	return nil
}

// // processEventCMCallDeleted handles the call-manager's call_deleted event.
// func (h *subscribeHandler) processEventCMCallDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":  "processEventCMCallDeleted",
// 		"event": m,
// 	})

// 	c := &cmcall.Call{}
// 	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
// 		log.Errorf("Could not unmarshal the data. err: %v", err)
// 		return err
// 	}

// 	if errEvent := h.resourceHandler.EventCallDeleted(ctx, c); errEvent != nil {
// 		log.Errorf("Could not handle the call_deleted event. err: %v", errEvent)
// 		return errors.Wrap(errEvent, "Could not handle the call_deleted event.")
// 	}

// 	return nil
// }
