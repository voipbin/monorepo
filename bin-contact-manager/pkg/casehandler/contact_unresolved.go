package casehandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
)

// CaseListUnresolved implements design §6's agent-facing unresolved
// queue: WHERE customer_id=? AND status='open' AND contact_id IS NULL,
// backed by idx_case_unresolved. Thin delegation to the dbhandler
// primitive already implemented in Task 3.2.
func (h *caseHandler) CaseListUnresolved(ctx context.Context, customerID uuid.UUID) ([]*kase.Case, error) {
	return h.db.CaseListUnresolved(ctx, customerID)
}

// CaseListAll returns every Case (all tenants), for case-control's
// `--all` reconcile-contact sweep. CLI-only usage -- never exposed via
// a customer-facing RPC/route.
func (h *caseHandler) CaseListAll(ctx context.Context) ([]*kase.Case, error) {
	return h.db.CaseListAll(ctx)
}
