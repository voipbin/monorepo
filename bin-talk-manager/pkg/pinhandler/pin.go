package pinhandler

import (
	"context"
	stderrors "errors"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/models/pin"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

// ownerTypeAgent is the only owner type that can pin: pins are the personal state of an agent.
const ownerTypeAgent = string(commonidentity.OwnerTypeAgent)

func invalidArgument(reason, msg string) error {
	return cerrors.InvalidArgument(commonoutline.ServiceNameTalkManager, reason, msg)
}

func notFound(reason, msg string, cause error) error {
	e := cerrors.NotFound(commonoutline.ServiceNameTalkManager, reason, msg)
	if cause != nil {
		return e.Wrap(cause)
	}
	return e
}

func validateOwner(customerID uuid.UUID, ownerType string, ownerID uuid.UUID) error {
	if customerID == uuid.Nil {
		return invalidArgument("INVALID_CUSTOMER_ID", "The customer id is required.")
	}
	if ownerType != ownerTypeAgent {
		return invalidArgument("INVALID_OWNER_TYPE", "Only an agent can pin.")
	}
	if ownerID == uuid.Nil {
		return invalidArgument("INVALID_OWNER_ID", "The owner id is required.")
	}
	return nil
}

// PinCreate pins a chat or a message for the owner.
func (h *pinHandler) PinCreate(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID, targetType pin.TargetType, targetID uuid.UUID) (*pin.Pin, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PinCreate",
		"customer_id": customerID,
		"owner_id":    ownerID,
		"target_type": targetType,
		"target_id":   targetID,
	})

	if err := validateOwner(customerID, ownerType, ownerID); err != nil {
		return nil, err
	}
	if !targetType.IsValid() {
		return nil, invalidArgument("INVALID_TARGET_TYPE", "The target type must be chat or message.")
	}
	if targetID == uuid.Nil {
		return nil, invalidArgument("INVALID_TARGET_ID", "The target id is required.")
	}

	// the chat of the target. For a message pin the message is read first, then its chat.
	chatID := targetID
	if targetType == pin.TargetTypeMessage {
		m, err := h.dbHandler.MessageGet(ctx, targetID)
		if err != nil {
			if stderrors.Is(err, dbhandler.ErrNotFound) {
				return nil, notFound("MESSAGE_NOT_FOUND", "The message was not found.", err)
			}
			return nil, errors.Wrap(err, "could not get the message")
		}
		if m.TMDelete != nil || m.CustomerID != customerID {
			return nil, notFound("MESSAGE_NOT_FOUND", "The message was not found.", nil)
		}
		chatID = m.ChatID
	}

	c, err := h.dbHandler.ChatGet(ctx, chatID)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, notFound("CHAT_NOT_FOUND", "The chat was not found.", err)
		}
		return nil, errors.Wrap(err, "could not get the chat")
	}
	if c.TMDelete != nil || c.CustomerID != customerID {
		return nil, notFound("CHAT_NOT_FOUND", "The chat was not found.", nil)
	}

	// a talk chat is open to the agents of the customer; the others need a seat in the chat. This is
	// the same condition the list applies, so that a pin is never created that the list would hide.
	if c.Type != chat.TypeTalk {
		participants, err := h.dbHandler.ParticipantList(ctx, map[participant.Field]any{
			participant.FieldChatID:    chatID,
			participant.FieldOwnerType: ownerType,
			participant.FieldOwnerID:   ownerID,
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not get the participants")
		}
		if len(participants) == 0 {
			return nil, cerrors.PermissionDenied(commonoutline.ServiceNameTalkManager, "NOT_A_PARTICIPANT", "The agent is not a participant of the chat.")
		}
	}

	p := &pin.Pin{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerType(ownerType),
			OwnerID:   ownerID,
		},
		ChatID:     chatID,
		TargetType: targetType,
		TargetID:   targetID,
	}

	res, err := h.dbHandler.PinCreate(ctx, p)
	if err != nil {
		log.Errorf("Could not create the pin. err: %v", err)
		return nil, errors.Wrap(err, "could not create the pin")
	}

	return res, nil
}

// PinList returns the pins of the owner that the owner can see.
func (h *pinHandler) PinList(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID, targetType pin.TargetType, chatID uuid.UUID, token string, size uint64) ([]*pin.Pin, error) {
	if err := validateOwner(customerID, ownerType, ownerID); err != nil {
		return nil, err
	}
	if targetType != "" && !targetType.IsValid() {
		return nil, invalidArgument("INVALID_TARGET_TYPE", "The target type must be chat or message.")
	}
	if size == 0 {
		size = defaultPageSize
	}

	res, err := h.dbHandler.PinList(ctx, customerID, ownerType, ownerID, targetType, chatID, token, size)
	if err != nil {
		return nil, errors.Wrap(err, "could not list the pins")
	}

	return res, nil
}

// PinDelete deletes a pin of the owner. The pin is read first to return it; a pin of another owner
// or customer is reported as not found, the same as an unknown id. A pin that is no longer
// visible (for example the agent left the chat) can still be deleted by its owner.
func (h *pinHandler) PinDelete(ctx context.Context, id, customerID uuid.UUID, ownerType string, ownerID uuid.UUID) (*pin.Pin, error) {
	if id == uuid.Nil {
		return nil, invalidArgument("INVALID_ID", "The pin id is required.")
	}
	if err := validateOwner(customerID, ownerType, ownerID); err != nil {
		return nil, err
	}

	p, err := h.dbHandler.PinGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, notFound("PIN_NOT_FOUND", "The pin was not found.", err)
		}
		return nil, errors.Wrap(err, "could not get the pin")
	}
	if p.CustomerID != customerID || string(p.OwnerType) != ownerType || p.OwnerID != ownerID {
		return nil, notFound("PIN_NOT_FOUND", "The pin was not found.", nil)
	}

	if err := h.dbHandler.PinDelete(ctx, id, customerID, ownerType, ownerID); err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, notFound("PIN_NOT_FOUND", "The pin was not found.", err)
		}
		return nil, errors.Wrap(err, "could not delete the pin")
	}

	return p, nil
}
