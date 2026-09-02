package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// insightDefaultListLimit / insightMaxListLimit bound the "limit" argument
// accepted by both Insight tools. Kept smaller than resourceListPageSize
// (get_resource's enrichment page size) because these are Insight-facing
// summary lists, not a full resource dump.
const (
	insightDefaultListLimit = 20
	insightMaxListLimit     = 50
)

// insightContactAddressLimit caps how many of a contact's reachable
// addresses get_contact_profile renders. Deliberately NOT insightMaxListLimit
// (50): a profile record is not a list tool, and 50 phone numbers is not a
// plausible answer to "what is this person's phone number". insightMaxListLimit
// stays reserved for the actual list tools (interactions/related cases/notes).
// Design docs/plans/2026-09-02-insight-assistant-get-contact-profile-design.md §3.4.
const insightContactAddressLimit = 5

// resolveInsightListLimit clamps an LLM-supplied limit into
// [1, insightMaxListLimit], defaulting to insightDefaultListLimit when unset
// or non-positive.
func resolveInsightListLimit(v int) uint64 {
	if v <= 0 {
		return insightDefaultListLimit
	}
	if v > insightMaxListLimit {
		return insightMaxListLimit
	}
	return uint64(v)
}

// isNotFoundErr reports whether err represents a "not found" outcome from a
// downstream RPC, across BOTH error shapes this codebase's managers use:
//   - legacy: the bare requesthandler.ErrNotFound sentinel (status-code
//     fallback path in parseResponse), used by e.g. ContactV1CaseGet.
//   - typed: a *cerrors.VoipbinError with Status == cerrors.StatusNotFound
//     (the migrated envelope, cerrors.FromResponse takes precedence over
//     the legacy sentinel in parseResponse), used by e.g. contact-manager's
//     interactionListByContact (CONTACT_NOT_FOUND, e.g. when the Contact
//     backing a Case has been soft-deleted).
//
// Round-2 adversarial review (VOIP-1234 PR #1100) found that checking only
// the legacy sentinel silently misclassified a typed NotFound (a routine,
// user-facing "no history yet" outcome) as an honest RPC failure. Both
// shapes are checked here so every not-found path -- regardless of which
// migration state the downstream manager is in -- is treated identically.
func isNotFoundErr(err error) bool {
	if stderrors.Is(err, requesthandler.ErrNotFound) {
		return true
	}
	var ve *cerrors.VoipbinError
	return stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound
}

// toolHandleGetContactInteractions lists past interactions (calls,
// conversation messages) tied to the Case's peer/contact. Scope is always
// the current Insight AIcall's own Case (c.ReferenceID) -- there is no
// case_id/contact_id argument (design VOIP-1234 §3: implicit scoping
// removes the IDOR-shaped bug class entirely rather than defending against
// it).
func (h *aicallHandler) toolHandleGetContactInteractions(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetContactInteractions",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_contact_interactions.")

	res := newToolResult(tc.ID)

	var args struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, errors.Wrap(err, "invalid arguments"))
		return res
	}
	limit := resolveInsightListLimit(args.Limit)

	if c.ReferenceType != aicall.ReferenceTypeContactCase {
		fillFailed(res, fmt.Errorf("get_contact_interactions is only supported for contact_case reference type"))
		return res
	}

	kase, err := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "interaction_list", c.ReferenceID.String(), msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the case. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil {
		// Defensive: tenant is already embedded in the RPC, but fail closed
		// on any mismatch rather than trust a foreign response shape.
		log.Warnf("Cross-customer case access blocked. case_customer_id: %s", kase.CustomerID)
		fillSuccess(res, "interaction_list", c.ReferenceID.String(), msgResourceNotFound)
		return res
	}

	var interactions []*tmpeerevent.PeerEvent
	if kase.ContactID != nil {
		interactions, _, err = h.reqHandler.ContactV1InteractionList(
			ctx, c.CustomerID, limit, "", "", "", *kase.ContactID, uuid.Nil, time.Time{})
	} else {
		interactions, _, err = h.reqHandler.ContactV1InteractionList(
			ctx, c.CustomerID, limit, "", string(kase.Peer.Type), kase.Peer.Target, uuid.Nil, uuid.Nil, time.Time{})
	}
	if err != nil {
		// Round-2 review finding (VOIP-1234 PR #1100): the Contact backing
		// this Case may have been soft-deleted (merge, GDPR erasure, etc.)
		// since the Case was created. contact-manager's interactionListByContact
		// returns a TYPED NotFound (CONTACT_NOT_FOUND) in that case, which is a
		// routine "no history to show" outcome for this tool -- not a genuine
		// downstream failure. Treat it as an empty result (success), same as
		// the Case-not-found path above, rather than an honest tool failure.
		if isNotFoundErr(err) {
			fillSuccess(res, "interaction_list", c.ReferenceID.String(), "no interactions found")
			return res
		}
		log.Errorf("Could not list interactions. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	if len(interactions) == 0 {
		fillSuccess(res, "interaction_list", c.ReferenceID.String(), "no interactions found")
		return res
	}

	lines := make([]string, 0, len(interactions))
	for _, it := range interactions {
		ts := it.Timestamp.UTC().Format(time.RFC3339)
		lines = append(lines, fmt.Sprintf(
			"[%s] direction=%s peer=%s/%s reference_type=%s reference_id=%s",
			ts, it.Direction, it.Peer.Type, it.Peer.Target, it.Publisher, it.ReferenceID,
		))
	}

	body := renderBodyLines("", lines, uint64(len(interactions)) >= limit, "interactions")
	fillSuccess(res, "interaction_list", c.ReferenceID.String(), body)
	return res
}

// toolHandleGetConversationContent retrieves the message transcript of a
// conversation, given the reference_id of a conversation_message-type
// interaction the LLM discovered via get_contact_interactions (design
// VOIP-1234 §5: explicit target selection, not an implicit
// server-picks-the-most-recent-thread auto-resolution).
//
// Resolution is a FIXED 2 RPC calls regardless of message/thread count:
//  1. ConversationV1MessageGet(reference_id) -- resolves the message AND is
//     the ownership/IDOR check (reference_id is now LLM-suppliable, unlike
//     the implicit Case scoping used elsewhere in this file).
//  2. ConversationV1MessageList(filters={conversation_id}) -- one list call
//     for the whole surrounding thread, capped at limit.
func (h *aicallHandler) toolHandleGetConversationContent(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetConversationContent",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_conversation_content.")

	res := newToolResult(tc.ID)

	var args struct {
		ReferenceID string `json:"reference_id"`
		Limit       int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, errors.Wrap(err, "invalid arguments"))
		return res
	}
	if args.ReferenceID == "" {
		fillFailed(res, fmt.Errorf("reference_id is required, call get_contact_interactions first to discover candidate ids"))
		return res
	}
	refID, err := uuid.FromString(args.ReferenceID)
	if err != nil || refID == uuid.Nil {
		fillFailed(res, fmt.Errorf("invalid reference_id"))
		return res
	}
	limit := resolveInsightListLimit(args.Limit)

	// RPC 1/2: resolve the message + ownership check (single masking site,
	// mirrors resolveResource's IDOR-safe contract in tool_resource.go).
	msg, err := h.reqHandler.ConversationV1MessageGet(ctx, refID)
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "conversation_content", args.ReferenceID, msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the message. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if msg == nil || msg.CustomerID != c.CustomerID || msg.CustomerID == uuid.Nil {
		log.Warnf("Cross-customer message access blocked. reference_id: %s", refID)
		fillSuccess(res, "conversation_content", args.ReferenceID, msgResourceNotFound)
		return res
	}

	// RPC 2/2: one list call filtered by conversation_id -- NOT a per-message
	// fetch loop. This is the fixed-cost path decided in design VOIP-1234 §5
	// after the original N+1 draft was rejected as wasteful.
	filters := map[cvmessage.Field]any{
		cvmessage.FieldConversationID: msg.ConversationID.String(),
	}
	msgs, err := h.reqHandler.ConversationV1MessageList(ctx, "", limit, filters)
	if err != nil {
		log.Errorf("Could not list conversation messages. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	if len(msgs) == 0 {
		fillSuccess(res, "conversation_content", args.ReferenceID, "no messages found")
		return res
	}

	lines := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ts := "unknown"
		if m.TMCreate != nil {
			ts = m.TMCreate.UTC().Format(time.RFC3339)
		}
		lines = append(lines, fmt.Sprintf("[%s %s] %s", ts, m.Direction, m.Text))
	}

	body := renderBodyLines("", lines, uint64(len(msgs)) >= limit, "messages")
	fillSuccess(res, "conversation_content", args.ReferenceID, body)
	return res
}

// toolHandleGetRelatedCases lists OTHER cases belonging to the same contact
// as the current Insight AIcall's Case -- metadata only (id/title/status/
// created_at), never case body or notes. Design docs/plans/
// 2026-07-30-case-insight-assistant-tool-expansion-design.md §1.1.
//
// The nil/zero-UUID ContactID branch below is NOT an optional nicety: this
// codebase's ContactV1CaseList silently DROPS its contact_id filter when
// given uuid.Nil (bin-common-handler/pkg/requesthandler/contact_cases.go,
// `if contactID != uuid.Nil { ... }`), which would otherwise return every
// case for the tenant instead of "no related contact". The RPC must never
// be reached in that case.
func (h *aicallHandler) toolHandleGetRelatedCases(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetRelatedCases",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_related_cases.")

	res := newToolResult(tc.ID)

	if c.ReferenceType != aicall.ReferenceTypeContactCase {
		fillFailed(res, fmt.Errorf("get_related_cases is only supported for contact_case reference type"))
		return res
	}

	kase, err := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "related_case_list", c.ReferenceID.String(), msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the case. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil {
		// Defensive: tenant is already embedded in the RPC, but fail closed
		// on any mismatch rather than trust a foreign response shape.
		log.Warnf("Cross-customer case access blocked. case_customer_id: %s", kase.CustomerID)
		fillSuccess(res, "related_case_list", c.ReferenceID.String(), msgResourceNotFound)
		return res
	}

	if kase.ContactID == nil || *kase.ContactID == uuid.Nil {
		// Legitimate outcome (no linked contact) -- not a denial, no audit
		// log needed. Never reach ContactV1CaseList with a nil contact id.
		fillSuccess(res, "related_case_list", c.ReferenceID.String(), "no related cases found")
		return res
	}

	// The pagination token is intentionally discarded: this tool always
	// fetches a single bounded page (insightMaxListLimit) and reports
	// truncation via the "pagedOut" flag below rather than paging through
	// the full history -- an Insight tool call is a single LLM function
	// invocation, not a paginated list endpoint.
	cases, _, err := h.reqHandler.ContactV1CaseList(
		ctx, c.CustomerID, "", "", uuid.Nil, *kase.ContactID, insightMaxListLimit, "", "")
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "related_case_list", c.ReferenceID.String(), "no related cases found")
			return res
		}
		log.Errorf("Could not list related cases. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	// The truncation flag must reflect whether the RAW fetch (pre-exclusion)
	// hit the page size cap -- that's the only reliable signal that more
	// rows may exist beyond this page. Computing it from the post-exclusion
	// count under-reports: if the raw page was exactly insightMaxListLimit
	// rows and one of them happened to be the current case, the excluded
	// count would fall one short of the limit and "truncated" would go
	// unset even though further related cases may exist beyond this page.
	// (Round-2 code review finding; supersedes the design doc's original
	// "exclude first" framing in design §1.1 row 6, which named the right
	// ordering for exclusion but the wrong basis for the truncation flag.)
	pagedOut := uint64(len(cases)) >= insightMaxListLimit

	filtered := make([]*kmkase.Case, 0, len(cases))
	for _, cs := range cases {
		if cs.ID == c.ReferenceID {
			continue
		}
		filtered = append(filtered, cs)
	}

	if len(filtered) == 0 {
		fillSuccess(res, "related_case_list", c.ReferenceID.String(), "no related cases found")
		return res
	}

	lines := make([]string, 0, len(filtered))
	for _, cs := range filtered {
		ts := "unknown"
		if cs.TMCreate != nil {
			ts = cs.TMCreate.UTC().Format(time.RFC3339)
		}
		lines = append(lines, fmt.Sprintf("[%s] case_id=%s title=%q status=%s", ts, cs.ID, cs.Name, cs.Status))
	}

	body := renderBodyLines("", lines, pagedOut, "related cases")
	fillSuccess(res, "related_case_list", c.ReferenceID.String(), body)
	return res
}

// toolHandleGetCaseNotes returns internal agent notes on the CURRENT case
// (not other cases) -- useful for handoffs between agents. Design
// docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md §1.2.
func (h *aicallHandler) toolHandleGetCaseNotes(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetCaseNotes",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_case_notes.")

	res := newToolResult(tc.ID)

	var args struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, errors.Wrap(err, "invalid arguments"))
		return res
	}
	limit := resolveInsightListLimit(args.Limit)

	if c.ReferenceType != aicall.ReferenceTypeContactCase {
		fillFailed(res, fmt.Errorf("get_case_notes is only supported for contact_case reference type"))
		return res
	}

	kase, err := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "case_note_list", c.ReferenceID.String(), msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the case. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil {
		log.Warnf("Cross-customer case access blocked. case_customer_id: %s", kase.CustomerID)
		fillSuccess(res, "case_note_list", c.ReferenceID.String(), msgResourceNotFound)
		return res
	}

	// ContactV1CaseNoteList takes no size/token args (unlike ContactV1CaseList
	// above); the caller-side clamp is the only limit enforcement here.
	notes, err := h.reqHandler.ContactV1CaseNoteList(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "case_note_list", c.ReferenceID.String(), "no notes found")
			return res
		}
		log.Errorf("Could not list case notes. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	if len(notes) == 0 {
		fillSuccess(res, "case_note_list", c.ReferenceID.String(), "no notes found")
		return res
	}

	// CaseNoteListByCase (bin-contact-manager/pkg/dbhandler/casenote.go)
	// orders notes by tm_create ASC (oldest first). "Most recent N" is
	// therefore the TAIL of this slice, not the head.
	truncated := uint64(len(notes)) > limit
	start := 0
	if truncated {
		start = len(notes) - int(limit)
	}
	recent := notes[start:]

	lines := make([]string, 0, len(recent))
	for _, n := range recent {
		ts := "unknown"
		if n.TMCreate != nil {
			ts = n.TMCreate.UTC().Format(time.RFC3339)
		}
		lines = append(lines, fmt.Sprintf("[%s] author_type=%s: %s", ts, n.AuthorType, n.Text))
	}

	body := renderBodyLines("", lines, truncated, "notes")
	fillSuccess(res, "case_note_list", c.ReferenceID.String(), body)
	return res
}

// contactDisplayName resolves a human-readable name for a contact: prefer
// DisplayName, fall back to "FirstName LastName", and finally the fixed
// "(unknown)" placeholder. The placeholder is a literal, not user data, so
// it is deliberately returned unquoted here -- the caller %q-quotes whatever
// this returns, which is correct for all three cases.
func contactDisplayName(contact *cmcontact.Contact) string {
	if name := strings.TrimSpace(contact.DisplayName); name != "" {
		return name
	}
	if name := strings.TrimSpace(contact.FirstName + " " + contact.LastName); name != "" {
		return name
	}
	return "(unknown)"
}

// toolHandleGetContactProfile returns the profile (name/company/job title and
// a capped set of reachable addresses) of the contact linked to the CURRENT
// Insight AIcall's Case. Design docs/plans/
// 2026-09-02-insight-assistant-get-contact-profile-design.md.
//
// Scope is always c.ReferenceID (the current Case) -- there is no contact_id
// or case_id argument, and the tool parses no arguments at all. No
// LLM-suppliable identifier exists anywhere in this tool's call surface, which
// removes the IDOR-shaped bug class by construction (design §3.1).
func (h *aicallHandler) toolHandleGetContactProfile(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetContactProfile",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_contact_profile.")

	res := newToolResult(tc.ID)

	if c.ReferenceType != aicall.ReferenceTypeContactCase {
		fillFailed(res, fmt.Errorf("get_contact_profile is only supported for contact_case reference type"))
		return res
	}

	// RPC 1/2: ContactV1CaseGet IS genuinely tenant-scoped server-side (it
	// takes customerID), so this both resolves the Case and establishes that
	// the ContactID read off it belongs to this tenant.
	kase, err := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "contact_profile", c.ReferenceID.String(), msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the case. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil {
		// Defensive: tenant is already embedded in the RPC, but fail closed
		// on any mismatch rather than trust a foreign response shape.
		log.Warnf("Cross-customer case access blocked. case_customer_id: %s", kase.CustomerID)
		fillSuccess(res, "contact_profile", c.ReferenceID.String(), msgResourceNotFound)
		return res
	}

	if kase.ContactID == nil || *kase.ContactID == uuid.Nil {
		// Legitimate outcome (no linked contact) -- not a denial, so this
		// stays a distinct, non-masked message rather than collapsing into
		// msgResourceNotFound (same precedent as get_related_cases). Also a
		// hard guard: ContactV1ContactGet must NEVER be reached with a nil or
		// zero contact id (design §3.2 step 3).
		fillSuccess(res, "contact_profile", c.ReferenceID.String(), "no contact profile found")
		return res
	}

	// RPC 2/2: ContactV1ContactGet takes NO customerID and does NO server-side
	// tenant filtering (contacthandler.Get only rejects soft-deleted rows).
	contact, err := h.reqHandler.ContactV1ContactGet(ctx, *kase.ContactID)
	if err != nil {
		if isNotFoundErr(err) {
			// Includes the soft-deleted contact case (merge, GDPR erasure).
			fillSuccess(res, "contact_profile", c.ReferenceID.String(), msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the contact. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	// MANDATORY tenant enforcement -- NOT defensive, NOT redundant, and NOT
	// removable. ContactV1ContactGet above has no server-side tenant filter of
	// any kind, so this response-side check is the SOLE thing preventing
	// cross-tenant contact disclosure on this path. It is safe today only
	// because *kase.ContactID was read off a Case already tenant-verified
	// above -- i.e. tenancy here rests on contact_cases.contact_id referential
	// integrity, not on the RPC. Anyone changing either table's write path, or
	// tempted to delete this block as "already covered upstream", must read
	// design §3.2 step 5 / §5 first. Deleting it is a real cross-tenant leak.
	if contact == nil || contact.CustomerID != c.CustomerID || contact.CustomerID == uuid.Nil {
		// Nil-safe forensic audit line: contact may legitimately be nil here,
		// so the offending tenant id is resolved through a guarded local.
		foreignCustomerID := "nil-contact"
		if contact != nil {
			foreignCustomerID = contact.CustomerID.String()
		}
		log.Warnf("Cross-customer or missing contact on lookup. case_id: %s, contact_id: %s, contact_customer_id: %s",
			c.ReferenceID, *kase.ContactID, foreignCustomerID)
		fillSuccess(res, "contact_profile", c.ReferenceID.String(), msgResourceNotFound)
		return res
	}

	// Identity fields go in the HEADER, never in `lines`: renderBodyLines is
	// contracted for a chronological, homogeneous list and on overflow drops
	// from the FRONT of `lines`. Mixing identity fields into `lines` would
	// sacrifice name/company first while keeping raw phone/email lines --
	// both the wrong UX and privacy-inverted (design §3.4, round-2 finding).
	headerLines := []string{fmt.Sprintf("name=%q", contactDisplayName(contact))}
	if contact.Company != "" {
		headerLines = append(headerLines, fmt.Sprintf("company=%q", contact.Company))
	}
	if contact.JobTitle != "" {
		headerLines = append(headerLines, fmt.Sprintf("job_title=%q", contact.JobTitle))
	}

	// Defense in depth FIRST, cap SECOND (round-1 code review MEDIUM fix):
	// filtering after capping could (a) make the truncation note below claim
	// "showing N of M" while fewer than N lines actually render, once a
	// capped-in row is later dropped by the tenant check, and (b) needlessly
	// discard a valid address at position 6+ to make room for a row that gets
	// filtered anyway. Each address row carries its own CustomerID, so skip
	// any row that does not match rather than trusting the parent contact's
	// tenant check to cover every nested row.
	filtered := make([]cmcontact.Address, 0, len(contact.Addresses))
	for _, a := range contact.Addresses {
		if a.CustomerID != uuid.Nil && a.CustomerID != c.CustomerID {
			log.Warnf("Skipping cross-customer contact address row. case_id: %s, address_customer_id: %s", c.ReferenceID, a.CustomerID)
			continue
		}
		filtered = append(filtered, a)
	}

	addrs := filtered
	if len(addrs) > insightContactAddressLimit {
		// Head-cap, which is meaningful only because AddressListByContactID
		// orders rows `is_primary desc, tm_create asc`
		// (bin-contact-manager/pkg/dbhandler/address.go) -- so the cap is
		// primary-preserving by construction, not an arbitrary truncation.
		addrs = addrs[:insightContactAddressLimit]

		// Write our OWN honest, ordering-agnostic truncation note into the
		// header instead of letting renderBodyLines emit its built-in
		// "...(earlier addresses omitted; showing the most recent N)" marker.
		// That marker asserts recency, which would be a false claim about a
		// primary-first list, and renderBodyLines forces it whenever
		// pagedOut=true regardless of actual overflow -- i.e. it would fire on
		// the ordinary common path, not a rare edge case (design §3.4 rev5).
		// len(filtered) (not len(contact.Addresses)) is the honest total: it
		// is post-tenant-filter, so the note can never overclaim what's shown.
		headerLines = append(headerLines, fmt.Sprintf(
			"(showing %d of %d addresses)", len(addrs), len(filtered)))
	}
	header := strings.Join(headerLines, "\n")

	addrLines := make([]string, 0, len(addrs))
	for _, a := range addrs {
		// Type + Target ONLY -- never the free-text Detail/Name/TargetName
		// sub-fields (same sensitivity class as Notes; design §5).
		addrLines = append(addrLines, fmt.Sprintf("address: type=%s target=%q", a.Type, a.Target))
	}

	var body string
	if len(addrLines) == 0 {
		// renderBodyLines' own empty-lines fallback does exactly this, but
		// calling it explicitly documents the no-addresses case.
		body = capSummaryRunes(header)
	} else {
		// pagedOut is hardcoded false (NOT the truncation condition above):
		// our own note is already in the header, and renderBodyLines' marker
		// would be misleading here. See the header comment above.
		body = renderBodyLines(header, addrLines, false, "addresses")
	}

	fillSuccess(res, "contact_profile", c.ReferenceID.String(), body)
	return res
}
