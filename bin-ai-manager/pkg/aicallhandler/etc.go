package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *aicallHandler) Send(ctx context.Context, id uuid.UUID, role message.Role, messageText string, runImmediately bool) (*message.Message, error) {
	c, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	switch c.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.SendReferenceTypeCall(ctx, c, role, messageText, runImmediately)

	default:
		return h.SendReferenceTypeOthers(ctx, c, role, messageText, runImmediately)
	}
}

func (h *aicallHandler) SendReferenceTypeCall(ctx context.Context, c *aicall.AIcall, role message.Role, messageText string, runImmediately bool) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "SendReferenceTypeCall",
		"aicall_id": c.ID,
	})

	pc, err := h.reqHandler.PipecatV1PipecatcallGet(ctx, c.PipecatcallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the pipecatcall correctly")
	}
	log.WithField("pipecatcall", pc).Debugf("Found the pipecatcall.")

	// note: after create a new aicall, we need to create a new message for the conversation message
	res, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", res.ID)
	}
	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", res.ID, res.ID)

	tmp, err := h.reqHandler.PipecatV1MessageSend(ctx, pc.HostID, pc.ID, "", messageText, true, true)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message to the pipecatcall correctly")
	}
	log.WithField("pipecat_message", tmp).Debugf("Sent the message to the pipecatcall.")

	return nil, nil
}

func (h *aicallHandler) SendReferenceTypeOthers(ctx context.Context, c *aicall.AIcall, role message.Role, messageText string, runImmediately bool) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "aicallhandler.Send",
		"aicall_id": c.ID,
	})

	// note: after create a new aicall, we need to create a new message for the conversation message
	res, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", res.ID)
	}
	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", res.ID, res.ID)

	pc, err := h.startPipecatcall(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	tmp, err := h.reqHandler.PipecatV1PipecatcallTerminate(ctx, pc.HostID, pc.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not terminate the pipecatcall correctly")
	}
	log.WithField("pipecatcall_terminate", tmp).Debugf("Terminated the pipecatcall correctly.")

	return nil, nil
}
