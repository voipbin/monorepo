package subscribehandler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_CallsQueueCreateAndConsume is a functional smoke test: Run() must declare this
// pod's queue and start consuming from it, with no error, using the arguments the rest of the
// service (and pkg/websockhandler's scopeRefCount, which binds against the same queue name)
// depends on.
func Test_Run_CallsQueueCreateAndConsume(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := "bin-manager.api-manager.subscribe.test-pod"

	mockSock.EXPECT().QueueCreate(queueName, "volatile").Return(nil)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameAPIManager), false, false, false, 10, gomock.Any()).
		Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, nil, queueName, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

// Test_Run_QueueCreateBeforeConsumeGoroutine_golden is a regression test guarding Run()'s
// synchronous/async ordering after VOIP-1425 removed the dead fanout subscribeTargets
// machinery.
//
// A runtime/mock-based ordering test was tried first and rejected: ConsumeMessage's mock
// returns instantly, so goroutine-scheduling latency alone dominates any observable timing --
// mutation testing confirmed that a deliberately-introduced regression (moving the QueueCreate
// call to after the `go func(){ ... ConsumeMessage(...) }()` statement) still passed a
// runtime-based ordering assertion 100% of the time, both with a plain channel drain and with
// gomock.InOrder, because QueueCreate remains synchronous either way and the spawned goroutine
// essentially never gets scheduled before Run() itself returns in a mocked (zero-latency) test.
// This is structurally different from the sibling services (bin-agent-manager,
// bin-call-manager, bin-timeline-manager), whose Run() performs several real synchronous
// QueueBind/QueueUnbind calls before the goroutine launch -- enough sequential work for a
// reordering to be runtime-observable. api-manager's Run() has exactly one synchronous op left
// after VOIP-1425, so the only reliable way to pin its position is structurally, matching the
// AST-based approach bin-call-manager/pkg/subscribehandler/binding_golden_test.go already uses
// for this same class of invariant (see Test_fanoutUnbindTargets_golden there).
//
// This test parses main.go and asserts, within func (h *subscribeHandler) Run(), that the
// top-level statement calling QueueCreate appears before the top-level `go func() { ... }()`
// statement that launches ConsumeMessage.
func Test_Run_QueueCreateBeforeConsumeGoroutine_golden(t *testing.T) {
	fset := token.NewFileSet()
	f, errParse := parser.ParseFile(fset, "main.go", nil, 0)
	if errParse != nil {
		t.Fatalf("could not parse main.go. err: %v", errParse)
	}

	var runFunc *ast.FuncDecl
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "Run" && fn.Recv != nil {
			runFunc = fn
			break
		}
	}
	if runFunc == nil {
		t.Fatal("could not find func (h *subscribeHandler) Run() in main.go")
	}

	queueCreateIdx := -1
	goStmtIdx := -1
	for i, stmt := range runFunc.Body.List {
		if queueCreateIdx == -1 && containsQueueCreateCall(stmt) {
			queueCreateIdx = i
		}
		if goStmtIdx == -1 {
			if _, ok := stmt.(*ast.GoStmt); ok {
				goStmtIdx = i
			}
		}
	}

	if queueCreateIdx == -1 {
		t.Fatal("Run() must contain a top-level statement calling QueueCreate -- the synchronous queue declaration this pod's ConsumeMessage (and pkg/websockhandler's scopeRefCount) depend on existing first.")
	}
	if goStmtIdx == -1 {
		t.Fatal("Run() must contain a top-level `go func() { ... }()` statement launching the ConsumeMessage goroutine.")
	}
	if queueCreateIdx >= goStmtIdx {
		t.Fatalf("QueueCreate must be called BEFORE the ConsumeMessage goroutine is launched (statement index %d), but it appears at index %d -- ordering regression. QueueCreate and the started goroutine share the same AMQP queue; declaring it after starting the consumer risks the consumer racing a not-yet-declared queue.", goStmtIdx, queueCreateIdx)
	}
}

// containsQueueCreateCall reports whether stmt is (or directly wraps, e.g. an if-statement's
// init/cond, matching how Run() actually calls it: `if err := h.sockHandler.QueueCreate(...);
// err != nil { ... }`) a call to QueueCreate.
func containsQueueCreateCall(stmt ast.Stmt) bool {
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "QueueCreate" {
			found = true
		}
		return true
	})
	return found
}
