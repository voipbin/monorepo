package toolhandler

import (
	"strings"
	"testing"

	"monorepo/bin-ai-manager/models/tool"
)

// Test_sendMessageDefinitionMedias pins that the send_message tool accepts storage file ids only:
// an array of uuid strings and no field that could carry a url.
func Test_sendMessageDefinitionMedias(t *testing.T) {
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameSendMessage {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("send_message tool definition not found")
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("send_message parameters has no properties map")
	}
	medias, ok := props["medias"].(map[string]any)
	if !ok {
		t.Fatalf("send_message has no medias property")
	}
	// maxItems is only a hint for the LLM; the real limit is enforced by message-manager and is
	// the same number (at most 10 files). Keep both in step.
	if medias["maxItems"] != 10 {
		t.Errorf("medias maxItems must be 10. got: %v", medias["maxItems"])
	}
	if d, _ := medias["description"].(string); !strings.Contains(d, "MMS") || !strings.Contains(d, "Omit") {
		t.Errorf("medias description must say that it sends an MMS and that it can be omitted. got: %q", d)
	}
	if medias["type"] != "array" {
		t.Errorf("medias must be an array. got: %v", medias["type"])
	}
	items, ok := medias["items"].(map[string]any)
	if !ok || items["type"] != "string" || items["format"] != "uuid" {
		t.Errorf("medias items must be uuid strings. got: %v", medias["items"])
	}
	if d, _ := items["description"].(string); !strings.Contains(d, "Never a URL") {
		t.Errorf("medias items must say never a URL. got: %q", d)
	}
	if !strings.Contains(def.Description, "MMS") {
		t.Errorf("the send_message description must mention MMS")
	}
	if required, ok := def.Parameters["required"].([]string); ok {
		for _, r := range required {
			if r == "medias" {
				t.Errorf("medias must stay optional so that a plain sms keeps working")
			}
		}
	}
	for _, name := range []string{"url", "urls", "media_urls"} {
		if _, found := props[name]; found {
			t.Errorf("send_message must not accept %s", name)
		}
	}
}

// Test_sendMessageDefinitionSource pins that the source of send_message is optional and that the description tells the
// model to leave it out, because a source that is not a number of the account makes the send fail.
func Test_sendMessageDefinitionSource(t *testing.T) {
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameSendMessage {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("send_message tool definition not found")
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("send_message parameters has no properties map")
	}
	source, ok := props["source"].(map[string]any)
	if !ok {
		t.Fatalf("send_message has no source property")
	}
	if d, _ := source["description"].(string); !strings.Contains(d, "Optional") || !strings.Contains(d, "default outgoing number") {
		t.Errorf("source description must say that it is optional and that the default outgoing number is used. got: %q", d)
	}
	required, ok := def.Parameters["required"].([]string)
	if !ok {
		t.Fatalf("send_message parameters has no required list")
	}
	for _, r := range required {
		if r == "source" {
			t.Errorf("source must stay optional")
		}
	}
}
