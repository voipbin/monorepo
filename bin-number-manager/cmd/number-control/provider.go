package main

import (
	"context"
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-number-manager/internal/config"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

// openProviderHandler creates the handler of the provider commands. Tests replace it.
var openProviderHandler = initProviderHandler

// initProviderHandler opens the database only. The provider commands need neither the cache nor
// RabbitMQ, so they work when only the database is reachable.
func initProviderHandler() (providerhandler.ProviderHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "could not connect to the database")
	}

	return providerhandler.NewProviderHandler(nil, dbhandler.NewHandler(db, nil)), nil
}

// noPositionalArgs rejects every positional argument. The message never holds the argument: a
// secret typed after the command must not be echoed, and a bool flag written as "--enabled false"
// would otherwise leave "false" as a silently ignored argument.
func noPositionalArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errors.New("unexpected argument: use flags only (a bool flag is written --enabled=false)")
	}
	return nil
}

func cmdProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Number provider settings",
		Long: `Number provider settings (telnyx, twilio). These commands open the database; they need neither
Redis nor RabbitMQ. A change is used by the next call, with no restart. Do not give the database
address as an argument: set it in the environment of the process. The provider secret is read by the
service from an environment variable (telnyx: TELNYX_API_KEY, twilio: TWILIO_AUTH_TOKEN) and is not
set by these commands.`,
		Args: noPositionalArgs,
	}

	cmd.AddCommand(cmdProviderList())
	cmd.AddCommand(cmdProviderUpdate())

	return cmd
}

func cmdProviderList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the providers with their state (never a secret)",
		Args:  noPositionalArgs,
		RunE:  runProviderList,
	}
}

func cmdProviderUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Change the enabled flag, the priority or the config of a provider",
		Long: `Change the enabled flag, the priority or the non-secret config of a provider. Only the given
settings change. A config entry is key=value; an entry with an empty value (key=) removes the key.
Telnyx needs connection_id (profile_id is optional); Twilio needs sid and trunk_sid. The row is created
with the defaults when it does not exist. Write a bool flag with an equal sign: --enabled=false.`,
		Args: noPositionalArgs,
		RunE: runProviderUpdate,
	}

	flags := cmd.Flags()
	flags.String("name", "", "Provider name: telnyx or twilio (required)")
	flags.String("actor-id", "", "ID of the person who makes the change (required)")
	flags.Bool("enabled", false, "Enable or disable the provider (--enabled=false)")
	flags.Int("priority", 0, "Purchase priority, lower is tried first")
	flags.StringArray("config", nil, "Config entry key=value, repeatable (an empty value removes the key)")

	return cmd
}

// flagProviderName reads the provider name. The message never holds the given value.
func flagProviderName(cmd *cobra.Command) (number.ProviderName, error) {
	val, err := cmd.Flags().GetString("name")
	if err != nil {
		return "", err
	}
	if val == "" {
		return "", errors.New("name is required")
	}

	name := number.ProviderName(val)
	if name != number.ProviderNameTelnyx && name != number.ProviderNameTwilio {
		return "", errors.New("unknown provider name (telnyx or twilio)")
	}

	return name, nil
}

// flagActorID reads the actor. The message never holds the given value.
func flagActorID(cmd *cobra.Command) (uuid.UUID, error) {
	val, err := cmd.Flags().GetString("actor-id")
	if err != nil {
		return uuid.Nil, err
	}
	if val == "" {
		return uuid.Nil, errors.New("actor-id is required")
	}

	res := uuid.FromStringOrNil(val)
	if res == uuid.Nil {
		return uuid.Nil, errors.New("actor-id must be a valid UUID")
	}

	return res, nil
}

// parseConfigEntries turns the --config entries into a map. The messages never hold an entry.
func parseConfigEntries(entries []string) (map[string]string, error) {
	res := map[string]string{}
	for _, entry := range entries {
		k, v, ok := strings.Cut(entry, "=")
		if !ok || k == "" {
			return nil, errors.New("invalid --config entry: want key=value")
		}
		if _, dup := res[k]; dup {
			return nil, errors.New("duplicate --config key")
		}
		res[k] = v
	}

	return res, nil
}

func runProviderList(cmd *cobra.Command, args []string) error {
	h, err := openProviderHandler()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to initialize handlers")
	}

	res, err := h.List(context.Background())
	if err != nil {
		return pkgerrors.Wrap(err, "failed to list the providers")
	}

	return printJSON(res)
}

func runProviderUpdate(cmd *cobra.Command, args []string) error {
	name, err := flagProviderName(cmd)
	if err != nil {
		return err
	}
	actor, err := flagActorID(cmd)
	if err != nil {
		return err
	}

	flags := cmd.Flags()

	var enabled *bool
	if flags.Changed("enabled") {
		v, errGet := flags.GetBool("enabled")
		if errGet != nil {
			return errGet
		}
		enabled = &v
	}

	var priority *int
	if flags.Changed("priority") {
		v, errGet := flags.GetInt("priority")
		if errGet != nil {
			return errGet
		}
		priority = &v
	}

	entries, err := flags.GetStringArray("config")
	if err != nil {
		return err
	}
	conf, err := parseConfigEntries(entries)
	if err != nil {
		return err
	}

	if enabled == nil && priority == nil && len(conf) == 0 {
		return errors.New("at least one of --enabled, --priority and --config is required")
	}

	h, err := openProviderHandler()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to initialize handlers")
	}

	res, err := h.Update(context.Background(), name, actor, enabled, priority, conf)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to update the provider")
	}

	return printJSON(res)
}
