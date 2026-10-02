package callhandler

//go:generate mockgen -package callhandler -destination ./mock_recoveryhandler.go -source recoveryhandler.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/jart/gosip/sip"
	"github.com/sirupsen/logrus"
)

type recoveryDetail struct {
	RequestURI   string
	Routes       string
	RecordRoutes string
	CallID       string

	FromDisplay string
	FromURI     string
	FromTag     string

	ToDisplay string
	ToURI     string
	ToTag     string

	// CSeq is the local CSeq for the recovery INVITE. 0 means "not known, let the dialog choose".
	CSeq int
}

type asteriskRole string

const (
	asteriskRoleUnknown asteriskRole = ""
	asteriskRoleUAC     asteriskRole = "uac" // User Agent Client: Asterisk sent the initial INVITE
	asteriskRoleUAS     asteriskRole = "uas" // User Agent Server: Asterisk received the initial INVITE
)

// recoveryCSeqMargin is added to the highest CSeq Asterisk is known to have used in the dialog.
// RFC 3261 12.2.1.1 requires the local CSeq to increase, not to be contiguous; the margin covers
// Asterisk requests missing from the capture (HEP loss, ingest delay, a refresh just before the crash),
// which would otherwise make the remote reject the recovery INVITE with 500 (RFC 3261 12.2.2).
const recoveryCSeqMargin = 100

type RecoveryHandler interface {
	GetRecoveryDetail(ctx context.Context, callID string, role asteriskRole) (*recoveryDetail, error)
}

type recoveryHandler struct {
	requestHandler requesthandler.RequestHandler

	httpClient      *http.Client
	homerAPIAddress string
	homerAuthToken  string
	loadBalancerIPs []string
}

var (
	defaultHomerSearchTimeRange = -24 * time.Hour // from 24 hours ago
)

// NewRecoveryHandler creates a new RecoveryHandler instance
func NewRecoveryHandler(
	requestHandler requesthandler.RequestHandler,

	homerAPIAddress string,
	homerAuthToken string,
	loadBalancerIPs []string,
) RecoveryHandler {
	return &recoveryHandler{
		requestHandler: requestHandler,

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},

		homerAPIAddress: homerAPIAddress,
		homerAuthToken:  homerAuthToken,

		loadBalancerIPs: loadBalancerIPs,
	}
}

// GetRecoveryDetail returns Asterisk's own side of the SIP dialog with the given Call-ID,
// reconstructed from the Homer capture. role is Asterisk's role in the dialog, taken from
// call-manager's own call record (outgoing call: UAC, incoming call: UAS).
func (h *recoveryHandler) GetRecoveryDetail(ctx context.Context, callID string, role asteriskRole) (*recoveryDetail, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "GetRecoveryDetail",
		"call_id": callID,
		"role":    role,
	})

	if h.homerAPIAddress == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("missing Homer API address or auth token")
	}

	sipMessages, err := h.getSIPMessages(ctx, callID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get SIP messages for call ID. call_id: %s", callID)
	}

	res, err := getRecoveryDetail(sipMessages, role)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get recovery details for call ID. call_id: %s", callID)
	}
	log.WithField("res", res).Debug("Found recovery details successfully")

	return res, nil
}

// getRecoveryDetail reconstructs Asterisk's side of the dialog from the captured messages.
// The result does not depend on the order of the messages; ambiguous captures fail.
func getRecoveryDetail(messages []*sip.Msg, role asteriskRole) (*recoveryDetail, error) {
	if len(messages) == 0 {
		return nil, errors.New("no SIP messages provided")
	}
	if role != asteriskRoleUAC && role != asteriskRoleUAS {
		return nil, fmt.Errorf("unsupported asterisk role. role: %s", role)
	}

	invites, responses, err := findDialogCreation(messages)
	if err != nil {
		return nil, err
	}

	invite := invites[0]
	response := responses[0]

	// the fields each role takes from the initial INVITE and from the dialog-creating 2xx must agree across copies.
	var inviteKey, responseKey func(m *sip.Msg) string
	switch role {
	case asteriskRoleUAC:
		inviteKey = func(m *sip.Msg) string { return addrString(m.From) + "|" + addrString(m.To) }
		responseKey = func(m *sip.Msg) string {
			return getTag(m.To) + "|" + addrString(m.Contact) + "|" + addrString(m.RecordRoute)
		}
	case asteriskRoleUAS:
		inviteKey = func(m *sip.Msg) string {
			return addrString(m.From) + "|" + addrString(m.To) + "|" + addrString(m.Contact) + "|" + addrString(m.RecordRoute)
		}
		responseKey = func(m *sip.Msg) string { return getTag(m.To) }
	}
	if !sameKey(invites, inviteKey) {
		return nil, errors.New("copies of the initial INVITE differ")
	}
	if !sameKey(responses, responseKey) {
		return nil, errors.New("copies of the dialog-creating response differ")
	}

	res := &recoveryDetail{
		CallID: invite.CallID,
	}

	switch role {
	case asteriskRoleUAC:
		res.FromDisplay, res.FromURI, res.FromTag = addrDisplay(invite.From), addrURI(invite.From), getTag(invite.From)
		res.ToDisplay, res.ToURI, res.ToTag = addrDisplay(invite.To), addrURI(invite.To), getTag(response.To)

		res.RequestURI = addrURI(response.Contact)
		if res.RequestURI == "" {
			res.RequestURI = addrURI(invite.To)
		}
		if response.RecordRoute != nil {
			res.Routes = response.RecordRoute.Reversed().String()
			res.RecordRoutes = response.RecordRoute.String()
		}

	case asteriskRoleUAS:
		res.FromDisplay, res.FromURI, res.FromTag = addrDisplay(invite.To), addrURI(invite.To), getTag(response.To)
		res.ToDisplay, res.ToURI, res.ToTag = addrDisplay(invite.From), addrURI(invite.From), getTag(invite.From)

		res.RequestURI = addrURI(invite.Contact)
		if res.RequestURI == "" {
			res.RequestURI = addrURI(invite.From)
		}
		if invite.RecordRoute != nil {
			res.Routes = invite.RecordRoute.String()
			res.RecordRoutes = invite.RecordRoute.String()
		}
	}

	// local CSeq: the transactions Asterisk started carry its tag in From (its requests and the responses to them).
	maxCSeq := 0
	for _, m := range messages {
		if getTag(m.From) == res.FromTag && m.CSeq > maxCSeq {
			maxCSeq = m.CSeq
		}
	}
	if maxCSeq > 0 {
		res.CSeq = maxCSeq + recoveryCSeqMargin
	}

	if res.CallID == "" || res.FromTag == "" || res.ToTag == "" || res.FromURI == "" || res.ToURI == "" || res.RequestURI == "" {
		return nil, fmt.Errorf("incomplete dialog. call_id: %s, from_tag: %s, to_tag: %s, from_uri: %s, to_uri: %s, request_uri: %s",
			res.CallID, res.FromTag, res.ToTag, res.FromURI, res.ToURI, res.RequestURI)
	}

	return res, nil
}

// findDialogCreation returns the copies of the INVITE that created the dialog and of the 2xx that answered it.
// The dialog-creating 2xx is the INVITE 2xx with a To tag, answering a To-tagless INVITE in the capture
// (same From tag and CSeq), with the lowest CSeq. This skips INVITEs challenged with 401/407 (no 2xx)
// and 2xx of re-INVITEs (the re-INVITE carries a To tag). Several copies of the returned messages
// (retransmissions, capture hops, forked 2xx) are returned together for the caller's consistency check.
func findDialogCreation(messages []*sip.Msg) ([]*sip.Msg, []*sip.Msg, error) {
	type key struct {
		fromTag string
		cseq    int
	}

	initialInvites := map[key][]*sip.Msg{}
	for _, m := range messages {
		if m.IsResponse() || m.Method != sip.MethodInvite || getTag(m.To) != "" {
			continue
		}
		k := key{fromTag: getTag(m.From), cseq: m.CSeq}
		initialInvites[k] = append(initialInvites[k], m)
	}
	if len(initialInvites) == 0 {
		return nil, nil, errors.New("no initial INVITE found")
	}

	var found []key
	responses := map[key][]*sip.Msg{}
	for _, m := range messages {
		if !m.IsResponse() || m.Status < 200 || m.Status >= 300 || m.CSeqMethod != sip.MethodInvite || getTag(m.To) == "" {
			continue
		}
		k := key{fromTag: getTag(m.From), cseq: m.CSeq}
		if _, ok := initialInvites[k]; !ok {
			continue
		}
		if _, ok := responses[k]; !ok {
			found = append(found, k)
		}
		responses[k] = append(responses[k], m)
	}
	if len(found) == 0 {
		return nil, nil, errors.New("no 2xx response for an initial INVITE found")
	}

	lowest := found[0]
	for _, k := range found[1:] {
		if k.cseq < lowest.cseq {
			lowest = k
		}
	}
	for _, k := range found {
		if k.cseq == lowest.cseq && k.fromTag != lowest.fromTag {
			return nil, nil, errors.New("several dialogs created with the same CSeq")
		}
	}

	// forked 2xx (different To tags) are rejected by the copy consistency check of the caller.
	return initialInvites[lowest], responses[lowest], nil
}

// sameKey returns true if key gives the same value for every message.
func sameKey(messages []*sip.Msg, key func(m *sip.Msg) string) bool {
	for _, m := range messages[1:] {
		if key(m) != key(messages[0]) {
			return false
		}
	}
	return true
}

// getTag returns the tag parameter of the given address, or "" if the address or the tag is missing.
func getTag(addr *sip.Addr) string {
	if addr == nil {
		return ""
	}
	tag := addr.Param.Get("tag")
	if tag == nil {
		return ""
	}
	return tag.Value
}

// addrURI returns the URI of the given address, or "" if the address is missing.
func addrURI(addr *sip.Addr) string {
	if addr == nil || addr.Uri == nil {
		return ""
	}
	return addr.Uri.String()
}

// addrDisplay returns the display name of the given address, or "" if the address is missing.
func addrDisplay(addr *sip.Addr) string {
	if addr == nil {
		return ""
	}
	return addr.Display
}

// addrString returns the given address list as a string, or "" if it is missing.
func addrString(addr *sip.Addr) string {
	if addr == nil {
		return ""
	}
	return addr.String()
}
