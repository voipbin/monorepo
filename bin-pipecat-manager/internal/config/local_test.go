package config

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	amai "monorepo/bin-ai-manager/models/ai"
)

func Test_LocalLLMModelsFromEnv(t *testing.T) {
	viper.Reset()
	t.Cleanup(func() { viper.Reset(); amai.SetLocalModels(nil) })

	raw := `[{"endpoint":"a","id":"m","label":"M"}]`
	t.Setenv("LOCAL_LLM_MODELS", raw)
	if err := bindConfig(&cobra.Command{}); err != nil {
		t.Fatal(err)
	}

	if got := LocalLLMModels(); got != raw {
		t.Errorf("LocalLLMModels = %q", got)
	}

	ConfigureLocalModels()
	if !amai.IsValidEngineModel("local.a.m") {
		t.Error("model must be registered")
	}
}

func Test_ConfigureLocalModelsBadValueDoesNotStop(t *testing.T) {
	viper.Reset()
	t.Cleanup(func() { viper.Reset(); amai.SetLocalModels(nil) })

	t.Setenv("LOCAL_LLM_MODELS", `[{`)
	if err := bindConfig(&cobra.Command{}); err != nil {
		t.Fatal(err)
	}

	ConfigureLocalModels()
	if amai.IsValidEngineModel("local.a.m") {
		t.Error("no local model may be registered")
	}
}

// The models value is deliberately not a Config field, so a log of the whole Config can never include it.
func Test_ConfigStructHasNoLocalLLMField(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Name == "LocalLLMModels" {
			t.Error("Config must not carry LocalLLMModels")
		}
	}
}
