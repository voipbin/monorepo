package aihandler

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test_aiEventsPublishOnlyThroughHelper guards every ai event call site at
// once: any non-test source in this module that names an ai.EventType* constant
// must go through publishAIEvent, which strips the engine key. Reverting a call
// site to a raw PublishWebhookEvent publish fails here.
func Test_aiEventsPublishOnlyThroughHelper(t *testing.T) {
	root := filepath.Join("..", "..")
	helperFile := "publish_event.go"
	sites := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || d.Name() == helperFile {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(body), "\n") {
			if !strings.Contains(line, "ai.EventType") || strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if strings.Contains(line, "publishAIEvent(") {
				sites++
				continue
			}
			// The constant declarations live in models/ai.
			if strings.Contains(line, "=") && !strings.Contains(line, "(") {
				continue
			}
			t.Errorf("%s:%d names an ai event type outside publishAIEvent: %s", path, i+1, strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	if sites != 8 {
		t.Errorf("expected 8 publishAIEvent call sites, found %d", sites)
	}
}
