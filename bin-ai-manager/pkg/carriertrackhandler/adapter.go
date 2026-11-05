package carriertrackhandler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"monorepo/bin-ai-manager/models/mcpserver"
)

// Adapter is one carrier's tracking lookup. A carrier is opened by adding its
// line to the carrier table of models/mcpserver together with its adapter in
// the handler's registry, and closed by removing both.
type Adapter interface {
	// Carrier is the lower-case name used in the table, e.g. "dhl".
	Carrier() string

	// Tool is the single read-only tool the model is offered.
	Tool() mcpserver.CarrierToolDef

	// NormalizeNumber cleans a tracking number and reports whether it has the
	// shape of this carrier's numbers.
	NormalizeNumber(raw string) (string, bool)

	// MaxBody is the most response bytes the adapter reads; 0 selects
	// mcpserverhandler.McpHTTPResponseSizeCapBytes.
	MaxBody() int64

	// Track looks the number up. It reads at most deps.MaxBody bytes of any
	// response, sends the credential only in a header or a body, and returns
	// no text of the carrier: only the Result or a CarrierError.
	Track(ctx context.Context, deps Deps, cred mcpserver.Credential, number string) (*Result, *CarrierError)
}

// Deps is what the handler gives an adapter for one call.
type Deps struct {
	// Carrier is the carrier name, used for metrics.
	Carrier string

	// Client is made for this call from the guarded client factory and closed
	// at the end of the call.
	Client *http.Client

	Now func() time.Time

	// Tokens and TokenKey cache the access token of this call's key. An empty
	// TokenKey disables caching.
	Tokens   *TokenCache
	TokenKey string

	// MaxBody is the response read limit.
	MaxBody int64
}

// CleanNumber trims a tracking number, drops spaces and hyphens and upper-cases it.
func CleanNumber(raw string) string {
	return strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(raw)))
}

// LooseNumber reports whether n is ASCII letters and digits with a length from
// minLen to maxLen. Carriers other than UPS have no published check rule, so this is
// all that can be said without a lookup.
func LooseNumber(n string, minLen, maxLen int) bool {
	if len(n) < minLen || len(n) > maxLen {
		return false
	}
	for i := 0; i < len(n); i++ {
		c := n[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

// errBodyTooLarge is returned by ReadBody for a body over the limit.
var errBodyTooLarge = errors.New("response body is over the limit")

// ReadBody reads r up to limit bytes. A body over the limit is an error, not a
// silently cut document that would fail to parse in a confusing way.
func ReadBody(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errBodyTooLarge
	}
	return b, nil
}
