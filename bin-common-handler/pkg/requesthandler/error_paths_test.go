package requesthandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// errPathSock is a deterministic stand-in for the message broker used by the
// tables below. Hand written on purpose (no gomock): every table in this
// file checks the publish count and the request fields for every one of the
// ~550 exported wrapper methods in one place, which a mock controller per
// sub-test would not make simpler.
type errPathSock struct {
	sockhandler.SockHandler

	res   *sock.Response
	err   error
	calls int

	lastQueue string
	lastReq   *sock.Request
}

func (s *errPathSock) RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error) {
	s.calls++
	s.lastQueue = queueName
	s.lastReq = req
	return s.res, s.err
}

func (s *errPathSock) RequestPublishWithDelay(queueName string, req *sock.Request, delay int) error {
	s.calls++
	s.lastQueue = queueName
	s.lastReq = req
	return s.err
}

// errPathNoSockCall lists the methods that reject a zero-valued argument set
// before ever reaching the broker. They are covered by their own tests below
// (Test_AgentV1AgentGetsByTagIDs_Errors).
var errPathNoSockCall = map[string]bool{
	"AgentV1AgentGetsByTagIDs":          true,
	"AgentV1AgentGetsByTagIDsAndStatus": true,
}

// errPathNotRPC lists the exported methods that publish an event rather than
// send a request/response RPC, so the request stand-in above does not apply.
var errPathNotRPC = map[string]bool{
	"CallPublishEvent": true,
}

// errPathNoFieldCheck lists the methods whose zero-valued call does not
// produce a non-empty queue/URI/method (SendRequest forwards its own,
// zero-valued, arguments verbatim). Covered separately below.
var errPathNoFieldCheck = map[string]bool{
	"SendRequest": true,
}

// errPathIgnoreStatus lists the methods that do not turn a 404/500 response
// into an error (confirmed by actually running the (b) table against every
// wrapper with a `Data: null` response before adding a name here). New
// wrappers that start doing so will fail the (b) table and force this list
// to be revisited; the entries carry no further reasoning on purpose.
var errPathIgnoreStatus = map[string]bool{
	"AgentV1AgentReserveRelease": true,
	"ContactV1CaseNoteDelete":    true,
	"ContactV1CaseTagAdd":        true,
	"ContactV1CaseTagRemove":     true,
	"PipecatV1Ping":              true,
	"SendRequest":                true,
	"AIV1ToolList":               true,
}

// Test_errPathExclusionListsReferenceRealMethods guards against a renamed or
// removed wrapper silently going stale in the lists above.
func Test_errPathExclusionListsReferenceRealMethods(t *testing.T) {
	typ := reflect.TypeOf(&requestHandler{})
	exists := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		exists[typ.Method(i).Name] = true
	}

	for _, list := range []map[string]bool{errPathNoSockCall, errPathNotRPC, errPathNoFieldCheck, errPathIgnoreStatus} {
		for name := range list {
			if !exists[name] {
				t.Errorf("exclusion list references %q which is not a requestHandler method", name)
			}
		}
	}
}

// errPathArgs builds a zero-valued argument list. context.Context becomes a
// real background context (its zero value is nil, which every wrapper would
// otherwise pass straight into context.WithTimeout and panic on); pointers
// point at a zero value so a wrapper that dereferences one does not panic.
func errPathArgs(mt reflect.Type) []reflect.Value {
	args := make([]reflect.Value, mt.NumIn())
	for i := range args {
		in := mt.In(i)
		switch {
		case in == reflect.TypeOf((*context.Context)(nil)).Elem():
			args[i] = reflect.ValueOf(context.Background())
		case in.Kind() == reflect.Pointer:
			args[i] = reflect.New(in.Elem())
		default:
			args[i] = reflect.Zero(in)
		}
	}
	return args
}

// errPathCall invokes the named wrapper on a handler backed by the given
// broker stand-in. A panic is reported as the sub-test's only failure (no
// "no return values" follow-up).
func errPathCall(t *testing.T, name string, fs *errPathSock) ([]reflect.Value, bool) {
	t.Helper()

	h := &requestHandler{sock: fs}
	m := reflect.ValueOf(h).MethodByName(name)
	mt := m.Type()

	panicked := false
	out := func() []reflect.Value {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s panicked: %v", name, r)
				panicked = true
			}
		}()
		if mt.IsVariadic() {
			return m.CallSlice(errPathArgs(mt))
		}
		return m.Call(errPathArgs(mt))
	}()

	return out, panicked
}

// errPathExpectFailure checks that the wrapper returned a non-nil error and
// that every other return value is the zero value (reflect.Value.IsZero: a
// nil slice is zero, an empty non-nil slice is not).
func errPathExpectFailure(t *testing.T, name string, out []reflect.Value) error {
	t.Helper()

	if len(out) == 0 {
		t.Errorf("Wrong match. name: %s, expect: a return value, got: none", name)
		return nil
	}

	err, _ := out[len(out)-1].Interface().(error)
	if err == nil {
		t.Errorf("Wrong match. name: %s, expect: an error, got: nil", name)
		return nil
	}

	for _, v := range out[:len(out)-1] {
		if !v.IsZero() {
			t.Errorf("Wrong match. name: %s, expect: a zero value alongside the error, got: %v", name, v.Interface())
		}
	}

	return err
}

func errPathMethodNames() []string {
	typ := reflect.TypeOf(&requestHandler{})
	names := make([]string, 0, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		names = append(names, typ.Method(i).Name)
	}
	sort.Strings(names)
	return names
}

func Test_RPCWrappers_TransportErrorPropagates(t *testing.T) {
	sentinel := errors.New("broker unavailable")

	for _, name := range errPathMethodNames() {
		if errPathNoSockCall[name] || errPathNotRPC[name] {
			continue
		}

		t.Run(name, func(t *testing.T) {
			fs := &errPathSock{err: sentinel}
			out, panicked := errPathCall(t, name, fs)
			if panicked {
				return
			}

			if fs.calls != 1 {
				t.Errorf("Wrong match. name: %s, expect: exactly one publish, got: %d", name, fs.calls)
			}
			if !errPathNoFieldCheck[name] {
				if fs.lastQueue == "" {
					t.Errorf("Wrong match. name: %s, expect: a non-empty queue, got: empty", name)
				}
				if fs.lastReq != nil {
					if fs.lastReq.URI == "" {
						t.Errorf("Wrong match. name: %s, expect: a non-empty URI, got: empty", name)
					}
					if fs.lastReq.Method == "" {
						t.Errorf("Wrong match. name: %s, expect: a non-empty method, got: empty", name)
					}
				}
			}

			err := errPathExpectFailure(t, name, out)
			if err != nil && !errors.Is(err, sentinel) {
				t.Errorf("Wrong match. name: %s, expect: the transport error propagated, got: %v", name, err)
			}
		})
	}
}

// Test_SendRequest_TransportErrorPropagates covers SendRequest with a
// non-zero target, since the table above skips the empty-field assertion
// for it (it forwards its arguments verbatim, so the zero-valued call does
// not produce a non-empty queue/URI/method).
func Test_SendRequest_TransportErrorPropagates(t *testing.T) {
	sentinel := errors.New("broker unavailable")
	fs := &errPathSock{err: sentinel}
	h := &requestHandler{sock: fs}

	res, err := h.SendRequest(context.Background(), "bin-manager.pod-a.request", "/v1/pod-a", sock.RequestMethodGet, requestTimeoutDefault, 0, ContentTypeNone, nil)
	if res != nil || !errors.Is(err, sentinel) {
		t.Errorf("Wrong match. expect: nil result and the transport error, got: res=%v, err=%v", res, err)
	}
	if fs.calls != 1 || fs.lastQueue != "bin-manager.pod-a.request" {
		t.Errorf("Wrong match. expect: one publish to bin-manager.pod-a.request, got: calls=%d, queue=%s", fs.calls, fs.lastQueue)
	}
}

// Test_SendRequest_StatusResponseIsReturnedAsIs asserts the documented
// contract of SendRequest: it returns the raw response, status included,
// without converting a non-2xx status into an error. This is a contract
// check, not a record of an unfixed defect in a different wrapper.
func Test_SendRequest_StatusResponseIsReturnedAsIs(t *testing.T) {
	fs := &errPathSock{res: &sock.Response{StatusCode: 404, Data: []byte("null")}}
	h := &requestHandler{sock: fs}

	res, err := h.SendRequest(context.Background(), "bin-manager.pod-a.request", "/v1/pod-a", sock.RequestMethodGet, requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil || res == nil || res.StatusCode != 404 {
		t.Errorf("Wrong match. expect: the raw 404 response and no error, got: res=%v, err=%v", res, err)
	}
}

func Test_RPCWrappers_ErrorStatusResponseReturnsError(t *testing.T) {
	statusCodes := []int{404, 500}

	for _, name := range errPathMethodNames() {
		if errPathNoSockCall[name] || errPathNotRPC[name] || errPathIgnoreStatus[name] {
			continue
		}

		for _, code := range statusCodes {
			t.Run(fmt.Sprintf("%s/%d", name, code), func(t *testing.T) {
				fs := &errPathSock{res: &sock.Response{StatusCode: code, Data: []byte("null")}}
				out, panicked := errPathCall(t, name, fs)
				if panicked {
					return
				}

				errPathExpectFailure(t, name, out)
			})
		}
	}
}

func Test_AgentV1AgentGetsByTagIDs_Errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f3a0e2c-1a52-11ee-9f3e-2b4d6c8e0a11")
	tagIDs := []uuid.UUID{
		uuid.FromStringOrNil("6a4b1f3d-1a52-11ee-9f3e-2b4d6c8e0a12"),
		uuid.FromStringOrNil("7b5c2a4e-1a52-11ee-9f3e-2b4d6c8e0a13"),
	}
	sentinel := errors.New("broker unavailable")

	calls := map[string]func(h *requestHandler, ids []uuid.UUID) ([]amagent.Agent, error){
		"GetsByTagIDs": func(h *requestHandler, ids []uuid.UUID) ([]amagent.Agent, error) {
			return h.AgentV1AgentGetsByTagIDs(context.Background(), customerID, ids)
		},
		"GetsByTagIDsAndStatus": func(h *requestHandler, ids []uuid.UUID) ([]amagent.Agent, error) {
			return h.AgentV1AgentGetsByTagIDsAndStatus(context.Background(), customerID, ids, amagent.StatusAvailable)
		},
	}

	for name, call := range calls {
		t.Run(name+"/empty tag ids", func(t *testing.T) {
			fs := &errPathSock{}
			res, err := call(&requestHandler{sock: fs}, nil)
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: an error and no result, got: res=%v, err=%v", res, err)
			}
			if fs.calls != 0 {
				t.Errorf("Wrong match. expect: no publish, got: %d", fs.calls)
			}
		})

		t.Run(name+"/transport error", func(t *testing.T) {
			fs := &errPathSock{err: sentinel}
			res, err := call(&requestHandler{sock: fs}, tagIDs)
			if !errors.Is(err, sentinel) || res != nil {
				t.Errorf("Wrong match. expect: the transport error and no result, got: res=%v, err=%v", res, err)
			}
		})

		t.Run(name+"/not found", func(t *testing.T) {
			fs := &errPathSock{res: &sock.Response{StatusCode: 404, Data: []byte("null")}}
			res, err := call(&requestHandler{sock: fs}, tagIDs)
			if !errors.Is(err, ErrNotFound) || res != nil {
				t.Errorf("Wrong match. expect: ErrNotFound and no result, got: res=%v, err=%v", res, err)
			}
		})
	}
}

// Test_RPCWrappers_DelayedTransportErrorPropagates covers the delayed-send
// branch of sendRequest, shared by every wrapper that takes a delay
// argument. A single representative wrapper is used because they all go
// through the same sendRequest code path.
func Test_RPCWrappers_DelayedTransportErrorPropagates(t *testing.T) {
	sentinel := errors.New("broker unavailable")
	fs := &errPathSock{err: sentinel}
	h := &requestHandler{sock: fs}

	err := h.ConferenceV1ConferenceDeleteDelay(context.Background(), uuid.Nil, 1000)
	if !errors.Is(err, sentinel) {
		t.Errorf("Wrong match. expect: the delayed transport error propagated, got: %v", err)
	}
	if fs.calls != 1 {
		t.Errorf("Wrong match. expect: exactly one delayed publish, got: %d", fs.calls)
	}
}
