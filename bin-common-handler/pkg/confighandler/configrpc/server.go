// Package configrpc serves the configs of one service over its RPC queue: the keys with their
// sources, the write of a secret, the removal of a stored value and the change history. It is the
// one place that holds the rules for these requests (the log-safe copy of a request, the check of
// a secret, the mapping of the errors), so that every service that adopts confighandler gets the
// same behavior with one mount.
//
// A response never holds a secret, a length, a hash or an environment value, and an error response
// never holds the text of the error.
package configrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/confighandler"
)

// handleTimeout bounds one request, so that a stalled store can not hold the consumer.
const handleTimeout = 15 * time.Second

// reasons of the typed errors
const (
	reasonKeyNotFound = "CONFIG_KEY_NOT_FOUND"
	reasonInvalid     = "CONFIG_INVALID"
	reasonMismatch    = "REVISION_MISMATCH"
	reasonNoMasterKey = "ENCRYPTION_KEY_NOT_CONFIGURED"
	reasonWriteVerify = "CONFIG_WRITE_VERIFY_FAILED"
	reasonInternal    = "INTERNAL"
)

const keyPattern = `[a-z0-9][a-z0-9._]{0,127}`

var (
	regService   = regexp.MustCompile(`^/v1/configs$`)
	regSecret    = regexp.MustCompile(`^/v1/configs/(` + keyPattern + `)/secret$`)
	regKey       = regexp.MustCompile(`^/v1/configs/(` + keyPattern + `)$`)
	regHistories = regexp.MustCompile(`^/v1/configs/(` + keyPattern + `)/histories$`)
)

// redactedData is the body that replaces the body of a request that may hold a secret.
var redactedData = json.RawMessage(`"[REDACTED]"`)

// Server serves the config requests of one service on its confighandler.Handler.
type Server struct {
	h      confighandler.Handler
	domain outline.ServiceName
}

// NewServer creates the server. domain is the name of the service that owns the handler, and is
// put in the typed errors.
func NewServer(h confighandler.Handler, domain outline.ServiceName) *Server {
	return &Server{h: h, domain: domain}
}

type route int

const (
	routeNone route = iota
	routeService
	routeSecretPut
	routeKeyDelete
	routeHistories
)

// find returns the route of the request, the key of the URI and the query.
func find(m *sock.Request) (route, string, url.Values) {
	path, rawQuery, _ := strings.Cut(m.URI, "?")

	if regService.MatchString(path) && m.Method == sock.RequestMethodGet {
		return routeService, "", nil
	}
	if sub := regSecret.FindStringSubmatch(path); sub != nil && m.Method == sock.RequestMethodPut {
		return routeSecretPut, sub[1], nil
	}
	if sub := regKey.FindStringSubmatch(path); sub != nil && m.Method == sock.RequestMethodDelete {
		return routeKeyDelete, sub[1], nil
	}
	if sub := regHistories.FindStringSubmatch(path); sub != nil && m.Method == sock.RequestMethodGet {
		query, err := url.ParseQuery(rawQuery)
		if err != nil {
			// still this route: Handle answers it with a 400
			return routeHistories, sub[1], nil
		}
		return routeHistories, sub[1], query
	}

	return routeNone, "", nil
}

// Match reports whether the request is one of the config requests. It is safe to call on a nil
// Server, which matches nothing.
func (s *Server) Match(m *sock.Request) bool {
	if s == nil || m == nil {
		return false
	}

	r, _, _ := find(m)

	return r != routeNone
}

// LogSafe returns the request to be logged. The body of every request that may hold a secret is
// replaced by a copy, and the original is left untouched. It decides by the URI alone, in lower
// case, and independent of the routing, so that a mistyped or unrouted secret request is
// redacted too: a URI that starts with /v1/config or ends with /secret is redacted.
func LogSafe(m *sock.Request) *sock.Request {
	uri := strings.ToLower(m.URI)
	if !strings.HasPrefix(uri, "/v1/config") && !strings.HasSuffix(uri, "/secret") {
		return m
	}

	res := *m
	res.Data = redactedData

	return &res
}

// Handle answers a request that Match accepted. Every failure is a response (a typed error), the
// returned error is always nil.
func (s *Server) Handle(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, handleTimeout)
	defer cancel()

	r, key, query := find(m)
	switch r {
	case routeService:
		return s.handleService(ctx), nil
	case routeSecretPut:
		return s.handleSecretPut(ctx, m, key), nil
	case routeKeyDelete:
		return s.handleKeyDelete(ctx, m, key), nil
	case routeHistories:
		return s.handleHistories(ctx, key, query), nil
	}

	return &sock.Response{StatusCode: 404}, nil
}

func (s *Server) handleService(ctx context.Context) *sock.Response {
	list, err := s.h.Describe(ctx)
	if err != nil {
		return s.errorResponse("Describe", err)
	}

	res := Service{SecretWritable: s.h.SecretWritable(), Keys: make([]Key, 0, len(list))}
	for _, r := range list {
		res.Keys = append(res.Keys, s.toKey(r))
	}

	return s.jsonResponse(res)
}

func (s *Server) handleSecretPut(ctx context.Context, m *sock.Request, key string) *sock.Response {
	// the body holds the secret: a parse failure is answered without any log or error text
	var req SecretPutRequest
	if err := json.Unmarshal(m.Data, &req); err != nil || req.Revision == nil {
		return s.invalid("The request body is not valid.")
	}
	if req.ActorID == uuid.Nil {
		return s.invalid("The actor is required.")
	}
	if !validSecretText(req.Secret) {
		return s.invalid("The secret must not start or end with white space and must not contain a line break.")
	}

	res, err := s.h.SetSecret(ctx, req.ActorID, key, req.Secret, confighandler.ExpectRevision(*req.Revision))
	if err != nil {
		return s.errorResponse("SetSecret", err)
	}

	return s.jsonResponse(s.toKey(res))
}

func (s *Server) handleKeyDelete(ctx context.Context, m *sock.Request, key string) *sock.Response {
	var req KeyDeleteRequest
	if err := json.Unmarshal(m.Data, &req); err != nil || req.Revision == nil {
		return s.invalid("The request body is not valid.")
	}
	if req.ActorID == uuid.Nil {
		return s.invalid("The actor is required.")
	}

	// only the secrets are written through this API: a setting is a different kind of change
	k, ok := s.h.Registry().Lookup(key)
	if !ok {
		return s.errorResponse("Clear", confighandler.ErrUnknownKey)
	}
	if k.Kind != confighandler.KindSecret {
		return s.errorResponse("Clear", confighandler.ErrWrongKind)
	}

	res, err := s.h.Clear(ctx, req.ActorID, key, confighandler.ExpectRevision(*req.Revision))
	if err != nil {
		return s.errorResponse("Clear", err)
	}

	return s.jsonResponse(s.toKey(res))
}

func (s *Server) handleHistories(ctx context.Context, key string, query url.Values) *sock.Response {
	if query == nil {
		return s.invalid("The query is not valid.")
	}

	size := 0
	if v := query.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return s.invalid("The page size is not valid.")
		}
		size = n
	}

	rows, next, err := s.h.Histories(ctx, key, query.Get("page_token"), size)
	if err != nil {
		return s.errorResponse("Histories", err)
	}

	res := HistoryList{Result: make([]History, 0, len(rows)), NextPageToken: next}
	for _, h := range rows {
		res.Result = append(res.Result, History{
			ID:            h.ID,
			Key:           h.Key,
			Action:        h.Action,
			IsSecret:      h.IsSecret,
			OldKeyVersion: h.OldKeyVersion,
			NewKeyVersion: h.NewKeyVersion,
			ActorID:       h.ActorID,
			Source:        h.Source,
			TMCreate:      h.TMCreate,
		})
	}

	return s.jsonResponse(res)
}

// toKey builds the description of a key from its resolved state and its declaration.
func (s *Server) toKey(r *confighandler.Resolved) Key {
	res := Key{
		Key:         r.Key,
		Kind:        string(r.Kind),
		Type:        string(r.Type),
		Source:      string(r.Source),
		HasSecret:   r.HasSecret,
		EnvShadowed: r.EnvShadowed,
		KeyVersion:  r.KeyVersion,
		Revision:    r.Revision,
		UpdatedBy:   r.UpdatedBy,
		TMUpdate:    r.TMUpdate,
	}

	if k, ok := s.h.Registry().Lookup(r.Key); ok {
		res.Description = k.Description
		res.EnvName = k.Env
	}

	return res
}

// validSecretText refuses a secret that starts or ends with white space or holds a line break: a
// value pasted with a trailing newline would be stored and then fail at the provider.
func validSecretText(secret string) bool {
	if strings.ContainsAny(secret, "\r\n") {
		return false
	}

	return strings.TrimFunc(secret, unicode.IsSpace) == secret
}

func (s *Server) jsonResponse(v any) *sock.Response {
	data, err := json.Marshal(v)
	if err != nil {
		return s.typed(cerrors.Internal(s.domain, reasonInternal, "An internal error occurred."))
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}
}

func (s *Server) typed(e *cerrors.VoipbinError) *sock.Response {
	res, err := cerrors.ToResponse(e)
	if err != nil {
		return &sock.Response{StatusCode: 500}
	}

	return res
}

func (s *Server) invalid(message string) *sock.Response {
	return s.typed(cerrors.InvalidArgument(s.domain, reasonInvalid, message))
}

// errorResponse maps an error of the library to a typed error. The messages are fixed text: the
// text of the error never leaves this function, except in the log line of an internal error,
// which holds the operation and the error text of the store (the errors of the library hold no
// value).
func (s *Server) errorResponse(op string, err error) *sock.Response {
	switch {
	// the write is stored: this comes before every other check, whatever the cause was
	case errors.Is(err, confighandler.ErrWriteVerify):
		return s.typed(cerrors.Internal(s.domain, reasonWriteVerify, "The change may have been stored, but it could not be confirmed. Reload to check the current state before you try again."))
	case errors.Is(err, confighandler.ErrConflict):
		return s.typed(cerrors.FailedPrecondition(s.domain, reasonMismatch, "The value was changed by someone else. Reload to see the current state."))
	case errors.Is(err, confighandler.ErrUnknownKey):
		return s.typed(cerrors.NotFound(s.domain, reasonKeyNotFound, "The config key does not exist."))
	case errors.Is(err, confighandler.ErrInvalid), errors.Is(err, confighandler.ErrWrongKind), errors.Is(err, confighandler.ErrTooLong):
		return s.invalid("The value does not fit the declaration of the key.")
	case errors.Is(err, confighandler.ErrKeyNotConfigured):
		return s.typed(cerrors.Unavailable(s.domain, reasonNoMasterKey, "The encryption key is not configured, so a secret can not be stored."))
	}

	logrus.WithField("func", op).Errorf("Could not process the config request. err: %v", err)

	return s.typed(cerrors.Internal(s.domain, reasonInternal, "An internal error occurred."))
}
