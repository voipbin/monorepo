package main

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/pkg/errors"

	"monorepo/bin-registrar-manager/internal/config"
)

const serviceName = "registrar-control"

func main() {
	cmd := initCommand()
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "registrar-control",
		Short: "VoIPbin Registrar Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return errors.Wrap(err, "failed to bind flags")
			}
			return config.Bootstrap(cmd)
		},
	}

	// Extension subcommands
	cmdExtension := &cobra.Command{Use: "extension", Short: "Extension operations"}
	cmdRoot.AddCommand(cmdExtension)

	// Trunk subcommands
	cmdTrunk := &cobra.Command{Use: "trunk", Short: "Trunk operations"}
	cmdRoot.AddCommand(cmdTrunk)

	return cmdRoot
}
