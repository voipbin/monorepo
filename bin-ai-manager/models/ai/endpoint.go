package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CustomEndpointPrefix is the engine model prefix of a model served by a customer specified
// OpenAI compatible endpoint: customendpoint.<model id>.
const CustomEndpointPrefix = "customendpoint."

const (
	maxEngineBaseURLLen = 512
	maxEngineBaseKeyLen = 512
	maxHostLen          = 253
	maxLabelLen         = 63

	// engineLookupTimeout bounds the validation lookup (inside the 3 second RPC budget).
	engineLookupTimeout = 1 * time.Second

	// maxEngineLookups bounds the validation lookups that run at once in one process.
	maxEngineLookups = 4
)

var (
	// ErrInvalidEngineBaseURL is a violation of the pure URL rules.
	ErrInvalidEngineBaseURL = errors.New("invalid engine base url")

	// ErrEngineBaseURLRejected covers a resolve failure, no answer, a blocked answer and a blocked
	// literal address. One error for all causes avoids an oracle for internal names.
	ErrEngineBaseURLRejected = errors.New("engine base url rejected")

	ErrEngineBaseURLRequired   = errors.New("engine base url required")
	ErrEngineBaseURLNotAllowed = errors.New("engine base url not allowed")
	ErrInvalidEngineBaseKey    = errors.New("invalid engine base key")
	ErrEngineBaseKeyNotAllowed = errors.New("engine base key not allowed")
)

// RedactedString is a secret string. Its text forms print <redacted>. Its JSON form is the plain
// string because the value has to cross the internal RPC, so the protection covers fmt verbs
// and text formatting only.
type RedactedString string

const redactedText = "<redacted>"

// String implements fmt.Stringer.
func (RedactedString) String() string { return redactedText }

// GoString implements fmt.GoStringer.
func (RedactedString) GoString() string { return redactedText }

// Format implements fmt.Formatter, so every verb prints the marker.
func (RedactedString) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, redactedText)
}

// IsCustomEndpointModel reports whether the model has the customendpoint. prefix (case sensitive).
func IsCustomEndpointModel(m EngineModel) bool {
	return strings.HasPrefix(string(m), CustomEndpointPrefix)
}

var (
	engineResolverMu sync.RWMutex

	// engineResolver resolves an absolute host name. Replaced in tests only (SetEngineResolverForTest).
	engineResolver = func(ctx context.Context, host string) ([]netip.Addr, error) {
		return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	}

	engineLookupSlots = make(chan struct{}, maxEngineLookups)
)

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return false
		}
	}

	return true
}

func validHostName(host string) bool {
	if len(host) == 0 || len(host) > maxHostLen || !strings.Contains(host, ".") {
		return false
	}

	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > maxLabelLen {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			isAlnum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
			if !isAlnum && c != '-' {
				return false
			}
		}
	}

	return true
}

// parseEngineBaseURL applies the pure rules. It returns the host and, for an IP literal, the address.
func parseEngineBaseURL(raw string) (host string, literal netip.Addr, err error) {
	if len(raw) == 0 || len(raw) > maxEngineBaseURLLen || !isPrintableASCII(raw) {
		return "", netip.Addr{}, ErrInvalidEngineBaseURL
	}
	if strings.ContainsAny(raw, "?#") || !strings.HasPrefix(raw, "https://") {
		return "", netip.Addr{}, ErrInvalidEngineBaseURL
	}

	u, errParse := url.Parse(raw)
	if errParse != nil {
		return "", netip.Addr{}, ErrInvalidEngineBaseURL
	}
	if u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", netip.Addr{}, ErrInvalidEngineBaseURL
	}

	if strings.HasSuffix(u.Host, ":") {
		return "", netip.Addr{}, ErrInvalidEngineBaseURL
	}
	if p := u.Port(); p != "" {
		n, errPort := strconv.Atoi(p)
		if errPort != nil || n < 1 || n > 65535 {
			return "", netip.Addr{}, ErrInvalidEngineBaseURL
		}
		for i := 0; i < len(p); i++ {
			if p[i] < '0' || p[i] > '9' {
				return "", netip.Addr{}, ErrInvalidEngineBaseURL
			}
		}
	}

	h := u.Hostname()
	if h == "" || strings.Contains(h, "%") {
		return "", netip.Addr{}, ErrInvalidEngineBaseURL
	}

	if a, errAddr := netip.ParseAddr(h); errAddr == nil {
		return h, a, nil
	}
	if strings.Contains(u.Host, "[") || strings.Contains(u.Host, "]") || !validHostName(h) {
		return "", netip.Addr{}, ErrInvalidEngineBaseURL
	}

	return h, netip.Addr{}, nil
}

// ValidateEngineBaseURL checks the rules of a customer specified base URL. A host name is resolved
// (absolute name, 1 second) and every answer must be outside the block table. The errors never
// carry the URL.
//
// At most maxEngineLookups validation lookups run at once in the process, taken without blocking,
// so that a slow name server of one customer cannot hold the RPC workers of ai-manager. Callers
// outside ai-manager use ValidateEngineBaseURLForSession.
func ValidateEngineBaseURL(ctx context.Context, raw string) error {
	return validateEngineBaseURL(ctx, raw, true)
}

// ValidateEngineBaseURLForSession applies the same rules as ValidateEngineBaseURL without the
// in-process limit of concurrent lookups. The limit protects the RPC workers of ai-manager; a
// session start (pipecat-manager) bounds each lookup by its own deadline, and a team with several
// custom endpoint members must not be rejected because its members are checked at the same time.
func ValidateEngineBaseURLForSession(ctx context.Context, raw string) error {
	return validateEngineBaseURL(ctx, raw, false)
}

func validateEngineBaseURL(ctx context.Context, raw string, limited bool) error {
	host, literal, err := parseEngineBaseURL(raw)
	if err != nil {
		return err
	}

	if literal.IsValid() {
		if isBlockedAddr(literal) {
			return ErrEngineBaseURLRejected
		}
		return nil
	}

	if limited {
		select {
		case engineLookupSlots <- struct{}{}:
			defer func() { <-engineLookupSlots }()
		default:
			return ErrEngineBaseURLRejected
		}
	}

	engineResolverMu.RLock()
	resolve := engineResolver
	engineResolverMu.RUnlock()

	cctx, cancel := context.WithTimeout(ctx, engineLookupTimeout)
	defer cancel()

	addrs, errLookup := resolve(cctx, host+".")
	if errLookup != nil || len(addrs) == 0 {
		return ErrEngineBaseURLRejected
	}
	for _, a := range addrs {
		if isBlockedAddr(a) {
			return ErrEngineBaseURLRejected
		}
	}

	return nil
}

// ValidateEngineEndpoint ties the model to the URL: a customendpoint. model needs a URL, any other
// model must not carry one.
func ValidateEngineEndpoint(model EngineModel, baseURL string) error {
	if IsCustomEndpointModel(model) {
		if baseURL == "" {
			return ErrEngineBaseURLRequired
		}
		return nil
	}

	if baseURL != "" {
		return ErrEngineBaseURLNotAllowed
	}

	return nil
}

// ValidateEngineBaseKey checks the write-only key of a request. nil means omitted and "" means
// delete, both fine for every model; a value needs a customendpoint. model and 1 to 512 printable
// ASCII bytes (which also rules out header injection).
func ValidateEngineBaseKey(model EngineModel, key *string) error {
	if key == nil || *key == "" {
		return nil
	}

	if !IsCustomEndpointModel(model) {
		return ErrEngineBaseKeyNotAllowed
	}
	if len(*key) > maxEngineBaseKeyLen || !isPrintableASCII(*key) {
		return ErrInvalidEngineBaseKey
	}

	return nil
}
