package subscribehandler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Test_topicPatterns_golden pins the EXACT bind set timeline-manager places on the
// global bin-manager.event topic exchange (VOIP-1406, design §5). timeline-manager is
// the archive-everything service: a single catch-all "#" binding, deliberately a
// superset of the old 25 fanout subscriptions, capturing every current and future
// topic publisher. Any change to topicPatterns must be a reviewed design decision
// that updates this golden in the same commit.
func Test_topicPatterns_golden(t *testing.T) {
	expectedPatterns := []string{
		"#",
	}

	if len(topicPatterns) != 1 {
		t.Fatalf("Wrong topicPatterns count. expect: 1, got: %d (%v)", len(topicPatterns), topicPatterns)
	}
	for i, expected := range expectedPatterns {
		if topicPatterns[i] != expected {
			t.Errorf("Wrong pattern at index %d. expect: %q, got: %q", i, expected, topicPatterns[i])
		}
	}
}

// Test_retainedFanoutTargets_golden pins the VOIP-1407 retention invariants this
// service carries -- the ONLY one of the 20 VOIP-1406 consumer services that carries
// BOTH §3.2 exceptions:
//
//  1. The asterisk fanout leg: a standalone QueueSubscribe(asterisk.all.event)
//     statement in Run(), never nested inside a loop. asterisk-proxy is excluded
//     from VOIP-1407's scope entirely and does not publish to the global topic
//     exchange, so this is one of the two fanout QueueSubscribe calls, across the
//     whole fanout-cutover, that survive.
//  2. The VOIP-1258 webhook-topic-bind block: RETAINED VERBATIM AND IN POSITION,
//     unrelated to the fanout-vs-topic cutover this ticket performs.
//
// Before VOIP-1407, this test pinned the fanout-subscribe-target list and the
// derived fanout-unbind-target list (two now-deleted package vars) directly:
// asterisk retention, a subscribe-list count of 26, a subscribe-list-minus-unbind-
// list arithmetic invariant of 1, and the VOIP-1258 exchange name. Both vars are now
// deleted (the generic fanout target loop and its unbind machinery are gone), so
// this test is REWRITTEN -- not dropped -- to assert the same two retention
// invariants against the new source shape, via AST
// inspection of Run() in main.go. The count/arithmetic assertions have no
// post-cutover equivalent and are dropped, not adapted.
func Test_retainedFanoutTargets_golden(t *testing.T) {
	// The retained VOIP-1258 webhook topic exchange: pin the exchange name so a
	// rename or accidental removal of the constant surfaces here.
	if string(commonoutline.QueueNameWebhookEventTopic) != "bin-manager.webhook-manager.event.topic" {
		t.Errorf("Wrong webhook topic exchange name. expect: %q, got: %q", "bin-manager.webhook-manager.event.topic", commonoutline.QueueNameWebhookEventTopic)
	}

	// And the global topic exchange the new "#" bind targets.
	if string(commonoutline.QueueNameEvent) != "bin-manager.event" {
		t.Errorf("Wrong global topic exchange name. expect: %q, got: %q", "bin-manager.event", commonoutline.QueueNameEvent)
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

	foundAsteriskStandalone := false
	foundAsteriskInsideLoop := false
	foundWebhookTopicBind := false
	foundLegacyWebhookUnbind := false
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
				foundAsteriskInsideLoop = true
			} else {
				foundAsteriskStandalone = true
			}
		}

		if isWebhookTopicQueueBindCall(n) {
			foundWebhookTopicBind = true
		}

		if isLegacyWebhookQueueUnbindCall(n) {
			foundLegacyWebhookUnbind = true
		}

		stack = append(stack, n)
		return true
	})

	if foundAsteriskInsideLoop {
		t.Error("QueueSubscribe(asterisk.all.event) must NOT be inside a for/range loop -- the generic fanout target loop was deleted by VOIP-1407; the asterisk leg is re-added as a standalone statement (design §3.2).")
	}
	if !foundAsteriskStandalone {
		t.Error("Run() must contain a standalone QueueSubscribe(subscribeQueue, string(commonoutline.QueueNameAsteriskEventAll)) statement -- the retained asterisk fanout leg (VOIP-1407 design §3.2).")
	}
	if !foundWebhookTopicBind {
		t.Error("Run() must contain a QueueBind(subscribeQueue, \"#\", string(commonoutline.QueueNameWebhookEventTopic), false, nil) call -- the retained VOIP-1258 webhook-topic-bind block (VOIP-1407 design §3.2), unrelated to and untouched by the fanout-vs-topic cutover.")
	}
	if !foundLegacyWebhookUnbind {
		t.Error("Run() must contain a QueueUnbind(subscribeQueue, \"\", string(commonoutline.QueueNameWebhookEvent), nil) call -- the retained VOIP-1258 block's legacy unbind, left untouched by VOIP-1407 design §3.2 sub-ruling 2.")
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
	return isStringConversionOf(call.Args[1], "QueueNameAsteriskEventAll")
}

// isWebhookTopicQueueBindCall reports whether n is a call of the shape
// <expr>.QueueBind(<any>, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).
func isWebhookTopicQueueBindCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 5 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "QueueBind" {
		return false
	}
	if !isStringLiteral(call.Args[1], "#") {
		return false
	}
	return isStringConversionOf(call.Args[2], "QueueNameWebhookEventTopic")
}

// isLegacyWebhookQueueUnbindCall reports whether n is a call of the shape
// <expr>.QueueUnbind(<any>, "", string(commonoutline.QueueNameWebhookEvent), nil).
func isLegacyWebhookQueueUnbindCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 4 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "QueueUnbind" {
		return false
	}
	if !isStringLiteral(call.Args[1], "") {
		return false
	}
	return isStringConversionOf(call.Args[2], "QueueNameWebhookEvent")
}

// isStringConversionOf reports whether e is the expression
// string(commonoutline.<selName>).
func isStringConversionOf(e ast.Expr, selName string) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "string" {
		return false
	}
	sel, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != selName {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == "commonoutline"
}

// isStringLiteral reports whether e is the raw string literal want.
func isStringLiteral(e ast.Expr, want string) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	unquoted, errUnquote := strconv.Unquote(lit.Value)
	return errUnquote == nil && unquoted == want
}
