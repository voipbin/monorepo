package servicehandler

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
)

const (
	builderSecret    = "SECRET-INPUT-STRING-do-not-log"
	builderCustomer  = "b3e2a1c0-0000-0000-0000-000000000002"
	builderOtherCust = "b3e2a1c0-0000-0000-0000-0000000000ff"
)

func builderAgent(perm amagent.Permission, customer string) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil(customer),
		},
		Permission: perm,
	})
}

func builderAccesskey() *auth.AuthIdentity {
	return auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{
		ID:         uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-0000000000aa"),
		CustomerID: uuid.FromStringOrNil(builderCustomer),
	})
}

func builderChatReq(content string) *builder.ChatRequest {
	return &builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleUser, Content: content}}}
}

func newBuilderServiceHandler(t *testing.T) (*serviceHandler, *requesthandler.MockRequestHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	return &serviceHandler{reqHandler: mockReq}, mockReq
}

func Test_AIBuilderChat_success(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)

	// The customer id sent to ai-manager is the AUTHENTICATED customer.
	mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), uuid.FromStringOrNil(builderCustomer), gomock.Any()).
		Return(&builder.ChatResponse{Message: "hi"}, nil)

	res, err := h.AIBuilderChat(context.Background(), a, builderChatReq("hello"))
	if err != nil || res.Message != "hi" {
		t.Fatalf("got %+v / %v", res, err)
	}
}

func Test_AIBuilderChat_permissions(t *testing.T) {
	tests := []struct {
		name   string
		a      *auth.AuthIdentity
		expect error
	}{
		{"admin", builderAgent(amagent.PermissionCustomerAdmin, builderCustomer), nil},
		{"manager", builderAgent(amagent.PermissionCustomerManager, builderCustomer), nil},
		// The platform pays for the LLM, so only a logged-in admin or manager may
		// spend it. HasPermission grants an accesskey and a delegate the admin bit,
		// so the agent requirement is what keeps a script from using the budget.
		{"plain agent", builderAgent(amagent.PermissionNone, builderCustomer), serviceerrors.ErrPermissionDenied},
		{"accesskey", builderAccesskey(), serviceerrors.ErrPermissionDenied},
		{"direct", auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil(builderCustomer)}), serviceerrors.ErrPermissionDenied},
		{"delegate", auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: uuid.FromStringOrNil(builderCustomer)}), serviceerrors.ErrPermissionDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockReq := newBuilderServiceHandler(t)
			if tt.expect == nil {
				mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(&builder.ChatResponse{Message: "ok"}, nil)
			}
			_, err := h.AIBuilderChat(context.Background(), tt.a, builderChatReq("hello")) // strict mock: no RPC when denied
			if !errors.Is(err, tt.expect) && !(tt.expect == nil && err == nil) {
				t.Errorf("got %v, want %v", err, tt.expect)
			}
		})
	}
}

// The request is validated here too, with the same function ai-manager uses, so
// a request that could never succeed does not cost an RPC.
func Test_AIBuilderChat_validatesBeforeTheRPC(t *testing.T) {
	h, _ := newBuilderServiceHandler(t) // strict mock
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)

	_, err := h.AIBuilderChat(context.Background(), a, &builder.ChatRequest{})
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != builder.ReasonInvalidArgument {
		t.Errorf("got %v", err)
	}
}

// Metrics. Both counters are raised here, in one place, so the server handler
// needs no way to reach them.
func Test_AIBuilderChat_timeoutBecomesBuilderTimeoutAndIsCounted(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, context.DeadlineExceeded)

	before := testutil.ToFloat64(metricBuilderTimeout)
	_, err := h.AIBuilderChat(context.Background(), a, builderChatReq("hello"))

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != builder.ReasonTimeout || ve.Status != cerrors.StatusUnavailable {
		t.Fatalf("got %v", err)
	}
	if testutil.ToFloat64(metricBuilderTimeout)-before != 1 {
		t.Error("the timeout counter must rise by one")
	}
}

// An error that WRAPS the deadline (the real RPC layer wraps) is still found.
func Test_AIBuilderChat_wrappedDeadlineIsStillFound(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	wrapped := errors.Join(errors.New("could not send"), context.DeadlineExceeded)
	mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, wrapped)

	_, err := h.AIBuilderChat(context.Background(), a, builderChatReq("hello"))
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != builder.ReasonTimeout {
		t.Errorf("got %v", err)
	}
}

func Test_AIBuilderChat_circuitOpenBecomesServiceUnavailableAndIsCounted(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.Join(errors.New("rpc"), circuitbreakerhandler.ErrCircuitOpen))

	before := testutil.ToFloat64(metricBuilderCircuitOpen)
	_, err := h.AIBuilderChat(context.Background(), a, builderChatReq("hello"))

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != "SERVICE_UNAVAILABLE" || ve.Status != cerrors.StatusUnavailable {
		t.Fatalf("got %v", err)
	}
	if testutil.ToFloat64(metricBuilderCircuitOpen)-before != 1 {
		t.Error("the circuit-open counter must rise by one")
	}
}

// A cancelled caller is not a timeout and is not counted as one.
func Test_AIBuilderChat_canceledIsNotCounted(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

	before := testutil.ToFloat64(metricBuilderTimeout)
	_, err := h.AIBuilderChat(context.Background(), a, builderChatReq("hello"))
	if err == nil {
		t.Fatal("expected an error")
	}
	var ve *cerrors.VoipbinError
	if errors.As(err, &ve) && ve.Reason == builder.ReasonTimeout {
		t.Error("a cancel must not be reported as a timeout")
	}
	if testutil.ToFloat64(metricBuilderTimeout) != before {
		t.Error("a cancel must not be counted")
	}
}

// A typed error from ai-manager is passed through untouched (its reason is what
// the client acts on), and no counter is raised for it.
func Test_AIBuilderChat_typedErrorPassesThrough(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	typed := cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "x")
	mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, typed)

	t0, c0 := testutil.ToFloat64(metricBuilderTimeout), testutil.ToFloat64(metricBuilderCircuitOpen)
	_, err := h.AIBuilderChat(context.Background(), a, builderChatReq("hello"))
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != builder.ReasonDailyLimit {
		t.Fatalf("got %v", err)
	}
	if testutil.ToFloat64(metricBuilderTimeout) != t0 || testutil.ToFloat64(metricBuilderCircuitOpen) != c0 {
		t.Error("no counter may rise for a typed error")
	}
}

// Logs carry the agent, the customer and a message count, never the input.
func Test_AIBuilderChat_neverLogsTheInput(t *testing.T) {
	var buf strings.Builder
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetOutput(logrus.StandardLogger().Out)

	for name, rpcErr := range map[string]error{
		"ok":        nil,
		"echoing":   errors.New("400: " + builderSecret),
		"timeout":   context.DeadlineExceeded,
		"circuit":   circuitbreakerhandler.ErrCircuitOpen,
		"typed err": cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "x").Wrap(errors.New(builderSecret)),
	} {
		t.Run(name, func(t *testing.T) {
			buf.Reset()
			h, mockReq := newBuilderServiceHandler(t)
			a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
			if rpcErr == nil {
				mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(&builder.ChatResponse{Message: "ok"}, nil)
			} else {
				mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, rpcErr)
			}
			_, _ = h.AIBuilderChat(context.Background(), a, &builder.ChatRequest{
				Messages:     []builder.Message{{Role: builder.RoleUser, Content: builderSecret}},
				CurrentDraft: &builder.Draft{Name: builderSecret, InitPrompt: builderSecret},
			})
			if strings.Contains(buf.String(), builderSecret) {
				t.Errorf("a log line carries the input:\n%s", buf.String())
			}
		})
	}
}

// ---------------- status ----------------

func Test_AIBuilderStatus_permissionIsCheckedBeforeTheCache(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	// A non-permitted caller gets available=false, and no RPC is made for them
	// even when the cache is cold.
	for _, a := range []*auth.AuthIdentity{
		builderAgent(amagent.PermissionNone, builderCustomer),
		builderAccesskey(),
		auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil(builderCustomer)}),
	} {
		res, err := h.AIBuilderStatus(context.Background(), a)
		if err != nil || res == nil || res.Available {
			t.Errorf("got %+v / %v", res, err)
		}
	}
	_ = mockReq // strict: any RPC would fail the test
}

func Test_AIBuilderStatus_reportsTheBackendAnswerAndCachesIt(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).Return(&builder.StatusResponse{Available: true, MaxMessages: 40, MaxMessageChars: 2000}, nil).Times(1)

	for i := 0; i < 3; i++ {
		res, err := h.AIBuilderStatus(context.Background(), a)
		if err != nil || !res.Available || res.MaxMessages != 40 {
			t.Fatalf("call %d: %+v / %v", i, res, err)
		}
	}
}

// A failed RPC is reported as unavailable (never as an error: the client uses
// this to decide whether to show a button), and is cached too, shorter.
func Test_AIBuilderStatus_failureIsUnavailableAndCachedBriefly(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).Return(nil, errors.New("down")).Times(1)

	for i := 0; i < 3; i++ {
		res, err := h.AIBuilderStatus(context.Background(), a)
		if err != nil || res == nil || res.Available {
			t.Fatalf("call %d: %+v / %v", i, res, err)
		}
	}
}

func Test_AIBuilderStatus_cacheExpires(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	now := time.Now()
	h.builderStatus.now = func() time.Time { return now }

	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).Return(&builder.StatusResponse{Available: true}, nil)
	_, _ = h.AIBuilderStatus(context.Background(), a)

	// Inside the success window: cached.
	now = now.Add(builderStatusOKTTL - time.Second)
	_, _ = h.AIBuilderStatus(context.Background(), a)

	// Past it: asked again, and the new answer is used.
	now = now.Add(2 * time.Second)
	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).Return(&builder.StatusResponse{Available: false}, nil)
	res, _ := h.AIBuilderStatus(context.Background(), a)
	if res.Available {
		t.Error("an expired answer must be replaced")
	}
}

func Test_AIBuilderStatus_failureWindowIsShorterThanSuccess(t *testing.T) {
	if !(builderStatusFailTTL < builderStatusOKTTL) || builderStatusOKTTL != 30*time.Second || builderStatusFailTTL != 5*time.Second {
		t.Errorf("ttls: ok=%v fail=%v", builderStatusOKTTL, builderStatusFailTTL)
	}

	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	now := time.Now()
	h.builderStatus.now = func() time.Time { return now }

	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).Return(nil, errors.New("down"))
	_, _ = h.AIBuilderStatus(context.Background(), a)

	now = now.Add(builderStatusFailTTL + time.Second)
	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).Return(&builder.StatusResponse{Available: true}, nil)
	res, _ := h.AIBuilderStatus(context.Background(), a)
	if !res.Available {
		t.Error("a failure must be retried after the short window")
	}
}

// Many callers on an expired cache cause ONE RPC, and none of them holds the
// lock while it runs.
func Test_AIBuilderStatus_concurrentExpiryMakesOneRPC(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)

	var calls int32
	release := make(chan struct{})
	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).DoAndReturn(func(context.Context) (*builder.StatusResponse, error) {
		atomic.AddInt32(&calls, 1)
		<-release
		return &builder.StatusResponse{Available: true}, nil
	}).Times(1)

	var wg sync.WaitGroup
	results := make([]*builder.StatusResponse, 20)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], _ = h.AIBuilderStatus(context.Background(), a)
		}(i)
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()

	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected one RPC, got %d", calls)
	}
	for i, r := range results {
		if r == nil || !r.Available {
			t.Errorf("caller %d got %+v", i, r)
		}
	}
}

// The first caller giving up must not fail the others, and its cancellation
// must not be cached as an outage.
func Test_AIBuilderStatus_aCancelledCallerDoesNotPoisonTheCache(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	mockReq.EXPECT().AIV1BuilderStatus(gomock.Any()).DoAndReturn(func(ctx context.Context) (*builder.StatusResponse, error) {
		entered <- struct{}{}
		select {
		case <-release:
			return &builder.StatusResponse{Available: true}, nil
		case <-ctx.Done():
			return nil, ctx.Err() // the shared RPC must not be tied to the first caller's ctx
		}
	}).Times(1)

	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstDone := make(chan *builder.StatusResponse, 1)
	go func() { r, _ := h.AIBuilderStatus(firstCtx, a); firstDone <- r }()
	<-entered

	secondDone := make(chan *builder.StatusResponse, 1)
	go func() { r, _ := h.AIBuilderStatus(context.Background(), a); secondDone <- r }()
	time.Sleep(50 * time.Millisecond)

	cancelFirst()
	select {
	case r := <-firstDone:
		if r == nil || r.Available {
			t.Errorf("the cancelled caller must get available=false, got %+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the cancelled caller did not return")
	}

	close(release)
	select {
	case r := <-secondDone:
		if r == nil || !r.Available {
			t.Errorf("the second caller must still get the real answer, got %+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the second caller did not return")
	}

	// And the answer that did arrive is what is cached.
	res, _ := h.AIBuilderStatus(context.Background(), a)
	if !res.Available {
		t.Error("the cache must hold the successful answer, not the first caller's cancellation")
	}
}
