package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

// PeerEventList resolves the contact_id OR peer_type+peer_target filter into
// a timeline-manager peer_pairs query and returns the raw (unfiltered)
// peer_events rows. Unlike InteractionList, this NEVER applies CRM
// eligibility filtering — the caller (square-admin/square-talk) is expected
// to do any presentation-layer grouping/filtering of noise itself.
func (h *serviceHandler) PeerEventList(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	peerType, peerTarget string,
	pageToken string,
	pageSize uint64,
) ([]*tmpeerevent.PeerEvent, string, error) {
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	pairs, err := h.resolvePeerPairs(ctx, a.CustomerID, contactID, peerType, peerTarget)
	if err != nil {
		return nil, "", err
	}
	if len(pairs) == 0 {
		return nil, "", nil // no addresses on this contact -> empty result, no RPC call
	}

	req := &tmpeerevent.PeerEventListRequest{
		CustomerID: a.CustomerID,
		PeerPairs:  pairs,
		PageToken:  pageToken,
		PageSize:   int(pageSize),
	}
	res, err := h.reqHandler.TimelineV1PeerEventList(ctx, req)
	if err != nil {
		return nil, "", err
	}
	return res.Result, res.NextPageToken, nil
}

// ServiceAgentPeerEventList is the service-agent-facing equivalent of
// PeerEventList. Mirrors ServiceAgentInteractionList's real relationship to
// InteractionList: uses a.CustomerID directly (no agentGet call), gated by
// PermissionAll instead of PermissionCustomerAdmin|PermissionCustomerManager.
func (h *serviceHandler) ServiceAgentPeerEventList(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	peerType, peerTarget string,
	pageToken string,
	pageSize uint64,
) ([]*tmpeerevent.PeerEvent, string, error) {
	if !a.IsAgent() {
		return nil, "", serviceerrors.ErrAuthenticationRequired
	}
	if a.IsDirect() {
		return nil, "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	pairs, err := h.resolvePeerPairs(ctx, a.CustomerID, contactID, peerType, peerTarget)
	if err != nil {
		return nil, "", err
	}
	if len(pairs) == 0 {
		return nil, "", nil
	}

	req := &tmpeerevent.PeerEventListRequest{
		CustomerID: a.CustomerID,
		PeerPairs:  pairs,
		PageToken:  pageToken,
		PageSize:   int(pageSize),
	}
	res, err := h.reqHandler.TimelineV1PeerEventList(ctx, req)
	if err != nil {
		return nil, "", err
	}
	return res.Result, res.NextPageToken, nil
}

// resolvePeerPairs implements the "exactly one filter" contract: contact_id
// resolves via contactGet + tenant check, deduping Contact.Addresses into
// peer_pairs; OR peer_type+peer_target is a single-pair passthrough.
// The HTTP layer (server/contact_peer_events.go) already enforces
// exactly-one-filter via filterCount, so in practice this is never reached
// with both filters set; the switch's contactID-first ordering is purely
// an implementation detail of that unreachable-in-practice case.
func (h *serviceHandler) resolvePeerPairs(
	ctx context.Context,
	customerID uuid.UUID,
	contactID uuid.UUID,
	peerType, peerTarget string,
) ([]tmpeerevent.PeerPair, error) {
	switch {
	case contactID != uuid.Nil:
		ct, err := h.contactGet(ctx, contactID)
		if err != nil {
			return nil, err
		}
		// Tenant guard: never resolve another customer's contact. Returns
		// ErrNotFound (not ErrPermissionDenied) to avoid leaking the
		// existence of another tenant's contact — same anti-enumeration
		// precedent as ContactAddressClaim/ServiceAgentContactAddressClaim
		// in contact_address.go.
		if ct.CustomerID != customerID {
			return nil, serviceerrors.ErrNotFound
		}

		seen := make(map[tmpeerevent.PeerPair]struct{})
		var pairs []tmpeerevent.PeerPair
		for _, addr := range ct.Addresses {
			p := tmpeerevent.PeerPair{PeerType: string(addr.Type), PeerTarget: addr.Target}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			pairs = append(pairs, p)
		}
		return pairs, nil

	case peerType != "" && peerTarget != "":
		return []tmpeerevent.PeerPair{{PeerType: peerType, PeerTarget: peerTarget}}, nil

	default:
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager, "INVALID_FILTER",
			"Exactly one filter is required: contact_id, or peer_type+peer_target.",
		)
	}
}
