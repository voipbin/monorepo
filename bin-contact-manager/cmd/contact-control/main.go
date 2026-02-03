package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
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

func initHandler() (contacthandler.ContactHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initContactHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initContactHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (contacthandler.ContactHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameContactEvent, serviceName, "")

	return contacthandler.NewContactHandler(reqHandler, db, notifyHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "contact-control",
		Short: "Voipbin Contact Management CLI",
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

	cmdContact := &cobra.Command{Use: "contact", Short: "Contact operations"}
	cmdContact.AddCommand(cmdCreate())
	cmdContact.AddCommand(cmdGet())
	cmdContact.AddCommand(cmdList())
	cmdContact.AddCommand(cmdUpdate())
	cmdContact.AddCommand(cmdDelete())
	cmdContact.AddCommand(cmdLookup())

	cmdRoot.AddCommand(cmdContact)
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

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new contact",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("first-name", "", "First name")
	flags.String("last-name", "", "Last name")
	flags.String("display-name", "", "Display name")
	flags.String("company", "", "Company")
	flags.String("job-title", "", "Job title")
	flags.String("source", "", "Source (manual, import, integration)")
	flags.String("external-id", "", "External ID")
	flags.String("notes", "", "Notes")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: customerID,
		},
		FirstName:   viper.GetString("first-name"),
		LastName:    viper.GetString("last-name"),
		DisplayName: viper.GetString("display-name"),
		Company:     viper.GetString("company"),
		JobTitle:    viper.GetString("job-title"),
		Source:      viper.GetString("source"),
		ExternalID:  viper.GetString("external-id"),
		Notes:       viper.GetString("notes"),
	}

	res, err := handler.Create(context.Background(), c)
	if err != nil {
		return errors.Wrap(err, "failed to create contact")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a contact by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Contact ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	contactID, err := resolveUUID("id", "Contact ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve contact ID")
	}

	res, err := handler.Get(context.Background(), contactID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve contact")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get contact list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of contacts to retrieve")
	flags.String("token", "", "Retrieve contacts before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[contact.Field]any{
		contact.FieldCustomerID: customerID,
		contact.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve contacts")
	}

	return printJSON(res)
}

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a contact",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Contact ID (required)")
	flags.String("first-name", "", "New first name")
	flags.String("last-name", "", "New last name")
	flags.String("display-name", "", "New display name")
	flags.String("company", "", "New company")
	flags.String("job-title", "", "New job title")
	flags.String("external-id", "", "New external ID")
	flags.String("notes", "", "New notes")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	contactID, err := resolveUUID("id", "Contact ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve contact ID")
	}

	fields := make(map[contact.Field]any)

	if cmd.Flags().Changed("first-name") {
		fields[contact.FieldFirstName] = viper.GetString("first-name")
	}
	if cmd.Flags().Changed("last-name") {
		fields[contact.FieldLastName] = viper.GetString("last-name")
	}
	if cmd.Flags().Changed("display-name") {
		fields[contact.FieldDisplayName] = viper.GetString("display-name")
	}
	if cmd.Flags().Changed("company") {
		fields[contact.FieldCompany] = viper.GetString("company")
	}
	if cmd.Flags().Changed("job-title") {
		fields[contact.FieldJobTitle] = viper.GetString("job-title")
	}
	if cmd.Flags().Changed("external-id") {
		fields[contact.FieldExternalID] = viper.GetString("external-id")
	}
	if cmd.Flags().Changed("notes") {
		fields[contact.FieldNotes] = viper.GetString("notes")
	}

	if len(fields) == 0 {
		return fmt.Errorf("at least one field must be specified for update")
	}

	res, err := handler.Update(context.Background(), contactID, fields)
	if err != nil {
		return errors.Wrap(err, "failed to update contact")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a contact",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Contact ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	contactID, err := resolveUUID("id", "Contact ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve contact ID")
	}

	res, err := handler.Delete(context.Background(), contactID)
	if err != nil {
		return errors.Wrap(err, "failed to delete contact")
	}

	return printJSON(res)
}

func cmdLookup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lookup",
		Short: "Lookup a contact by phone or email",
		RunE:  runLookup,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("phone-e164", "", "Phone number in E.164 format")
	flags.String("email", "", "Email address")

	return cmd
}

func runLookup(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	phoneE164 := viper.GetString("phone-e164")
	email := viper.GetString("email")

	if phoneE164 == "" && email == "" {
		return fmt.Errorf("either phone-e164 or email must be specified")
	}

	var res *contact.Contact
	if phoneE164 != "" {
		res, err = handler.LookupByPhone(context.Background(), customerID, phoneE164)
	} else {
		res, err = handler.LookupByEmail(context.Background(), customerID, email)
	}

	if err != nil {
		return errors.Wrap(err, "failed to lookup contact")
	}

	return printJSON(res)
}
