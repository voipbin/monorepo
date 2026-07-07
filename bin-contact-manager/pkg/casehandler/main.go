package casehandler

//go:generate mockgen -package casehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// CaseHandler interface for Case business logic operations (design §4's
// get-or-create algorithm and §5's lifecycle).
type CaseHandler interface {
	// GetOrCreate implements design doc §4 exactly: reuse an open Case for
	// (customerID, peerType, peerTarget, referenceType) if one exists and
	// has not timed out; otherwise close-and-reopen (timeout) or insert a
	// fresh Case (first contact / no prior case), honoring an explicit
	// case_id hint (§4.3) when present and valid. caseIDHint may be nil.
	//
	// self is the local (business-side) endpoint address for this event,
	// used only for design §4.4's proactive-link write: when a brand-new
	// Case opens for a non-"conversation_message" referenceType (today:
	// "call"), GetOrCreate looks up the sibling message Conversation for
	// (self, peer) and, if found, stamps Metadata.ContactCaseID on it.
	// Pass a zero commonaddress.Address to opt out of this optimization
	// entirely (e.g. reference_type == "conversation_message" itself,
	// which never triggers this write per §4.4's scope).
	GetOrCreate(ctx context.Context, customerID uuid.UUID, self commonaddress.Address, peerType commonaddress.Type, peerTarget, referenceType string, caseIDHint *uuid.UUID) (*kase.Case, error)
}

type caseHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewCaseHandler returns CaseHandler interface
func NewCaseHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) CaseHandler {
	return &caseHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
	}
}
