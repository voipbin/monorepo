package config

import (
	"testing"

	"github.com/spf13/cobra"
)

// The Paddle credentials are read from the environment by billingconfig and are not part of the
// loaded configuration: no field and no command line flag exists for the API key.
func Test_noPaddleAPIKeyFlag(t *testing.T) {
	resetLoadState(t)

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if cmd.PersistentFlags().Lookup("paddle_api_key") != nil {
		t.Errorf("Wrong match. a flag paddle_api_key exists")
	}
}
