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
	GetOrCreate(ctx context.Context, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string, caseIDHint *uuid.UUID) (*kase.Case, error)
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
