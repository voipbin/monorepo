package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	cminteraction "monorepo/bin-contact-manager/models/interaction"
	cmresolution "monorepo/bin-contact-manager/models/resolution"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// ContactV1InteractionGet sends a request to contact-manager to get a single interaction.
func (r *requestHandler) ContactV1InteractionGet(ctx context.Context, customerID, id uuid.UUID) (*cminteraction.Interaction, error) {
	uri := fmt.Sprintf("/v1/interactions/%s", id)

	m, err := json.Marshal(map[string]string{"customer_id": customerID.String()})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/interactions/<id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cminteraction.Interaction
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1InteractionList lists interactions from contact-manager.
// Exactly one of (peerType+peerTarget), contactID, or addressID should be non-zero.
func (r *requestHandler) ContactV1InteractionList(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID, addressID uuid.UUID,
) ([]*cminteraction.Interaction, string, error) {
	u := url.Values{}
	if peerType != "" {
		u.Set("peer_type", peerType)
	}
	if peerTarget != "" {
		u.Set("peer_target", peerTarget)
	}
	if contactID != uuid.Nil {
		u.Set("contact_id", contactID.String())
	}
	if addressID != uuid.Nil {
		u.Set("address_id", addressID.String())
	}
	if size > 0 {
		u.Set("page_size", fmt.Sprintf("%d", size))
	}
	if token != "" {
		u.Set("page_token", token)
	}

	uri := "/v1/interactions?" + u.Encode()

	m, err := json.Marshal(map[string]string{"customer_id": customerID.String()})
	if err != nil {
		return nil, "", err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/interactions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, "", err
	}

	var res struct {
		Result        []*cminteraction.Interaction `json:"result"`
		NextPageToken string                       `json:"next_page_token"`
	}
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, "", errParse
	}

	return res.Result, res.NextPageToken, nil
}

// ContactV1InteractionListUnresolved lists unresolved interactions from contact-manager.
// since specifies the lookback window in "Nd" format (e.g. "7d", "30d"). Empty string uses default (30d).
// Maximum is "180d" — enforced by the listenhandler.
func (r *requestHandler) ContactV1InteractionListUnresolved(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	since string,
) ([]*cminteraction.Interaction, string, error) {
	u := url.Values{}
	if since != "" {
		u.Set("since", since)
	}
	if size > 0 {
		u.Set("page_size", fmt.Sprintf("%d", size))
	}
	if token != "" {
		u.Set("page_token", token)
	}

	uri := "/v1/interactions/unresolved"
	if len(u) > 0 {
		uri = uri + "?" + u.Encode()
	}

	m, err := json.Marshal(map[string]string{"customer_id": customerID.String()})
	if err != nil {
		return nil, "", err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/interactions/unresolved", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, "", err
	}

	var res struct {
		Result        []*cminteraction.Interaction `json:"result"`
		NextPageToken string                       `json:"next_page_token"`
	}
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, "", errParse
	}

	return res.Result, res.NextPageToken, nil
}

// ContactV1ResolutionCreate creates a resolution in contact-manager.
func (r *requestHandler) ContactV1ResolutionCreate(
	ctx context.Context,
	customerID, contactID, interactionID uuid.UUID,
	resolutionType, resolvedByType string,
	resolvedByID uuid.UUID,
) (*cmresolution.Resolution, error) {
	uri := fmt.Sprintf("/v1/interactions/%s/resolutions", interactionID)

	data := &cmrequest.V1DataInteractionsResolutionsPost{
		CustomerID:     customerID,
		ContactID:      contactID,
		ResolutionType: resolutionType,
		ResolvedByType: resolvedByType,
		ResolvedByID:   resolvedByID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/interactions/<id>/resolutions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmresolution.Resolution
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ResolutionDelete soft-deletes a resolution in contact-manager.
func (r *requestHandler) ContactV1ResolutionDelete(
	ctx context.Context,
	customerID uuid.UUID,
	interactionID, resolutionID uuid.UUID,
) error {
	uri := fmt.Sprintf("/v1/interactions/%s/resolutions/%s", interactionID, resolutionID)

	m, err := json.Marshal(cmrequest.V1DataInteractionsResolutionsIDDelete{CustomerID: customerID})
	if err != nil {
		return err
	}

	_, err = r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/interactions/<id>/resolutions/<rid>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	return err
}
