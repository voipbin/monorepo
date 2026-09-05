package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

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

// insightCallTranscribeSessionLimit / insightCallTranscribeFetchMaxPages
// bound get_call_transcript's fetch. insightCallTranscribeSessionLimit is a
// NEW, separate policy-bound constant (not insightMaxListLimit=50, not
// derived from any codebase-enforced constraint) for the CUSTOMER-VISIBLE
// number of transcribe sessions rendered -- raise if telemetry shows
// truncation. insightCallTranscribeFetchMaxPages bounds the pagination loop
// (§3.3/§3.4) that determines whether the true genuine count is known, not
// how many rows are ultimately rendered. Design
// docs/plans/2026-09-03-insight-assistant-get-call-transcript-design.md §3.3.
const (
	insightCallTranscribeSessionLimit  = 10
	insightCallTranscribeFetchMaxPages = 5
)

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

// transcriptLine is the intermediate accumulator entry for
// toolHandleGetCallTranscript's cross-session merge (§3.5): one entry per
// real Transcript row, plus one entry per synthesized gap marker (§3.6).
// After the full merge is assembled it is sorted with sort.SliceStable on
// the strict total-order key (tmCreate, rank, transcribeID, transcriptID,
// seq), THEN rendered to its final display string (§3.7). Design
// docs/plans/2026-09-03-insight-assistant-get-call-transcript-design.md §3.5.
type transcriptLine struct {
	tmCreate     *time.Time // sort key component 1; also the rendered timestamp
	rank         int        // sort key component 2: 0 = gap marker, 1 = real row
	transcribeID uuid.UUID  // sort key component 3 (uuid.Nil for markers)
	transcriptID uuid.UUID  // sort key component 4 (uuid.Nil for markers)
	seq          int        // sort key component 5, monotonic per-call counter
	bracket      string     // "<direction> <language>" (or "gap <language>"),
	// the content that goes INSIDE the rendered [<ts> ...] brackets
	message string // the free text after the closing bracket
}

// toolHandleGetCallTranscript returns the merged, chronological transcript
// of everything said on a call, sourced from live in-call transcription
// (transcribe_start) sessions. Design docs/plans/
// 2026-09-03-insight-assistant-get-call-transcript-design.md.
//
// Access is TENANT-ONLY (§3.2), a conscious, explicit product decision
// (§0.2) to widen from case-level to tenant-level isolation, consistent with
// the already-shipped get_conversation_content precedent: call_id is
// LLM-suppliable, and this tool does not narrow to the current Case's
// peer/contact.
// paginateUntilExact fetches pages of T from fetch, keeping only items for
// which keep returns true, until it has unambiguous proof of either
// overflow (more than `limit` genuine items exist) or exhaustion (the
// source is genuinely out of rows) -- or gives up honestly at `maxPages`.
// See docs/plans/2026-09-03-insight-assistant-get-call-transcript-design.md
// §3.3 (lines ~103-220) for the full history of how this invariant was
// derived across 11 design-review rounds; do not change the comparators
// below without re-reading that section first.
//
// capped is true iff genuine overflow occurred OR the loop could not
// obtain proof within maxPages (the caller must treat this as "possibly
// incomplete", not "definitely not capped" -- see the design doc's
// possiblyIncomplete discussion).
//
// SECURITY: `keep` is the ONLY tenant/deletion enforcement this helper
// performs -- paginateUntilExact itself does no filtering by identity. Every
// call site MUST independently re-verify ownership fields (CustomerID,
// ReferenceID/TranscribeID, TMDelete) inside `keep`, mirroring §3.3/§3.4 of
// the design doc, because the underlying RPC's filter map is caller-supplied
// and NOT server-enforced. Do not add a new call site whose `keep` trusts
// the RPC's own filtering.
//
// `limit` MUST match the `+1` sentinel size actually requested inside
// `fetch` (i.e. fetch should request `limit+1` items per page) -- the
// pagesExhausted/overflow proof below depends on that exact coupling, and
// nothing in this function's signature enforces it. If a future edit
// changes one without the other, `capped` silently stops meaning what it
// claims to mean.
func paginateUntilExact[T any](
	ctx context.Context,
	maxPages int,
	limit uint64,
	fetch func(ctx context.Context, pageToken string) ([]T, error),
	tmCreateOf func(T) *time.Time,
	keep func(T) bool,
) (verified []T, capped bool, err error) {
	pageToken := ""
	pagesExhausted := false // true only once a short page PROVES no more source rows remain
	for page := 0; page < maxPages; page++ {
		items, ferr := fetch(ctx, pageToken)
		if ferr != nil {
			// Whole-call-failure vs. degrade-one-unit-visibly is a POLICY
			// decision that varies by caller (fail the whole tool vs. skip
			// just this unit) -- deliberately left to the caller, not
			// decided here. Any partial `verified` accumulated so far in
			// THIS call is discarded on error, matching both of this
			// function's original call sites' behavior exactly (neither
			// ever returned a partial page's worth of rows alongside an
			// error).
			return nil, false, ferr
		}

		// Filter FIRST, before deciding whether to fetch another page or
		// stop -- every field `keep` checks is caller-supplied to the RPC
		// and NOT server-enforced (RPC list endpoints parse filters from
		// the request body with no injected/enforced identity constraint),
		// so an excluded row must never be allowed to count toward "we
		// have enough genuine rows" or "the source is exhausted." This
		// ordering is the single most safety-critical property of this
		// function -- do not move the exhaustion/overflow checks above it.
		for _, it := range items {
			if keep(it) {
				verified = append(verified, it)
			}
		}
		if uint64(len(items)) < limit+1 {
			pagesExhausted = true
			break // raw page returned fewer than requested -- source is genuinely exhausted, regardless of how many were excluded
		}
		if uint64(len(verified)) > limit {
			break // already have unambiguous proof of overflow -- no need to keep paging just to count higher
		}

		// Otherwise: the page was full AND we still don't have more than
		// `limit` worth of GENUINE rows -- keep paging, because the
		// shortfall might be entirely accounted for by excluded rows in
		// this page.
		//
		// This nil-TMCreate guard is DEFENSIVE, at an untrusted RPC/
		// deserialization boundary -- for the two current call sites it is
		// NOT reachable via their DB queries in normal operation (both
		// filter with WHERE tm_create < token, and under standard SQL
		// three-valued logic NULL < <any value> evaluates to NULL, not
		// TRUE, so a tm_create IS NULL row would be excluded by the WHERE
		// clause on every page). Kept as defense-in-depth against a
		// malformed or future-changed RPC response, not because current
		// production data can trigger it -- but a NEW caller of this
		// generic helper must not assume that same DB-level guarantee
		// without re-verifying it for its own data source.
		last := items[len(items)-1]
		tm := tmCreateOf(last)
		if tm == nil {
			break // cannot safely construct a continuation token; `capped` below will honestly reflect "possibly incomplete"
		}
		// utilhandler.ISO8601Layout, NOT time.RFC3339Nano: this function's
		// two current callers both talk to bin-transcribe-manager, whose
		// own default token (TimeGetCurTime()) uses this fixed-precision
		// layout, not RFC3339Nano's variable-precision format -- a token
		// built with the wrong layout will not error, it will silently
		// mis-paginate. A future caller against a DIFFERENT RPC must
		// confirm which layout THAT service's own list endpoint expects
		// rather than assuming this one.
		pageToken = tm.UTC().Format(utilhandler.ISO8601Layout)
	}
	// possiblyIncomplete captures every path that did not reach a proof
	// (overflow or exhaustion) within the page budget -- falling out of
	// the loop without either would otherwise silently report `capped` as
	// false, exactly the unmarked hole this function exists to prevent.
	possiblyIncomplete := !pagesExhausted && uint64(len(verified)) <= limit
	capped = uint64(len(verified)) > limit || possiblyIncomplete
	if uint64(len(verified)) > limit {
		verified = verified[:limit]
	}
	return verified, capped, nil
}

func (h *aicallHandler) toolHandleGetCallTranscript(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetCallTranscript",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_call_transcript.")

	res := newToolResult(tc.ID)

	// §3.0 entry guard. This is explicitly NOT a security control -- unlike
	// the Case-scoped sibling tools, this tool's access check (§3.2) derives
	// nothing from the Case; c.CustomerID (used for the tenant check) is
	// already present on the AIcall struct with no RPC needed. The guard
	// exists purely to keep this tool scoped to the Insight-on-Case product
	// surface, for consistency and blast-radius reduction, not because
	// anything downstream depends on it.
	if c.ReferenceType != aicall.ReferenceTypeContactCase {
		fillFailed(res, fmt.Errorf("get_call_transcript is only supported for contact_case reference type"))
		return res
	}

	// §3.1 argument: call_id is the ONLY tool in the Insight set besides
	// get_conversation_content (reference_id) to take an LLM-suppliable
	// identifier. Three-branch validation, mirroring
	// get_conversation_content's handling exactly.
	var args struct {
		CallID string `json:"call_id"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, errors.Wrap(err, "invalid arguments"))
		return res
	}
	if args.CallID == "" {
		fillFailed(res, fmt.Errorf("call_id is required, call get_contact_interactions first to discover candidate call ids"))
		return res
	}
	callID, err := uuid.FromString(args.CallID)
	if err != nil || callID == uuid.Nil {
		fillFailed(res, fmt.Errorf("invalid call_id"))
		return res
	}

	// §3.2 access check -- TENANT-ONLY (confirmed product decision, §0.2).
	// CallV1CallGet is unscoped (no customerID param), same shape as
	// ContactV1ContactGet -- this check is the SOLE tenant enforcement on
	// the call itself, load-bearing exactly like get_contact_profile's
	// mandatory contact check. Do NOT treat this as "defensive" or
	// "redundant". No further narrowing: this does NOT check whether call's
	// peer matches c's Case's peer/contact (confirmed decision, §0.2/§5).
	// Soft-deleted calls ARE returned by CallV1CallGet (no tm_delete filter
	// server-side) -- accepted, intentional: calls are historical records,
	// the Deleted:false filters/rechecks on Transcribe/Transcript below are
	// the actual data-lifecycle boundary this tool respects.
	call, err := h.reqHandler.CallV1CallGet(ctx, callID)
	if err != nil {
		if isNotFoundErr(err) {
			fillSuccess(res, "call_transcript", callID.String(), msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the call. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if call == nil || call.CustomerID != c.CustomerID || call.CustomerID == uuid.Nil {
		log.Warnf("Cross-customer or missing call on lookup. call_id: %s", callID)
		fillSuccess(res, "call_transcript", callID.String(), msgResourceNotFound)
		return res
	}

	// §3.3 Transcribe session list -- pagination-until-exact via the shared
	// paginateUntilExact helper. The exclusion rule below is stated
	// GENERICALLY ("exclude any row whose CustomerID != c.CustomerID"), not
	// as "exclude IDAIManager specifically", so it stays correct if a future
	// system-initiated transcriber appears (design §3.3's
	// summaryhandler/start.go IDAIManager note).
	verified, sessionCapped, err := paginateUntilExact(ctx, insightCallTranscribeFetchMaxPages, insightCallTranscribeSessionLimit,
		func(ctx context.Context, pageToken string) ([]tmtranscribe.Transcribe, error) {
			return h.reqHandler.TranscribeV1TranscribeList(ctx, pageToken, insightCallTranscribeSessionLimit+1, map[tmtranscribe.Field]any{
				tmtranscribe.FieldCustomerID:    c.CustomerID,
				tmtranscribe.FieldReferenceType: tmtranscribe.ReferenceTypeCall,
				tmtranscribe.FieldReferenceID:   callID,
				tmtranscribe.FieldDeleted:       false,
			})
		},
		func(t tmtranscribe.Transcribe) *time.Time { return t.TMCreate },
		func(t tmtranscribe.Transcribe) bool {
			if t.CustomerID != c.CustomerID {
				log.Warnf("Skipping cross-customer transcribe session. call_id: %s, transcribe_id: %s, transcribe_customer_id: %s", callID, t.ID, t.CustomerID)
				return false
			}
			if t.ReferenceType != tmtranscribe.ReferenceTypeCall || t.ReferenceID != callID {
				log.Warnf("Skipping transcribe session with mismatched reference. call_id: %s, transcribe_id: %s, transcribe_reference_type: %s, transcribe_reference_id: %s", callID, t.ID, t.ReferenceType, t.ReferenceID)
				return false
			}
			return t.TMDelete == nil
		},
	)
	if err != nil {
		// Honest failure, not masked -- this is a list call, not a
		// not-found-shaped Get, and there is no partial result to salvage
		// yet (unlike §3.4's per-session degrade-visibly policy below, a
		// session-list failure fails the whole tool).
		log.Errorf("Could not list transcribe sessions. call_id: %s, err: %v", callID, err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	// Not-found handling, guarded by sessionCapped: if verified is empty
	// AND sessionCapped == false (the loop actually proved exhaustion), this
	// is a legitimate absence. If verified is empty but sessionCapped ==
	// true (possiblyIncomplete fired before any genuine row was confirmed),
	// the tool does NOT know whether a transcript exists -- asserting "no
	// transcript found" would be affirmatively dishonest.
	if len(verified) == 0 {
		if !sessionCapped {
			fillSuccess(res, "call_transcript", callID.String(), "no transcript found for this call")
			return res
		}
		fillSuccess(res, "call_transcript", callID.String(), "could not confirm whether a transcript exists for this call")
		return res
	}

	// §3.4 Transcript segment fetch -- pagination-until-exact per session,
	// MANDATORY tenant filter + per-row recheck.
	var mergedLines []transcriptLine
	var sessionsUnavailable int
	var realLineCount int
	seq := 0

	for _, t := range verified {
		verifiedTranscripts, sessionFetchTruncated, err := paginateUntilExact(ctx, insightCallTranscribeFetchMaxPages, resourceListPageSize,
			func(ctx context.Context, pageToken string) ([]tmtranscript.Transcript, error) {
				return h.reqHandler.TranscribeV1TranscriptList(ctx, pageToken, resourceListPageSize+1, map[tmtranscript.Field]any{
					tmtranscript.FieldCustomerID:   c.CustomerID,
					tmtranscript.FieldTranscribeID: t.ID,
					tmtranscript.FieldDeleted:      false,
				})
			},
			func(tr tmtranscript.Transcript) *time.Time { return tr.TMCreate },
			func(tr tmtranscript.Transcript) bool {
				if tr.CustomerID != c.CustomerID {
					log.Warnf("Skipping cross-customer transcript row. call_id: %s, transcribe_id: %s, transcript_id: %s, transcript_customer_id: %s", callID, t.ID, tr.ID, tr.CustomerID)
					return false
				}
				if tr.TranscribeID != t.ID {
					// TranscribeID is exactly as unenforced as
					// CustomerID/Deleted -- recheck it too. Without this, a
					// same-tenant row from a DIFFERENT session could render
					// under THIS session's t.Language tag, mislabeling
					// which language was actually spoken on that line.
					log.Warnf("Skipping transcript row with mismatched transcribe_id. call_id: %s, transcribe_id: %s, transcript_id: %s, transcript_transcribe_id: %s", callID, t.ID, tr.ID, tr.TranscribeID)
					return false
				}
				return tr.TMDelete == nil
			},
		)
		if err != nil {
			// Partial-failure policy: one session's TranscriptList call
			// failing does NOT fail the whole tool -- skip this session,
			// but make the skip VISIBLE (sessionsUnavailable, surfaced in
			// §3.7's header) rather than a silent drop, matching
			// renderTranscribe's own "(transcripts unavailable)"
			// degrade-VISIBLY precedent. Contrast with §3.3's call site
			// above, which fails the whole tool on error -- this is the
			// caller-side policy difference paginateUntilExact
			// deliberately stays out of.
			log.Errorf("Could not list transcripts for session. call_id: %s, transcribe_id: %s, err: %v", callID, t.ID, err)
			sessionsUnavailable++
			continue // to the next session in the outer loop; this session contributes nothing
		}

		if sessionFetchTruncated && len(verifiedTranscripts) > 0 {
			// Now load-bearing, not merely defense-in-depth -- with
			// possiblyIncomplete folded into sessionFetchTruncated, the
			// flag CAN be true while verifiedTranscripts is empty (the
			// page-cap-exhaustion path), so this guard is what prevents an
			// index-out-of-range on the boundary line below. When this
			// branch is skipped for that reason, the session contributes
			// ZERO lines and ZERO gap marker -- an honest, but currently
			// unsurfaced, silent contribution of nothing; accepted as a
			// narrow, documented residual rather than adding new header
			// machinery for a path this pathological.
			//
			// The DB order is TMCreate DESC, so this slice keeps the
			// newest resourceListPageSize rows and index len-1 is the
			// earliest KEPT row -- the gap boundary. Copy its TMCreate
			// VERBATIM, including if nil: §3.5's nil-sorts-last rule
			// places the marker adjacent to that row wherever it ends up
			// sorting, with no separate nil-handling case needed for the
			// marker itself.
			boundary := verifiedTranscripts[len(verifiedTranscripts)-1].TMCreate
			mergedLines = append(mergedLines, transcriptLine{
				tmCreate:     boundary,
				rank:         0, // gap marker, per §3.5
				transcribeID: uuid.Nil,
				transcriptID: uuid.Nil,
				seq:          seq,
				bracket:      fmt.Sprintf("gap %s", t.Language),
				message:      "(earlier lines of this transcribe session were not fetched -- omitted for length)",
			})
			seq++
		}

		for _, tr := range verifiedTranscripts {
			mergedLines = append(mergedLines, transcriptLine{
				tmCreate:     tr.TMCreate,
				rank:         1, // real row
				transcribeID: tr.TranscribeID,
				transcriptID: tr.ID,
				seq:          seq,
				bracket:      fmt.Sprintf("%s %s", tr.Direction, t.Language),
				message:      tr.Message,
			})
			seq++
			realLineCount++
		}
	}

	// §3.5/§3.7: sort the accumulated transcriptLine entries per the strict
	// total-order key (tmCreate, rank, transcribeID, transcriptID, seq),
	// THEN render each to its final display string. mergedLines (the
	// []transcriptLine accumulator) and renderedLines (the []string it
	// becomes) are deliberately different variables -- the sort key fields
	// have no meaning once flattened to display text, so sorting must
	// happen before rendering.
	sort.SliceStable(mergedLines, func(i, j int) bool {
		a, b := mergedLines[i], mergedLines[j]
		// nil TMCreate sorts last (§3.5) -- treat nil as "infinitely new".
		// This is a NEW decision this design makes, not an inherited file
		// convention.
		if (a.tmCreate == nil) != (b.tmCreate == nil) {
			return b.tmCreate == nil // a sorts first iff b is the nil one
		}
		if a.tmCreate != nil && !a.tmCreate.Equal(*b.tmCreate) {
			return a.tmCreate.Before(*b.tmCreate)
		}
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if a.transcribeID != b.transcribeID {
			return a.transcribeID.String() < b.transcribeID.String()
		}
		if a.transcriptID != b.transcriptID {
			return a.transcriptID.String() < b.transcriptID.String()
		}
		return a.seq < b.seq
	})

	renderedLines := make([]string, 0, len(mergedLines))
	for _, l := range mergedLines {
		ts := "unknown"
		if l.tmCreate != nil {
			ts = l.tmCreate.UTC().Format(time.RFC3339)
		}
		renderedLines = append(renderedLines, fmt.Sprintf("[%s %s] %s", ts, l.bracket, l.message))
	}

	// transcript_lines: N is a PRE-RENDER, MERGED total (realLineCount),
	// not the actually-rendered line count -- render-budget truncation
	// below routinely drops it. transcribe_sessions deliberately omits a
	// numeric "of N" total: given the +1-sentinel/pagination fetch, the
	// true total is never exactly known, and a naive pre-filter total would
	// numerically leak whether a hidden IDAIManager session exists on this
	// call (§5).
	headerParts := []string{fmt.Sprintf("transcript_lines: %d", realLineCount)}
	if sessionCapped {
		headerParts = append(headerParts, fmt.Sprintf("transcribe_sessions: %d (capped, more may exist)", insightCallTranscribeSessionLimit))
	}
	if sessionsUnavailable > 0 {
		// Surface whole-session RPC failures (§3.4) visibly, rather than a
		// silent drop.
		headerParts = append(headerParts, fmt.Sprintf("sessions_unavailable: %d (transcript list fetch failed for this many sessions)", sessionsUnavailable))
	}
	header := strings.Join(headerParts, "\n")

	// pagedOut is hardcoded false (not sessionCapped): renderBodyLines
	// forces its head marker whenever pagedOut=true regardless of whether
	// the rendered lines actually overflow the rune budget -- exactly the
	// failure get_contact_profile already fixed in its own history. The
	// session cap is instead reported in the caller-owned header above, and
	// renderBodyLines' own fast-path/walk logic decides -- honestly --
	// whether the render budget was actually exceeded.
	//
	// Cross-file dependency note (§3.6): the gap-marker mechanism's honesty
	// guarantee depends on renderBodyLines retaining a CONTIGUOUS newest
	// suffix of its lines input during its backward-walk truncation path
	// (tool_resource.go's backward walk never skips/samples from the
	// middle), and on its final capSummaryRunes(sb.String()) call being
	// understood as a SEPARATE, coarser string-tail safety net layered on
	// top of that walk, not the same mechanism. If either changes, a gap
	// marker could stop honestly reflecting the true position of a fetch-
	// layer hole -- re-read design §3.6 before modifying renderBodyLines.
	body := renderBodyLines(header, renderedLines, false, "transcript lines")
	fillSuccess(res, "call_transcript", callID.String(), body)
	return res
}

// notifyAgentMaxMessageLen bounds a proactive note's length.
//
// The tool description asks for one or two sentences written for a busy human
// mid-call; this is the backstop for when the model ignores that. It is
// generous on purpose -- the point is to stop a runaway generation from landing
// a wall of text in the agent's panel mid-call, not to police phrasing.
//
// COUNTED IN CHARACTERS (runes), NOT BYTES (review round 1 finding LOW-1). The
// cap exists to bound what a human reads in a panel, and no downstream storage
// imposes a byte limit here -- so a len() byte count would silently cut a
// Korean or Japanese note to roughly a third of the intended length while the
// error message still said "characters".
const notifyAgentMaxMessageLen = 500

// parseNotifyAgentMessage extracts and validates the note from a notify_agent
// tool call's arguments.
func parseNotifyAgentMessage(arguments string) (string, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", errors.Wrap(err, "invalid arguments")
	}

	msg := strings.TrimSpace(args.Message)
	if msg == "" {
		return "", fmt.Errorf("message is required and must not be empty")
	}

	if msgLen := utf8.RuneCountInString(msg); msgLen > notifyAgentMaxMessageLen {
		return "", fmt.Errorf("message is too long: %d characters, maximum %d", msgLen, notifyAgentMaxMessageLen)
	}

	return msg, nil
}

// toolHandleNotifyAgent pushes a proactive note into the agent's Insight
// Assistant panel.
//
// It takes listenTurn as a parameter rather than deriving it: ToolHandle
// resolves it once, from Redis set membership, and shares that one answer with
// the Origin tagging decision. Two independent derivations could disagree.
//
// The row it writes is role=assistant with Origin=proactive, NOT
// role=notification. That distinction is load-bearing: getPipecatcallMessages
// skips RoleNotification when assembling LLM context, so a notification-role row
// would mean that when the agent replies "what did you mean by that?", the AI
// would have no memory of its own notification. It is a genuine assistant
// utterance and is stored as one.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.5, §5.6.
func (h *aicallHandler) toolHandleNotifyAgent(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall, listenTurn bool) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleNotifyAgent",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool notify_agent.")

	res := newToolResult(tc.ID)

	if !listenTurn {
		// This tool fired on the agent's own conversational turn, or the
		// membership check could not run at all (ToolHandle degrades a Redis
		// failure to listenTurn=false, which is provably correct during an
		// outage). Reject rather than let RunLLM's best-effort suppression
		// silently eat the agent's real question: with run_llm=false in effect,
		// a notify_agent call during a Q&A turn means the agent gets an
		// unrelated notification INSTEAD of the answer they asked for.
		fillFailed(res, fmt.Errorf("notify_agent is only usable while proactively monitoring a call; you were asked a question - answer it directly instead"))
		return res
	}

	msg, err := parseNotifyAgentMessage(tc.Function.Arguments)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	tmp, errCreate := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID,
		message.DirectionIncoming, message.RoleAssistant, msg, nil, "",
		messagehandler.WithActiveAIID(h.resolveActiveAIIDFromAIcall(ctx, c)),
		messagehandler.WithOrigin(message.OriginProactive))
	if errCreate != nil {
		log.Errorf("Could not create the proactive message. err: %v", errCreate)
		fillFailed(res, fmt.Errorf("could not deliver the notification"))
		return res
	}
	log.WithField("message", tmp).Debugf("Created the proactive notification message. message_id: %s", tmp.ID)

	promListenNotifyTotal.WithLabelValues(listenKindOf(c).label()).Inc()

	fillSuccess(res, "message", tmp.ID.String(), "Notification delivered to the agent.")
	return res
}
