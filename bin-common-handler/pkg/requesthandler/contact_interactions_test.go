package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
)

// interactionListResult flattens the three return values of ContactV1InteractionList
// into comparable literals. Timestamps are rendered in UTC so the comparison does not
// depend on the time zone that the decoded values carry.
type interactionListResult struct {
	ReferenceIDs []string
	Timestamps   []string
	NextToken    string
}

func interactionListCall(h *requestHandler, customerID uuid.UUID, size uint64, token string, peerType, peerTarget string, contactID, addressID uuid.UUID) (any, error) {
	res, next, err := h.ContactV1InteractionList(context.Background(), customerID, size, token, peerType, peerTarget, contactID, addressID)
	if err != nil {
		return nil, err
	}

	out := interactionListResult{ReferenceIDs: []string{}, Timestamps: []string{}, NextToken: next}
	for _, e := range res {
		out.ReferenceIDs = append(out.ReferenceIDs, e.ReferenceID.String())
		out.Timestamps = append(out.Timestamps, e.Timestamp.UTC().Format("2006-01-02T15:04:05.000000Z"))
	}
	return out, nil
}

func Test_ContactV1InteractionList(t *testing.T) {
	customerID := uuid.FromStringOrNil("6a7b8c9d-0e1f-4a2b-9c3d-5e6f70819203")
	contactID := uuid.FromStringOrNil("7b8c9d0e-1f2a-4b3c-8d4e-6f7081920314")
	addressID := uuid.FromStringOrNil("8c9d0e1f-2a3b-4c4d-9e5f-708192031425")

	const body = `{"customer_id":"6a7b8c9d-0e1f-4a2b-9c3d-5e6f70819203"}`

	// two events. The first one is later than the last one, so the token must be taken from
	// the last element of the list and not from the newest timestamp. The last timestamp has a
	// non-UTC offset, so the token also proves the conversion to UTC.
	const twoEvents = `[` +
		`{"timestamp":"2026-03-04T05:06:07.123456Z","reference_id":"9d0e1f2a-3b4c-4d5e-8f60-819203142536"},` +
		`{"timestamp":"2026-03-04T14:00:00.654321+09:00","reference_id":"0e1f2a3b-4c5d-4e6f-9071-92031425364a"}` +
		`]`
	twoEventsWant := interactionListResult{
		ReferenceIDs: []string{"9d0e1f2a-3b4c-4d5e-8f60-819203142536", "0e1f2a3b-4c5d-4e6f-9071-92031425364a"},
		Timestamps:   []string{"2026-03-04T05:06:07.123456Z", "2026-03-04T05:00:00.654321Z"},
		NextToken:    "2026-03-04T05:00:00.654321Z",
	}

	newCase := func(name, query string, responseData string, expect interactionListResult, size uint64, token string, peerType, peerTarget string, cID, aID uuid.UUID) rpcCase {
		return rpcCase{
			name:           name,
			queue:          contactQueue,
			expectURI:      "/v1/interactions" + query,
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     body,
			responseData:   responseData,
			call: func(h *requestHandler) (any, error) {
				return interactionListCall(h, customerID, size, token, peerType, peerTarget, cID, aID)
			},
			expectRes: expect,
		}
	}

	oneEvent := `[{"timestamp":"2026-03-04T05:06:07.123456Z","reference_id":"9d0e1f2a-3b4c-4d5e-8f60-819203142536"}]`
	oneEventWant := interactionListResult{
		ReferenceIDs: []string{"9d0e1f2a-3b4c-4d5e-8f60-819203142536"},
		Timestamps:   []string{"2026-03-04T05:06:07.123456Z"},
		NextToken:    "2026-03-04T05:06:07.123456Z",
	}

	empty := interactionListResult{ReferenceIDs: []string{}, Timestamps: []string{}, NextToken: ""}

	cases := []rpcCase{
		newCase("peer_all_params",
			"?address_id=8c9d0e1f-2a3b-4c4d-9e5f-708192031425&contact_id=7b8c9d0e-1f2a-4b3c-8d4e-6f7081920314&page_size=25&page_token=token-1&peer_target=user%40example.com&peer_type=email",
			twoEvents, twoEventsWant, 25, "token-1", "email", "user@example.com", contactID, addressID),
		newCase("peer_only", "?peer_target=%2B15550100001&peer_type=tel", twoEvents, twoEventsWant, 0, "", "tel", "+15550100001", uuid.Nil, uuid.Nil),
		newCase("contact_only", "?contact_id=7b8c9d0e-1f2a-4b3c-8d4e-6f7081920314", twoEvents, twoEventsWant, 0, "", "", "", contactID, uuid.Nil),
		newCase("address_only", "?address_id=8c9d0e1f-2a3b-4c4d-9e5f-708192031425", twoEvents, twoEventsWant, 0, "", "", "", uuid.Nil, addressID),
		newCase("paging_only", "?page_size=5&page_token=t-2", twoEvents, twoEventsWant, 5, "t-2", "", "", uuid.Nil, uuid.Nil),
		newCase("page_size_one", "?page_size=1", twoEvents, twoEventsWant, 1, "", "", "", uuid.Nil, uuid.Nil),
		newCase("single_event_has_next_token", "?contact_id=7b8c9d0e-1f2a-4b3c-8d4e-6f7081920314", oneEvent, oneEventWant, 0, "", "", "", contactID, uuid.Nil),
		newCase("no_filters", "?", twoEvents, twoEventsWant, 0, "", "", "", uuid.Nil, uuid.Nil),
		newCase("empty_list_has_no_next_token", "?contact_id=7b8c9d0e-1f2a-4b3c-8d4e-6f7081920314", `[]`, empty, 0, "", "", "", contactID, uuid.Nil),
		newCase("empty_response_data_has_no_next_token", "?contact_id=7b8c9d0e-1f2a-4b3c-8d4e-6f7081920314", ``, empty, 0, "", "", "", contactID, uuid.Nil),
	}

	runRPCCases(t, cases)
}
