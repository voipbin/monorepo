package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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

// maxSecretStdinBytes bounds what is read from stdin. The handler accepts a shorter secret, so
// one more byte than the longest value tells that the input is too long.
const maxSecretStdinBytes = 4096

// providerIn is where the secret is read from. Tests replace it.
var providerIn io.Reader = os.Stdin

// openProviderHandler creates the handler of the provider commands. Tests replace it.
var openProviderHandler = initProviderHandler

// initProviderHandler opens the database only. The provider commands need neither the cache nor
// RabbitMQ, so they work when only the database is reachable.
func initProviderHandler() (providerhandler.ProviderHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "could not connect to the database")
	}

	return newProviderHandler(nil, dbhandler.NewHandler(db, nil))
}

// noPositionalArgs rejects every positional argument. The message never holds the argument: a
// secret typed after the command must not be echoed, and a bool flag written as "--enabled false"
// would otherwise leave "false" as a silently ignored argument.
func noPositionalArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errors.New("unexpected argument: use flags only (a bool flag is written --enabled=false, the secret is read from stdin)")
	}
	return nil
}

func cmdProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Number provider settings and secrets",
		Long: `Number provider settings and secrets (telnyx, twilio). These commands open the database and
read the key list from NUMBER_SECRET_ENCRYPTION_KEYS; they need neither Redis nor RabbitMQ. A change is
used by the next call, with no restart. Do not give the database address or the key list as an
argument: set them in the environment of the process. A malformed key list makes every provider
command fail; fix the key list and run it again.`,
		Args: noPositionalArgs,
	}

	cmd.AddCommand(cmdProviderList())
	cmd.AddCommand(cmdProviderUpdate())
	cmd.AddCommand(cmdProviderSecret())

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
Telnyx needs connection_id (profile_id is optional); Twilio needs sid and trunk_sid. The config of a
row is applied after its secret is stored: set the config first, then the secret. The row is created
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

func cmdProviderSecret() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Store or delete the secret of a provider",
		Args:  noPositionalArgs,
	}

	cmd.AddCommand(cmdProviderSecretSet())
	cmd.AddCommand(cmdProviderSecretDelete())

	return cmd
}

func cmdProviderSecretSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Store the secret of a provider; the secret is read from stdin",
		Long: `Store the secret (the API token) of a provider, encrypted under the newest key. The secret is
read from stdin and never from an argument, so it does not appear in the process list or in the shell
history. Pipe it in or redirect a file that only you can read; typing it by hand shows it on the
screen. In bash: read -rs TOKEN; printf '%s' "$TOKEN" | number-control provider secret set ...
(TOKEN is a shell variable, do not export it). When the command runs in a container, keep stdin
attached without a terminal (docker compose exec -T, kubectl exec -i). One trailing newline is
removed, surrounding spaces are removed by the handler, a value with more than one line is refused,
and the secret may be 1024 bytes at most.

The provider needs its required config first (Telnyx: connection_id; Twilio: sid and trunk_sid), see
"provider update". A stored secret is overwritten without being read. Do not change the account of a
provider that still has numbers (a different token or Twilio sid): releasing a number needs the
account it was bought on.

The service reads the provider secret from an environment variable (telnyx: TELNYX_API_KEY, twilio:
TWILIO_AUTH_TOKEN); the stored value is not used. Rotate the secret in the environment of the service.`,
		Args: noPositionalArgs,
		RunE: runProviderSecretSet,
	}

	flags := cmd.Flags()
	flags.String("name", "", "Provider name: telnyx or twilio (required)")
	flags.String("actor-id", "", "ID of the person who makes the change (required)")

	return cmd
}

func cmdProviderSecretDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete the stored secret of a provider",
		Long: `Delete the stored secret of a provider. The config of the row is kept. The credential_source in
the output tells which credential the provider uses afterwards. A provider that has no row yet is
reported as not found. Do not change the account of a provider that still has numbers. The service
reads the provider secret from an environment variable, not from the stored value, so the deletion
does not remove the credential of the service.`,
		Args: noPositionalArgs,
		RunE: runProviderSecretDelete,
	}

	flags := cmd.Flags()
	flags.String("name", "", "Provider name: telnyx or twilio (required)")
	flags.String("actor-id", "", "ID of the person who makes the change (required)")

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

// parseConfigEntries turns the --config entries into a map. The messages never hold an entry:
// an entry without "=" may be a pasted secret.
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

// readSecret reads the secret from the stream and removes one trailing line break. A character
// device (a terminal, /dev/null) is refused, so a secret is never typed at a prompt by mistake.
func readSecret(r io.Reader) (string, error) {
	if f, ok := r.(*os.File); ok {
		if st, err := f.Stat(); err == nil && st.Mode()&os.ModeCharDevice != 0 {
			return "", errors.New("stdin must be a pipe or a file")
		}
	}

	data, err := io.ReadAll(io.LimitReader(r, maxSecretStdinBytes+1))
	if err != nil {
		return "", pkgerrors.Wrap(err, "could not read the secret from stdin")
	}
	if len(data) > maxSecretStdinBytes {
		return "", errors.New("the value on stdin is too long")
	}

	val := string(data)
	val = strings.TrimSuffix(val, "\n")
	val = strings.TrimSuffix(val, "\r")

	// a credential is one line; a pasted multi-line text is a mistake that would surface only as an
	// authentication error at the provider.
	if strings.ContainsAny(val, "\r\n") {
		return "", errors.New("the value on stdin has more than one line")
	}

	return val, nil
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

func runProviderSecretSet(cmd *cobra.Command, args []string) error {
	name, err := flagProviderName(cmd)
	if err != nil {
		return err
	}
	actor, err := flagActorID(cmd)
	if err != nil {
		return err
	}

	// the input is read and checked before the database is opened
	secret, err := readSecret(providerIn)
	if err != nil {
		return err
	}

	h, err := openProviderHandler()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to initialize handlers")
	}

	// the error of the handler never holds the secret
	res, err := h.UpdateSecret(context.Background(), name, actor, secret)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to store the provider secret")
	}

	printSecretNotice(cmd, name)

	return printJSON(res)
}

func runProviderSecretDelete(cmd *cobra.Command, args []string) error {
	name, err := flagProviderName(cmd)
	if err != nil {
		return err
	}
	actor, err := flagActorID(cmd)
	if err != nil {
		return err
	}

	h, err := openProviderHandler()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to initialize handlers")
	}

	res, err := h.DeleteSecret(context.Background(), name, actor)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to delete the provider secret")
	}

	printSecretNotice(cmd, name)

	return printJSON(res)
}

// secretEnvNames is the environment variable that holds the secret of each provider.
var secretEnvNames = map[number.ProviderName]string{
	number.ProviderNameTelnyx: "TELNYX_API_KEY",
	number.ProviderNameTwilio: "TWILIO_AUTH_TOKEN",
}

// printSecretNotice tells the operator where the provider secret is read from: the stored value
// is not used, so a change of the stored secret does not change the credential of the service.
func printSecretNotice(cmd *cobra.Command, name number.ProviderName) {
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "The provider secret is read from the environment variable %s; the stored value is not used.\n", secretEnvNames[name])
}
