package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

// PeerEventList resolves the contact_id OR peer address filter into a
// timeline-manager peer_addresses query and returns the raw (unfiltered)
// peer_events rows. Unlike InteractionList, this NEVER applies CRM
// eligibility filtering — the caller (square-admin/square-talk) is expected
// to do any presentation-layer grouping/filtering of noise itself.
func (h *serviceHandler) PeerEventList(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	peerAddress *commonaddress.Address,
	pageToken string,
	pageSize uint64,
) ([]*tmpeerevent.PeerEvent, string, error) {
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	addrs, err := h.resolvePeerAddresses(ctx, a.CustomerID, contactID, peerAddress)
	if err != nil {
		return nil, "", err
	}
	if len(addrs) == 0 {
		return nil, "", nil // no addresses on this contact -> empty result, no RPC call
	}

	req := &tmpeerevent.PeerEventListRequest{
		CustomerID:    a.CustomerID,
		PeerAddresses: addrs,
		PageToken:     pageToken,
		PageSize:      int(pageSize),
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
	peerAddress *commonaddress.Address,
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

	addrs, err := h.resolvePeerAddresses(ctx, a.CustomerID, contactID, peerAddress)
	if err != nil {
		return nil, "", err
	}
	if len(addrs) == 0 {
		return nil, "", nil
	}

	req := &tmpeerevent.PeerEventListRequest{
		CustomerID:    a.CustomerID,
		PeerAddresses: addrs,
		PageToken:     pageToken,
		PageSize:      int(pageSize),
	}
	res, err := h.reqHandler.TimelineV1PeerEventList(ctx, req)
	if err != nil {
		return nil, "", err
	}
	return res.Result, res.NextPageToken, nil
}

// resolvePeerAddresses implements the "exactly one filter" contract:
// contact_id resolves via contactGet + tenant check, deduping
// Contact.Addresses directly (no dbhandler.PeerPairFilter/PeerPair
// intermediate type anymore — commonaddress.Address is used end-to-end
// from bin-api-manager through bin-common-handler to bin-timeline-manager);
// OR peerAddress is a single-address passthrough. The HTTP layer
// (server/contact_peer_events.go) already enforces exactly-one-filter via
// filterCount, so in practice this is never reached with both filters set;
// the switch's contactID-first ordering is purely an implementation detail
// of that unreachable-in-practice case.
func (h *serviceHandler) resolvePeerAddresses(
	ctx context.Context,
	customerID uuid.UUID,
	contactID uuid.UUID,
	peerAddress *commonaddress.Address,
) ([]commonaddress.Address, error) {
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

		type typeTarget struct {
			Type   commonaddress.Type
			Target string
		}
		seen := make(map[typeTarget]struct{})
		var addrs []commonaddress.Address
		for _, ca := range ct.Addresses {
			key := typeTarget{Type: ca.Type, Target: ca.Target}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			addrs = append(addrs, commonaddress.Address{Type: ca.Type, Target: ca.Target})
		}
		return addrs, nil

	case peerAddress != nil && peerAddress.Type != "" && peerAddress.Target != "":
		return []commonaddress.Address{{Type: peerAddress.Type, Target: peerAddress.Target}}, nil

	default:
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager, "INVALID_FILTER",
			"Exactly one filter is required: contact_id, or peer_type+peer_target.",
		)
	}
}
