package config

import (
	"testing"

	"monorepo/bin-ai-manager/models/ai"
)

func Test_ConfigureLocalModels(t *testing.T) {
	old := globalConfig.LocalLLMModels
	t.Cleanup(func() { globalConfig.LocalLLMModels = old; ai.SetLocalModels(nil) })

	globalConfig.LocalLLMModels = `[{"endpoint":"a","id":"m","label":"M"}]`
	ConfigureLocalModels()
	if !ai.IsValidEngineModel("local.a.m") {
		t.Error("model must be registered")
	}

	globalConfig.LocalLLMModels = `[{`
	ConfigureLocalModels()
	if ai.IsValidEngineModel("local.a.m") {
		t.Error("a bad value must leave no local model")
	}

	globalConfig.LocalLLMModels = ""
	ConfigureLocalModels()
}
