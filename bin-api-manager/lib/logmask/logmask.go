// Package logmask masks request credentials (accesskey, token) that gin writes
// to its access log and panic recovery dump.
//
// api-manager accepts the accesskey and the JWT through the URL query and
// through cookies. gin's access logger prints the raw query, and its recovery
// dump prints the request line and the Cookie header, so without masking the
// credentials reach stdout in plaintext.
package logmask

import (
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const masked = "***"

// secretNames are the credential names read from the query and cookies by the
// authenticate middleware. Names are matched exactly (case-sensitive), the same
// way gin's c.Query and c.Cookie match them.
var secretNames = map[string]struct{}{
	"accesskey": {},
	"token":     {},
}

var (
	// reQueryPair matches one "?key=value" or "&key=value" pair. The key may be
	// percent-encoded (Go decodes query keys, so "%61ccesskey" authenticates).
	// The value runs to the next "&" or whitespace. Backslashes are plain
	// characters: in the raw recovery dump "a\&accesskey=V" is two keys for Go.
	reQueryPair = regexp.MustCompile(`([?&])([^=&\s]*)=([^&\s]*)`)

	// reCookieLine matches a whole Cookie header line of the recovery dump.
	reCookieLine = regexp.MustCompile(`(?m)^Cookie:[^\r\n]*`)
)

// Mask returns s with the value of every accesskey and token query parameter
// and cookie replaced by "***". Key names and all other content are kept.
func Mask(s string) string {
	if !strings.ContainsAny(s, "?&") && !strings.Contains(s, "Cookie:") {
		return s
	}

	s = reQueryPair.ReplaceAllStringFunc(s, maskQueryPair)
	s = reCookieLine.ReplaceAllStringFunc(s, maskCookieLine)
	return s
}

func maskQueryPair(m string) string {
	sub := reQueryPair.FindStringSubmatch(m)
	sep, rawKey, val := sub[1], sub[2], sub[3]

	key, err := url.QueryUnescape(rawKey)
	if err != nil {
		key = rawKey
	}
	if _, ok := secretNames[key]; !ok {
		return m
	}

	// In the Go-quoted access log line the value is followed by the closing
	// quote of the path. Keep it so the log format is unchanged.
	tail := ""
	if strings.HasSuffix(val, `"`) {
		tail = `"`
	}
	return sep + rawKey + "=" + masked + tail
}

func maskCookieLine(line string) string {
	const header = "Cookie:"
	parts := strings.Split(line[len(header):], ";")
	for i, p := range parts {
		eq := strings.Index(p, "=")
		if eq < 0 {
			continue
		}
		if _, ok := secretNames[strings.TrimSpace(p[:eq])]; ok {
			parts[i] = p[:eq+1] + masked
		}
	}
	return header + strings.Join(parts, ";")
}

type writer struct {
	w io.Writer
}

// NewWriter returns an io.Writer that masks credentials in everything written
// to w. It is stateless, so it is safe for concurrent use as long as w is.
// Write always reports len(p) on success so callers never see a short write.
// Each log record must arrive in a single Write call (gin does this for both
// the access logger and the recovery logger).
func NewWriter(w io.Writer) io.Writer {
	return &writer{w: w}
}

func (m *writer) Write(p []byte) (int, error) {
	out := Mask(string(p))
	if _, err := m.w.Write([]byte(out)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Middlewares returns the gin access logger and the panic recovery middleware,
// both with credential masking. out and errOut default to gin.DefaultWriter and
// gin.DefaultErrorWriter (the sinks gin.Default() uses) when nil.
func Middlewares(skipPaths []string, out, errOut io.Writer) []gin.HandlerFunc {
	return middlewares(skipPaths, out, errOut, true)
}

// middlewares builds the same wiring with masking switchable, so tests can
// prove the assertions fail without the fix (negative control).
func middlewares(skipPaths []string, out, errOut io.Writer, mask bool) []gin.HandlerFunc {
	if out == nil {
		out = gin.DefaultWriter
	}
	if errOut == nil {
		errOut = gin.DefaultErrorWriter
	}
	if mask {
		out = NewWriter(out)
		errOut = NewWriter(errOut)
	}

	return []gin.HandlerFunc{
		gin.LoggerWithConfig(gin.LoggerConfig{Output: out, SkipPaths: skipPaths}),
		gin.RecoveryWithWriter(errOut),
	}
}
