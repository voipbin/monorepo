// Package webhook defines the shared shape of a webhook event payload as
// consumed inside bin-api-manager's subscribehandler package. This lives
// under models/, not inline in pkg/subscribehandler, following this
// service's existing convention (see models/auth, models/hook,
// models/common, models/stream) of keeping small model types out of
// pkg/*handler files.
package webhook

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// CommonData is the resource-agnostic shape every webhook event payload
// shares: an Identity (id, customer_id) and an optional Owner
// (owner_id/owner_type). It intentionally does NOT carry any
// resource-specific fields (e.g. an aicall_id, chat_id, or session_id) --
// those vary per event type and belong to that event type's own
// unmarshal target, not to a struct every publisher's payload is forced
// through. See pkg/subscribehandler/webhookmanager.go's createTopics for
// how each resource's own extra fields are decoded locally, scoped to
// just the switch case that needs them.
type CommonData struct {
	commonidentity.Identity
	commonidentity.Owner
}
