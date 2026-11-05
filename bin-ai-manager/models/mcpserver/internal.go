package mcpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"sort"
	"strings"

	"github.com/gofrs/uuid"
)

// ReservedHost is the host of the url convention that routes an MCP server to
// a built-in carrier adapter instead of the network. ".invalid" is reserved
// by RFC 2606 for names that never resolve, so it cannot collide with a real
// host, and the guarded dialer refuses every ".invalid" host as a second line.
const ReservedHost = "carrier.invalid"

// CredentialFormat is how a carrier's credential is stored in the single
// secret of an McpServer record.
type CredentialFormat string

const (
	// CredentialClientIDSecret is "CLIENT_ID:CLIENT_SECRET".
	CredentialClientIDSecret CredentialFormat = "client_id_secret"
	// CredentialAPIKey is the whole secret as one key.
	CredentialAPIKey CredentialFormat = "api_key"
)

// IntegrationUnavailableText is what the model is told when a reserved-host
// server has no adapter behind it (the carrier was removed or never added).
const IntegrationUnavailableText = "This carrier integration is not available. Tell the agent that an account admin must remove this MCP server. Do not retry."

// CarrierEntry is one line of the carrier table.
type CarrierEntry struct {
	// Carrier is the carrier's short lower-case name, e.g. "dhl".
	Carrier string
	// Credential is the format of the secret this carrier needs.
	Credential CredentialFormat
}

// CarrierTable maps the path of a reserved url (e.g. "/dhl") to its carrier.
// A carrier is opened by adding its line here together with its adapter, and
// closed by removing both.
type CarrierTable map[string]CarrierEntry

// DefaultCarrierTable returns the production table, as a new map on each
// call. It is empty: no carrier is open.
func DefaultCarrierTable() CarrierTable {
	return CarrierTable{}
}

// NormalizeHost reduces a host to the form used for every comparison against
// ReservedHost: ideographic, fullwidth and halfwidth full stops become ".",
// letters are lower-cased and trailing dots are removed.
var stopReplacer = strings.NewReplacer("\u3002", ".", "\uff0e", ".", "\uff61", ".")

func NormalizeHost(host string) string {
	host = stopReplacer.Replace(host)
	return strings.TrimRight(strings.ToLower(host), ".")
}

// IsReservedHost reports whether rawURL points at ReservedHost. Case folding
// can map a non-ASCII letter to an ASCII one; that is kept as reserved on
// purpose, because the wrong answer here sends a url to the network path.
// InternalCarrier is the strict check. It looks at the host only and does not depend on the table, so a url that is not (or no
// longer) a known carrier url is still kept off the network.
func IsReservedHost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return NormalizeHost(u.Hostname()) == ReservedHost
}

// InternalCarrier returns the table entry rawURL names. It accepts only the
// canonical form: https, the reserved host (any case, trailing dots allowed),
// no user info, port (an empty one included), query, fragment or encoded path,
// and a path that equals a table key exactly.
func (t CarrierTable) InternalCarrier(rawURL string) (CarrierEntry, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return CarrierEntry{}, false
	}
	switch {
	case strings.Contains(rawURL, "#"), !isASCII(stopReplacer.Replace(u.Hostname())),
		u.Scheme != "https",
		NormalizeHost(u.Hostname()) != ReservedHost,
		strings.Contains(u.Host, ":"),
		u.User != nil,
		u.RawQuery != "", u.ForceQuery,
		u.Fragment != "", u.RawFragment != "", u.Opaque != "",
		u.RawPath != "":
		return CarrierEntry{}, false
	}

	entry, ok := t[u.Path]
	return entry, ok
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// AllowedURLs returns the canonical urls of the table in sorted order.
func (t CarrierTable) AllowedURLs() []string {
	res := make([]string, 0, len(t))
	for path := range t {
		res = append(res, "https://"+ReservedHost+path)
	}
	sort.Strings(res)
	return res
}

// ErrCredentialMalformed is returned for a secret that does not have the
// format its carrier needs. It never carries the input.
var ErrCredentialMalformed = errors.New("credential does not have the required format")

// maxCredentialLen bounds the stored secret of a carrier server.
const maxCredentialLen = 512

// Credential is a parsed carrier credential. It prints as "[redacted]" in
// every formatting, encoding and logging path.
type Credential struct {
	ID     string
	Secret string
}

const redacted = "[redacted]"

// String implements fmt.Stringer.
func (Credential) String() string { return redacted }

// GoString implements fmt.GoStringer.
func (Credential) GoString() string { return redacted }

// MarshalJSON implements json.Marshaler.
func (Credential) MarshalJSON() ([]byte, error) { return json.Marshal(redacted) }

// MarshalText implements encoding.TextMarshaler.
func (Credential) MarshalText() ([]byte, error) { return []byte(redacted), nil }

// LogValue implements slog.LogValuer.
func (Credential) LogValue() slog.Value { return slog.StringValue(redacted) }

// ParseCredential parses a carrier secret. Outer white space is trimmed; the
// rest must be 1 to 512 visible ASCII characters. For CredentialClientIDSecret
// the value is split at the first ":" and neither part may be empty.
func ParseCredential(format CredentialFormat, raw string) (Credential, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxCredentialLen {
		return Credential{}, ErrCredentialMalformed
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] < 0x21 || raw[i] > 0x7e {
			return Credential{}, ErrCredentialMalformed
		}
	}

	switch format {
	case CredentialClientIDSecret:
		id, secret, found := strings.Cut(raw, ":")
		if !found || id == "" || secret == "" {
			return Credential{}, ErrCredentialMalformed
		}
		return Credential{ID: id, Secret: secret}, nil

	case CredentialAPIKey:
		return Credential{Secret: raw}, nil

	default:
		return Credential{}, ErrCredentialMalformed
	}
}

// CarrierToolDef is the definition of a tool a carrier adapter offers. It has
// the fields of the MCP tools/list entry the model is shown.
type CarrierToolDef struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

// CarrierCall is one tool call routed to a carrier adapter.
type CarrierCall struct {
	// CustomerID is the owner of the server record, read from the database
	// row and not from the model's arguments.
	CustomerID uuid.UUID
	ServerID   uuid.UUID

	Carrier    string
	Credential CredentialFormat

	ToolName      string
	ArgumentsJSON string

	// Secret decrypts the server's secret on demand, so the adapter reads it
	// only after it has checked what does not need it. present is false when
	// the record holds no usable secret.
	Secret func() (secret string, present bool, err error)
}
