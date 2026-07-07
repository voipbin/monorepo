package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameContactManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (casehandler.CaseHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initCaseHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initCaseHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (casehandler.CaseHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameContactEvent, serviceName)

	return casehandler.NewCaseHandler(reqHandler, db, notifyHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "case-control",
		Short: "Voipbin Case Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}

			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bind infrastructure config"))
	}

	cmdCase := &cobra.Command{Use: "case", Short: "Case operations"}
	cmdCase.AddCommand(cmdReconcileContact())

	cmdRoot.AddCommand(cmdCase)
	return cmdRoot
}

func resolveUUID(flagName string, label string) (uuid.UUID, error) {
	val := viper.GetString(flagName)
	if val == "" {
		return uuid.Nil, fmt.Errorf("%s is required", label)
	}

	res := uuid.FromStringOrNil(val)
	if res == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid format for %s: '%s' is not a valid UUID", label, val)
	}

	return res, nil
}

// cmdReconcileContact implements design §3.4's recovery path: `case
// reconcile-contact <case_id | --all>` re-runs deriveCaseContactID and
// overwrites Case.contact_id. Idempotent -- used only if drift is
// discovered (e.g. a bulk import wrote Resolution rows without going
// through the handler). No scheduled reconciliation job at this stage.
func cmdReconcileContact() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reconcile-contact",
		Short: "Recompute Case.contact_id from the source-of-truth Resolution rows",
		RunE:  runReconcileContact,
	}

	flags := cmd.Flags()
	flags.String("case-id", "", "Case ID to reconcile (mutually exclusive with --all)")
	flags.Bool("all", false, "Reconcile every Case across all tenants")

	return cmd
}

func runReconcileContact(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	all := viper.GetBool("all")
	caseIDStr := viper.GetString("case-id")

	if all == (caseIDStr != "") {
		return fmt.Errorf("exactly one of --case-id or --all must be specified")
	}

	ctx := context.Background()

	if !all {
		caseID, err := resolveUUID("case-id", "Case ID")
		if err != nil {
			return errors.Wrap(err, "failed to resolve case ID")
		}
		if err := handler.ReconcileContact(ctx, caseID); err != nil {
			return errors.Wrap(err, "failed to reconcile case")
		}
		fmt.Printf("reconciled case %s\n", caseID)
		return nil
	}

	cases, err := handler.CaseListAll(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to list cases")
	}

	reconciled := 0
	for _, c := range cases {
		if err := handler.ReconcileContact(ctx, c.ID); err != nil {
			return errors.Wrapf(err, "failed to reconcile case %s", c.ID)
		}
		reconciled++
	}
	fmt.Printf("reconciled %d case(s)\n", reconciled)
	return nil
}
