package config

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-pipecat-manager/internal/logtest"
)

func Test_ConfigureLocalModelsSkipsInvalidEntries(t *testing.T) {
	viper.Reset()
	t.Cleanup(func() { viper.Reset(); amai.SetLocalModels(nil) })

	// the second entry has an upper case endpoint, which the parser skips
	t.Setenv("LOCAL_LLM_MODELS", `[{"endpoint":"a","id":"m","label":"M"},{"endpoint":"B","id":"n","label":"N"}]`)
	if err := bindConfig(&cobra.Command{}); err != nil {
		t.Fatal(err)
	}

	logs := logtest.New(t)
	ConfigureLocalModels()

	if !amai.IsValidEngineModel("local.a.m") {
		t.Error("the valid entry must be registered")
	}
	if amai.IsValidEngineModel("local.B.n") {
		t.Error("the invalid entry must not be registered")
	}
	if dump := logs.Dump(); !strings.Contains(dump, "Some local llm model entries were skipped.") {
		t.Errorf("Wrong log. expect: the skipped entries error, got: %q", dump)
	}
}
