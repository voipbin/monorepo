package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *aicallHandler) Send(ctx context.Context, id uuid.UUID, role message.Role, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	c, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	switch c.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.SendReferenceTypeCall(ctx, c, role, messageText, runImmediately, audioResponse)

	default:
		return h.SendReferenceTypeOthers(ctx, c, role, messageText)
	}
}

func (h *aicallHandler) SendReferenceTypeCall(ctx context.Context, c *aicall.AIcall, role message.Role, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "SendReferenceTypeCall",
		"aicall_id": c.ID,
	})

	pc, err := h.reqHandler.PipecatV1PipecatcallGet(ctx, c.PipecatcallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the pipecatcall correctly")
	}
	log.WithField("pipecatcall", pc).Debugf("Found the pipecatcall.")

	res, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", res.ID)
	}
	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", res.ID, res.ID)

	tmp, err := h.reqHandler.PipecatV1MessageSend(ctx, pc.HostID, pc.ID, res.ID.String(), messageText, runImmediately, audioResponse)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message to the pipecatcall correctly")
	}
	log.WithField("pipecat_message", tmp).Debugf("Sent the message to the pipecatcall.")

	return res, nil
}

func (h *aicallHandler) SendReferenceTypeOthers(ctx context.Context, c *aicall.AIcall, role message.Role, messageText string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "aicallhandler.Send",
		"aicall_id": c.ID,
	})

	// note: after create a new aicall, we need to create a new message for the conversation message
	res, errTerminate := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not create the message. aicall_id: %s", res.ID)
	}

	newPipecatcallID := h.utilHandler.UUIDCreate()
	c, errTerminate = h.UpdatePipecatcallID(ctx, c.ID, newPipecatcallID)
	if errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not update the pipecatcall id for existing aicall. aicall_id: %s", c.ID)
	}

	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", res.ID, res.ID)
	pc, errTerminate := h.startPipecatcall(ctx, c)
	if errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	if errTerminate = h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultPipecatcallTerminateDelay); errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not send the pipecatcall terminate request correctly")
	}

	return res, nil
}
