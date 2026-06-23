package analysishandler

import (
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// commonIdentity builds an Identity from id + customerID.
func commonIdentity(id, customerID uuid.UUID) commonidentity.Identity {
	return commonidentity.Identity{
		ID:         id,
		CustomerID: customerID,
	}
}
