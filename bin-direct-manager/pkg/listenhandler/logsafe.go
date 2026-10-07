package listenhandler

import (
	"strings"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-direct-manager/models/direct"
)

const byHashURIMarker = "/v1/directs/by-hash/"

// maskURI masks the hash segment of a by-hash request URI so a direct hash
// is not written to the logs. Any query string after the hash is kept.
func maskURI(uri string) string {
	idx := strings.Index(uri, byHashURIMarker)
	if idx < 0 {
		return uri
	}

	start := idx + len(byHashURIMarker)
	rest := uri[start:]
	hash := rest
	tail := ""
	if end := strings.IndexAny(rest, "?/"); end >= 0 {
		hash = rest[:end]
		tail = rest[end:]
	}

	return uri[:start] + direct.MaskHash(hash) + tail
}

// requestLogFields returns the log-safe fields of a request. The body is not
// logged and the URI is masked.
func requestLogFields(m *sock.Request) logrus.Fields {
	if m == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"method":    m.Method,
		"uri":       maskURI(m.URI),
		"data_type": m.DataType,
	}
}

// responseLogFields returns the log-safe fields of a response. The body is
// not logged because it carries the Direct JSON including the hash.
func responseLogFields(r *sock.Response) logrus.Fields {
	if r == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"status_code": r.StatusCode,
		"data_type":   r.DataType,
	}
}
