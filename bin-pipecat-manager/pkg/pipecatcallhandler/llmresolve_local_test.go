package pipecatcallhandler

import (
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
)

// These tests change the global local registry; they do not use t.Parallel().

func Test_resolveSessionLLMLocal(t *testing.T) {
	t.Cleanup(func() { amai.SetLocalModels(nil) })
	amai.SetLocalModels(nil)

	if err := classifySessionLLM("local.a.qwen2.5-7b"); err == nil {
		t.Fatal("an unconfigured local model must be rejected")
	}
	if _, _, err := resolveSessionLLM("local.a.qwen2.5-7b", ""); err == nil {
		t.Fatal("an unconfigured local model must be rejected")
	}

	if _, err := amai.ConfigureLocalModels(`[{"endpoint":"a","id":"qwen2.5-7b","label":"Qwen"}]`); err != nil {
		t.Fatal(err)
	}

	if err := classifySessionLLM("local.a.qwen2.5-7b"); err != nil {
		t.Errorf("classify = %v", err)
	}

	runnerType, key, err := resolveSessionLLM("local.a.qwen2.5-7b", "customer-key")
	if err != nil || runnerType != "local.a.qwen2.5-7b" || key != "" {
		t.Errorf("runnerType=%q key=%q err=%v", runnerType, key, err)
	}

	// the customer key is never forwarded, even when the live AI still holds one
	live := &amai.AI{EngineModel: "local.a.qwen2.5-7b"}
	if err := checkLiveEngineVendor("local.a.qwen2.5-7b", live, key); err != nil {
		t.Errorf("checkLiveEngineVendor = %v", err)
	}

	if _, _, err := resolveSessionLLM("local.a.other", ""); err == nil {
		t.Error("an unlisted local model must be rejected")
	}
}
