package transcribehandler

import (
	"github.com/gofrs/uuid"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/streaming"
)

func newTestCustomer(id uuid.UUID) *cucustomer.Customer {
	return &cucustomer.Customer{ID: id}
}

func newTestStreaming(id uuid.UUID) *streaming.Streaming {
	return &streaming.Streaming{Identity: commonidentity.Identity{ID: id}}
}
