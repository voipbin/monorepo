package config

import (
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	amai "monorepo/bin-ai-manager/models/ai"
)

func Test_ConfigureLocalModelsSkipsInvalidEntries(t *testing.T) {
	viper.Reset()
	t.Cleanup(func() { viper.Reset(); amai.SetLocalModels(nil) })

	// the second entry has an upper case endpoint, which the parser skips
	t.Setenv("LOCAL_LLM_MODELS", `[{"endpoint":"a","id":"m","label":"M"},{"endpoint":"B","id":"n","label":"N"}]`)
	if err := bindConfig(&cobra.Command{}); err != nil {
		t.Fatal(err)
	}

	hook := logrustest.NewLocal(logrus.StandardLogger())
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })
	ConfigureLocalModels()

	if !amai.IsValidEngineModel("local.a.m") {
		t.Error("the valid entry must be registered")
	}
	if amai.IsValidEngineModel("local.B.n") {
		t.Error("the invalid entry must not be registered")
	}

	// skipped entries are reported at the error level, once.
	var errors []string
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel {
			errors = append(errors, e.Message)
		}
	}
	if len(errors) != 1 || errors[0] != "Some local llm model entries were skipped." {
		t.Errorf("Wrong log. expect: one skipped entries error, got: %v", errors)
	}
}
