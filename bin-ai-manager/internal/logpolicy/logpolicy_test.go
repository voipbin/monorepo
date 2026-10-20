// Package logpolicy holds the source scan of the log policy (VOIP-1633): a log call must not pass
// conversation text, a model that holds it, or a request or event payload to the logger.
//
// The scan is syntactic (go/ast, no type information). It is a regression guard. The proof that a
// path does not log a sentence is the log capture tests of the packages (see internal/logtest).
// Not seen: a field map built in a variable, a constant key, a text copied into another variable
// first, an encoding other than %x and %X, and a call through an interface that wraps the logger.
package logpolicy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// auditedKeys are the field keys that held conversation text, a model or a payload in the audit.
var auditedKeys = map[string]bool{
	"request": true, "event": true, "message": true, "aicall": true, "ai_call": true, "pipecatcall": true,
	"summary": true, "content": true, "session": true, "frame": true, "response": true, "variable": true,
	"tool_arg": true, "tool_call": true, "request_content": true, "pipecat_message": true,
	"conversation_message": true, "team": true, "function": true, "answer": true, "mcp_server": true, "activeflow": true,
}

// allowed lists the audited keys that are kept on purpose. The key is "<file>|<key>|<value>" and the
// value is the reason. An entry without a reason is a mistake.
var allowed = map[string]string{
	"cmd/ai-manager/main.go|arg|m": "m is a startup note of the builder configuration check, not conversation text",
}

// riskyIdents are variable names that held a model, a payload or a text in the audit.
var riskyIdents = map[string]bool{
	"tmp": true, "res": true, "m": true, "evt": true, "msg": true, "message": true, "content": true,
	"text": true, "req": true, "resp": true, "frame": true, "data": true, "body": true, "answer": true,
	"messageText": true, "title": true, "cc": true, "sm": true, "pc": true, "af": true,
}

// riskyFields are struct fields that hold conversation text.
var riskyFields = map[string]bool{
	"Content": true, "Text": true, "Data": true, "Prompt": true, "Arguments": true,
	"LLMMessages": true, "Messages": true, "InitPrompt": true, "Digit": true,
	"Header": true, "Params": true, "Metadata": true, "Variables": true,
}

var logMethods = map[string]bool{
	"Debug": true, "Info": true, "Warn": true, "Warning": true, "Error": true, "Trace": true,
	"Debugf": true, "Infof": true, "Warnf": true, "Warningf": true, "Errorf": true, "Tracef": true,
	"Debugln": true, "Infoln": true, "Warnln": true, "Warningln": true, "Errorln": true,
	"Print": true, "Printf": true, "Println": true, "Fatal": true, "Fatalf": true, "Fatalln": true,
	"Panic": true, "Panicf": true, "Panicln": true,
}

func exprString(fset *token.FileSet, e ast.Expr) string {
	var sb strings.Builder
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			sb.WriteString(x.Name)
		case *ast.SelectorExpr:
			sb.WriteString(exprString(fset, x.X))
			sb.WriteString(".")
			sb.WriteString(x.Sel.Name)
			return false
		case *ast.CallExpr:
			sb.WriteString(exprString(fset, x.Fun))
			sb.WriteString("()")
			return false
		}
		return true
	})
	return sb.String()
}

func isRisky(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.StarExpr:
		return isRisky(x.X)
	case *ast.UnaryExpr:
		return isRisky(x.X)
	case *ast.ParenExpr:
		return isRisky(x.X)
	case *ast.IndexExpr:
		return isRisky(x.X)
	case *ast.SliceExpr:
		return isRisky(x.X)
	case *ast.BinaryExpr:
		return isRisky(x.X) || isRisky(x.Y)
	case *ast.Ident:
		return riskyIdents[x.Name]
	case *ast.SelectorExpr:
		return riskyFields[x.Sel.Name]
	case *ast.CallExpr:
		// string(x) of a risky value
		if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "string" && len(x.Args) == 1 {
			return isRisky(x.Args[0])
		}
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
			switch sel.Sel.Name {
			case "GetHeader":
				// a request header can carry an authorization value
				return true
			case "Sprint", "Sprintf", "Sprintln":
				for _, a := range x.Args {
					if isRisky(a) {
						return true
					}
				}
			case "String":
				return isRisky(sel.X)
			}
		}
	}
	return false
}

// verbs returns the conversion verbs of a format string in order, without flags and widths.
func verbs(format string) []byte {
	var res []byte
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		for i < len(format) && strings.IndexByte("+-# 0123456789.*[]", format[i]) >= 0 {
			i++
		}
		if i >= len(format) {
			break
		}
		if format[i] == '%' {
			continue
		}
		res = append(res, format[i])
	}
	return res
}

type finding struct {
	pos  string
	text string
}

func scanFile(fset *token.FileSet, file *ast.File, rel string) []finding {
	var res []finding

	checkKey := func(key string, val ast.Expr, pos token.Pos) {
		// a value that is text by its field name is refused under any key
		if sel, ok := val.(*ast.SelectorExpr); ok && riskyFields[sel.Sel.Name] || !auditedKeys[key] && isRisky(val) {
			id := rel + "|" + key + "|" + exprString(fset, val)
			if _, ok := allowed[id]; !ok {
				res = append(res, finding{fset.Position(pos).String(), "log field " + key + " passes " + exprString(fset, val)})
			}
			return
		}
		if !auditedKeys[key] {
			return
		}
		id := rel + "|" + key + "|" + exprString(fset, val)
		if reason, ok := allowed[id]; ok {
			if reason == "" {
				res = append(res, finding{fset.Position(pos).String(), "allowlist entry without a reason: " + id})
			}
			return
		}
		res = append(res, finding{fset.Position(pos).String(), "log field " + key + " passes " + exprString(fset, val)})
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CompositeLit:
			// logrus.Fields{...}
			if sel, ok := x.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Fields" {
				for _, el := range x.Elts {
					kv, ok := el.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if lit, ok := kv.Key.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						key, _ := strconv.Unquote(lit.Value)
						checkKey(key, kv.Value, kv.Pos())
					}
				}
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			// the level is set in internal/config only, from the setting; a fixed level elsewhere would
			// bring the Debug lines back
			if sel.Sel.Name == "SetLevel" && !strings.HasPrefix(rel, "internal/config/") {
				res = append(res, finding{fset.Position(x.Pos()).String(), "SetLevel outside internal/config"})
			}
			if sel.Sel.Name == "WithField" && len(x.Args) == 2 {
				if lit, ok := x.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					key, _ := strconv.Unquote(lit.Value)
					checkKey(key, x.Args[1], x.Pos())
				}
			}
			// fmt.Print* of the CLI tools writes to the terminal of the operator, not to the log
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "fmt" {
				return true
			}
			if logMethods[sel.Sel.Name] {
				isF := strings.HasSuffix(sel.Sel.Name, "f")
				if isF && len(x.Args) >= 1 {
					lit, ok := x.Args[0].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}
					format, _ := strconv.Unquote(lit.Value)
					vs := verbs(format)
					for i, arg := range x.Args[1:] {
						if i >= len(vs) {
							break
						}
						if (vs[i] == 'v' || vs[i] == 's' || vs[i] == 'q' || vs[i] == 'x' || vs[i] == 'X') && isRisky(arg) {
							id := rel + "|fmt|" + exprString(fset, arg)
							if _, ok := allowed[id]; !ok {
								res = append(res, finding{fset.Position(arg.Pos()).String(), "format argument " + exprString(fset, arg) + " of " + format})
							}
						}
					}
				} else if !isF {
					for _, arg := range x.Args {
						if isRisky(arg) {
							id := rel + "|arg|" + exprString(fset, arg)
							if _, ok := allowed[id]; !ok {
								res = append(res, finding{fset.Position(arg.Pos()).String(), "argument " + exprString(fset, arg) + " of " + sel.Sel.Name})
							}
						}
					}
				}
			}
		}
		return true
	})

	return res
}

func Test_logCallsDoNotPassConversationText(t *testing.T) {
	root := filepath.Join("..", "..")
	fset := token.NewFileSet()
	var findings []finding
	scanned := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "logtest", "logpolicy", "eval":
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, "mock_") {
			return nil
		}
		file, errParse := parser.ParseFile(fset, path, nil, 0)
		if errParse != nil {
			return errParse
		}
		rel, _ := filepath.Rel(root, path)
		findings = append(findings, scanFile(fset, file, filepath.ToSlash(rel))...)
		scanned++
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if scanned < 50 {
		t.Fatalf("only %d files were scanned, the scan root is wrong", scanned)
	}

	for _, f := range findings {
		t.Errorf("%s: %s", f.pos, f.text)
	}
}

// The scan must detect what it is meant to detect. Each source below breaks the policy once.
func Test_scanDetects(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		expect int
	}{
		{"field map with a model", `package p; import "github.com/sirupsen/logrus"; func f(m *M){ _ = logrus.Fields{"request": m} }`, 1},
		{"with field", `package p; func f(l L, tmp *M){ l.WithField("message", tmp).Debug("x") }`, 1},
		{"format argument", `package p; func f(l L, m *M){ l.Debugf("a: %v", m.Data) }`, 1},
		{"format string with a number first", `package p; func f(l L, m *M){ l.Debugf("a: %d, b: %s", 1, m.Content) }`, 1},
		{"number verb passes", `package p; func f(l L, m *M){ l.Debugf("a: %d", m.Data) }`, 0},
		{"length passes", `package p; func f(l L, m *M){ l.WithField("data_len", len(m.Data)).Debugf("a: %d", len(m.Data)) }`, 0},
		{"id passes", `package p; func f(l L, m *M){ l.Debugf("a: %s", m.ID) }`, 0},
		{"text under another key", `package p; func f(l L, m *M){ l.WithField("renamed", m.Content).Debug("x") }`, 1},
		{"digit under another key", `package p; func f(l L, m *M){ l.WithField("pressed", m.Digit).Debug("x") }`, 1},
		{"arguments in a field map", `package p; import "github.com/sirupsen/logrus"; func f(m *M){ _ = logrus.Fields{"a": m.Arguments} }`, 1},
		{"text variable under another key", `package p; func f(l L, messageText string){ l.WithField("renamed", messageText).Debug("x") }`, 1},
		{"header", `package p; func f(l L, c *C){ l.WithField("h", c.Request.Header).Debug("x") }`, 1},
		{"get header", `package p; func f(l L, c *C){ l.WithField("auth", c.GetHeader("Authorization")).Debug("x") }`, 1},
		{"sprintf wrapper", `package p; import "fmt"; func f(l L, content string){ l.WithField("c", fmt.Sprintf("%s", content)).Debug("x") }`, 1},
		{"model in format", `package p; func f(l L, cc *C){ l.Debugf("a: %v", cc) }`, 1},
		{"pointer dereference", `package p; func f(l L, tmp *M){ l.Debugf("a: %+v", *tmp) }`, 1},
		{"slice of a text", `package p; func f(l L, answer string){ l.Debugf("a: %s", answer[:3]) }`, 1},
		{"index of a text", `package p; func f(l L, req *R){ l.Debugf("a: %v", req.Messages[0]) }`, 1},
		{"concatenation", `package p; func f(l L, title string){ l.Debug("title " + title) }`, 1},
		{"metadata", `package p; func f(l L, res *R){ l.WithField("a", res.Metadata).Debug("x") }`, 1},
		{"variables", `package p; func f(l L, tmp *R){ l.WithField("vars", tmp.Variables).Debug("x") }`, 1},
		{"team key", `package p; func f(l L, t *T){ l.WithField("team", t).Debug("x") }`, 1},
		{"hex of a text", `package p; func f(l L, answer string){ l.Debugf("a: %x", answer) }`, 1},
		{"println", `package p; func f(l L, content string){ l.Println(content) }`, 1},
		{"debugln", `package p; func f(l L, content string){ l.Debugln(content) }`, 1},
		{"printf", `package p; func f(l L, content string){ l.Printf("%s", content) }`, 1},
		{"set level in a service", `package p; import "github.com/sirupsen/logrus"; func f(){ logrus.SetLevel(logrus.DebugLevel) }`, 1},
		{"call without a format", `package p; func f(l L, m *M){ l.Debug(m) }`, 1},
		{"string conversion", `package p; func f(l L, body []byte){ l.Errorf("a: %s", string(body)) }`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "x.go", tt.src, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := scanFile(fset, file, "x.go")
			if len(got) != tt.expect {
				t.Errorf("expect %d findings, got %d: %v", tt.expect, len(got), got)
			}
		})
	}
}
