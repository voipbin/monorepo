package servicehandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	pmdevice "monorepo/bin-push-manager/models/device"
)

// newPushDeviceNotFound returns the canonical "push device not found" error.
//
// A device that does not exist, a device of another customer and a device of another agent all yield this same
// error, so that the caller cannot tell them apart.
func newPushDeviceNotFound() error {
	return cerrors.NotFound(commonoutline.ServiceNamePushManager, "DEVICE_NOT_FOUND", "The device was not found.")
}

// ServiceAgentPushDeviceCreate registers or refreshes a push device of the calling agent.
//
// The owner always comes from the authenticated identity. The token is a credential: it is sent to the push-manager
// only and it is neither logged nor returned.
func (h *serviceHandler) ServiceAgentPushDeviceCreate(ctx context.Context, a *auth.AuthIdentity, platform pmdevice.Platform, token string) (*pmdevice.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentPushDeviceCreate",
		"customer_id": a.CustomerID,
		"platform":    platform,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	if !pmdevice.IsValidPlatform(platform) {
		return nil, fmt.Errorf("%w: unsupported platform", serviceerrors.ErrInvalidArgument)
	}
	if token == "" || len(token) > pmdevice.MaxTokenLength {
		return nil, fmt.Errorf("%w: invalid token length", serviceerrors.ErrInvalidArgument)
	}

	tmp, err := h.reqHandler.PushV1DeviceCreate(ctx, a.CustomerID, agentID, platform, token)
	if err != nil {
		log.Errorf("Could not register the push device. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ServiceAgentPushDeviceDelete deletes a push device of the calling agent.
//
// The device must belong to the caller's customer and to the caller. Anything else, including an unknown id, yields
// the same not-found error.
func (h *serviceHandler) ServiceAgentPushDeviceDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*pmdevice.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentPushDeviceDelete",
		"customer_id": a.CustomerID,
		"device_id":   id,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.PushV1DeviceGet(ctx, id)
	if err != nil {
		log.Infof("Could not get the push device. err: %v", err)
		if pushDeviceIsNotFound(err) {
			return nil, newPushDeviceNotFound()
		}
		return nil, err
	}

	if tmp == nil || tmp.CustomerID != a.CustomerID || tmp.AgentID != agentID {
		return nil, newPushDeviceNotFound()
	}

	deleted, err := h.reqHandler.PushV1DeviceDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the push device. err: %v", err)
		if pushDeviceIsNotFound(err) {
			return nil, newPushDeviceNotFound()
		}
		return nil, err
	}

	return deleted.ConvertWebhookMessage(), nil
}

// pushDeviceIsNotFound reports whether an RPC error means the device does not exist.
func pushDeviceIsNotFound(err error) bool {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound {
		return true
	}

	return stderrors.Is(err, requesthandler.ErrNotFound)
}
