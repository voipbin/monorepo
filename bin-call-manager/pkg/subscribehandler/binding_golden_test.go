package subscribehandler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Test_topicPatterns_golden pins the EXACT pattern strings this service binds on the
// global topic exchange `bin-manager.event` (VOIP-1406, design §5: call-manager).
// The expected values are deliberate string literals, NOT eventtopic calls: this is the
// consumer-side sibling of the routing-key golden tests, and it must catch drift in
// either direction -- a dispatch case added without a pattern, or a pattern added
// without a case, or a change in the eventtopic normalization itself.
func Test_topicPatterns_golden(t *testing.T) {
	expectedPatterns := []string{
		"customer-manager.customer.*.deleted",
		"customer-manager.customer.*.frozen",
		"flow-manager.activeflow.*.updated",
		"sentinel-manager.pod.*.deleted",
	}

	if len(topicPatterns) != 4 {
		t.Errorf("Wrong pattern count. expect: 4, got: %d (%v)", len(topicPatterns), topicPatterns)
	}

	if len(topicPatterns) != len(expectedPatterns) {
		t.Fatalf("Wrong match. expect: %v, got: %v", expectedPatterns, topicPatterns)
	}
	for i, expected := range expectedPatterns {
		if topicPatterns[i] != expected {
			t.Errorf("Wrong pattern at index %d. expect: %s, got: %s", i, expected, topicPatterns[i])
		}
	}
}

// Test_fanoutUnbindTargets_golden pins the VOIP-1407 retained-asterisk invariant: this
// service keeps exactly one fanout leg -- QueueSubscribe(asterisk.all.event) -- as a
// standalone statement in Run(), never nested inside a loop. asterisk-proxy is excluded
// from VOIP-1407's scope entirely and does not publish to the global topic exchange, so
// this is the one fanout QueueSubscribe, across the whole fanout-cutover, that survives.
//
// Before VOIP-1407, this test pinned the `fanoutUnbindTargets`/`subscribeTargets`
// package vars directly and asserted the asterisk target was excluded from the unbind
// set. Both vars are now deleted (the generic fanout target loop and its unbind
// machinery are gone), so this test is REWRITTEN -- not dropped -- to assert the same
// retention invariant against the new source shape, via AST inspection of Run() in
// main.go, independent of any subscribeTargets/fanoutUnbindTargets var.
func Test_fanoutUnbindTargets_golden(t *testing.T) {
	retained := "asterisk.all.event"
	if string(commonoutline.QueueNameAsteriskEventAll) != retained {
		t.Fatalf("Wrong retained asterisk target. expect: %s, got: %s", retained, string(commonoutline.QueueNameAsteriskEventAll))
	}

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

	foundStandalone := false
	foundInsideLoop := false
	var stack []ast.Node
	ast.Inspect(runFunc.Body, func(n ast.Node) bool {
		if n == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return true
		}

		if isAsteriskQueueSubscribeCall(n) {
			inLoop := false
			for _, anc := range stack {
				switch anc.(type) {
				case *ast.ForStmt, *ast.RangeStmt:
					inLoop = true
				}
			}
			if inLoop {
				foundInsideLoop = true
			} else {
				foundStandalone = true
			}
		}

		stack = append(stack, n)
		return true
	})

	if foundInsideLoop {
		t.Error("QueueSubscribe(asterisk.all.event) must NOT be inside a for/range loop -- the generic fanout target loop was deleted by VOIP-1407; the asterisk leg is re-added as a standalone statement (design §3.2).")
	}
	if !foundStandalone {
		t.Error("Run() must contain a standalone QueueSubscribe(subscribeQueue, string(commonoutline.QueueNameAsteriskEventAll)) statement -- the retained asterisk fanout leg (VOIP-1407 design §3.2).")
	}
}

// isAsteriskQueueSubscribeCall reports whether n is a call of the shape
// <expr>.QueueSubscribe(<any>, string(commonoutline.QueueNameAsteriskEventAll)).
func isAsteriskQueueSubscribeCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "QueueSubscribe" {
		return false
	}
	return isAsteriskEventAllConversion(call.Args[1])
}

// isAsteriskEventAllConversion reports whether e is the expression
// string(commonoutline.QueueNameAsteriskEventAll).
func isAsteriskEventAllConversion(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "string" {
		return false
	}
	sel, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "QueueNameAsteriskEventAll" {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == "commonoutline"
}
