package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"monorepo/bin-common-handler/pkg/confighandler"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/internal/emailconfig"
	"monorepo/bin-email-manager/pkg/dbhandler"
)

// historySourceCLI is the source recorded in the history rows of the changes made here.
const historySourceCLI = "cli"

// maxStdinBytes bounds what is read from stdin. The longest accepted secret is shorter, so one
// more byte than the longest value tells that the input is too long.
const maxStdinBytes = 4096

// configOut is where the config commands print. Tests replace it.
var configOut io.Writer = os.Stdout

// configIn is where the secret value is read from. Tests replace it.
var configIn io.Reader = os.Stdin

// openConfigHandler creates the handler of the config commands. Tests replace it.
var openConfigHandler = initConfigHandler

// initConfigHandler opens the database and the cipher only. The config commands need neither
// the cache nor RabbitMQ, so they work when only the database is reachable.
func initConfigHandler() (confighandler.Handler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	return newConfigHandler(db)
}

func newConfigHandler(db *sql.DB) (confighandler.Handler, error) {
	return emailconfig.New(config.Get(), dbhandler.NewConfigStore(db), confighandler.WithSource(historySourceCLI))
}

func cmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Stored configuration (provider credentials)",
		Long: `Stored configuration of the provider credentials.

A stored value takes precedence over the environment variable of the same key. A change is
used by the next call, with no restart. The daemon applies the table migration at its start; on a
local database run "migrate up" first. The master key list is read from the environment
(EMAIL_CONFIG_ENCRYPTION_KEYS) and is never a flag.`,
	}

	cmd.AddCommand(cmdConfigList())
	cmd.AddCommand(cmdConfigSet())
	cmd.AddCommand(cmdConfigClear())

	return cmd
}

func cmdConfigList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the keys with their source (never a value)",
		RunE:  runConfigList,
	}
}

func cmdConfigSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Store the value of a key; the value is read from stdin",
		Long: `Store the value of a key. The value is read from stdin, never from an argument, so it does
not appear in the process list or in the shell history. Pipe the value in; typing it at a terminal
(end with Ctrl-D) shows it on the screen. One trailing newline is removed; a value with more than
one line is refused.`,
		RunE: runConfigSet,
	}

	flags := cmd.Flags()
	flags.String("key", "", "Config key (required)")
	flags.String("actor-id", "", "ID of the person who makes the change (required)")

	return cmd
}

func cmdConfigClear() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove the stored value of a key (the environment variable applies again)",
		RunE:  runConfigClear,
	}

	flags := cmd.Flags()
	flags.String("key", "", "Config key (required)")
	flags.String("actor-id", "", "ID of the person who makes the change (required)")

	return cmd
}

func flagUUID(cmd *cobra.Command, name, label string) (uuid.UUID, error) {
	val, err := cmd.Flags().GetString(name)
	if err != nil {
		return uuid.Nil, err
	}
	if val == "" {
		return uuid.Nil, fmt.Errorf("%s is required", label)
	}

	res := uuid.FromStringOrNil(val)
	if res == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid format for %s: '%s' is not a valid UUID", label, val)
	}

	return res, nil
}

func flagKey(cmd *cobra.Command) (string, error) {
	val, err := cmd.Flags().GetString("key")
	if err != nil {
		return "", err
	}
	if val == "" {
		return "", errors.New("key is required")
	}

	return val, nil
}

// readValue reads the value from the stream and removes one trailing line break.
func readValue(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxStdinBytes+1))
	if err != nil {
		return "", errors.Wrap(err, "could not read the value from stdin")
	}
	if len(data) > maxStdinBytes {
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

type configRowOutput struct {
	Key         string  `json:"key"`
	Kind        string  `json:"kind"`
	Type        string  `json:"type"`
	Source      string  `json:"source"`
	HasSecret   bool    `json:"has_secret"`
	EnvShadowed bool    `json:"env_shadowed"`
	KeyVersion  int     `json:"key_version,omitempty"`
	Revision    uint32  `json:"revision,omitempty"`
	UpdatedBy   *string `json:"updated_by,omitempty"`
}

func toOutput(r *confighandler.Resolved) configRowOutput {
	res := configRowOutput{
		Key:         r.Key,
		Kind:        string(r.Kind),
		Type:        string(r.Type),
		Source:      string(r.Source),
		HasSecret:   r.HasSecret,
		EnvShadowed: r.EnvShadowed,
		KeyVersion:  r.KeyVersion,
		Revision:    r.Revision,
	}
	if r.UpdatedBy != nil {
		s := r.UpdatedBy.String()
		res.UpdatedBy = &s
	}

	return res
}

func printConfig(v any) error {
	return printJSONTo(configOut, v)
}

func runConfigList(cmd *cobra.Command, args []string) error {
	h, err := openConfigHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	list, err := h.Describe(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to list the config keys")
	}

	res := make([]configRowOutput, 0, len(list))
	for _, r := range list {
		res = append(res, toOutput(r))
	}

	return printConfig(res)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, err := flagKey(cmd)
	if err != nil {
		return err
	}
	actor, err := flagUUID(cmd, "actor-id", "Actor ID")
	if err != nil {
		return err
	}

	h, err := openConfigHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	return setConfig(h, key, actor)
}

func setConfig(h confighandler.Handler, key string, actor uuid.UUID) error {
	k, ok := h.Registry().Lookup(key)
	if !ok {
		return fmt.Errorf("unknown config key: %s", key)
	}

	value, err := readValue(configIn)
	if err != nil {
		return err
	}

	var res *confighandler.Resolved
	if k.Kind == confighandler.KindSecret {
		res, err = h.SetSecret(context.Background(), actor, key, value)
	} else {
		res, err = h.Set(context.Background(), actor, key, value)
	}
	if err != nil {
		// the error never holds the value
		return errors.Wrap(err, "failed to store the value")
	}

	out := toOutput(res)
	return printConfig(&out)
}

func runConfigClear(cmd *cobra.Command, args []string) error {
	key, err := flagKey(cmd)
	if err != nil {
		return err
	}
	actor, err := flagUUID(cmd, "actor-id", "Actor ID")
	if err != nil {
		return err
	}

	h, err := openConfigHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := h.Clear(context.Background(), actor, key)
	if err != nil {
		return errors.Wrap(err, "failed to clear the value")
	}

	out := toOutput(res)
	return printConfig(&out)
}
