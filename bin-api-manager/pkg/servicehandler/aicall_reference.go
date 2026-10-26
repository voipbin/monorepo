package servicehandler

import (
	"context"
	stderrors "errors"
	"fmt"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// aicallReferenceValidate validates the reference of an aicall to be created
// and returns the reference id to use.
//
// reference_type must be one of the explicit allowlist below (the generated
// enum validator is not used: it also accepts the internal-only task type).
// For the none type the reference id is forced to uuid.Nil, because the
// generated active_reference_key column keys off the id, not the type.
// For every other type the referenced resource must exist and belong to
// customerID. Missing, deleted and foreign resources all return the same bare
// serviceerrors.ErrNotFound so a caller cannot tell them apart.
func (h *serviceHandler) aicallReferenceValidate(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType amaicall.ReferenceType,
	referenceID uuid.UUID,
) (uuid.UUID, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "aicallReferenceValidate",
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	switch referenceType {
	case amaicall.ReferenceTypeNone:
		return uuid.Nil, nil

	case amaicall.ReferenceTypeCall, amaicall.ReferenceTypeConversation, amaicall.ReferenceTypeContactCase:
		// handled below

	default:
		return uuid.Nil, fmt.Errorf("%w: unsupported reference type: %s", serviceerrors.ErrInvalidArgument, referenceType)
	}

	if referenceID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%w: reference_id is required for reference_type: %s", serviceerrors.ErrInvalidArgument, referenceType)
	}

	var ownerID uuid.UUID
	var deleted bool
	switch referenceType {
	case amaicall.ReferenceTypeCall:
		tmp, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
		if err != nil {
			return uuid.Nil, aicallReferenceGetError(err)
		}
		ownerID = tmp.CustomerID
		deleted = tmp.TMDelete != nil

	case amaicall.ReferenceTypeConversation:
		tmp, err := h.reqHandler.ConversationV1ConversationGet(ctx, referenceID)
		if err != nil {
			return uuid.Nil, aicallReferenceGetError(err)
		}
		ownerID = tmp.CustomerID
		deleted = tmp.TMDelete != nil

	case amaicall.ReferenceTypeContactCase:
		tmp, err := h.reqHandler.ContactV1CaseGet(ctx, customerID, referenceID)
		if err != nil {
			return uuid.Nil, aicallReferenceGetError(err)
		}
		ownerID = tmp.CustomerID
	}

	if deleted || ownerID != customerID {
		log.Info("The referenced resource is not available to the customer.")
		return uuid.Nil, serviceerrors.ErrNotFound
	}

	return referenceID, nil
}

// aicallReferenceGetError classifies a failed reference lookup. Every
// not-found form is reduced to the bare sentinel, which drops the backend
// error from the chain: the error translator passes typed errors through
// unchanged, so keeping it would let the backend's reason code differ from the
// ownership-mismatch response. Any other error keeps its classification.
func aicallReferenceGetError(err error) error {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound {
		return serviceerrors.ErrNotFound
	}
	if stderrors.Is(err, requesthandler.ErrNotFound) || stderrors.Is(err, serviceerrors.ErrNotFound) {
		return serviceerrors.ErrNotFound
	}

	return errors.Wrapf(err, "could not get the reference info")
}
