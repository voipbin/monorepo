package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-storage-manager/internal/config"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/cachehandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
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

const serviceName = commonoutline.ServiceNameStorageManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initStorageHandler() (storagehandler.StorageHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initHandler(db, cache)
}

func initAccountHandler() (accounthandler.AccountHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	dbHandler := dbhandler.NewHandler(db, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameStorageEvent, serviceName)

	return accounthandler.NewAccountHandler(notifyHandler, dbHandler), nil
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (storagehandler.StorageHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameStorageEvent, serviceName)
	accountHandler := accounthandler.NewAccountHandler(notifyHandler, db)
	fileHandler := filehandler.NewFileHandler(
		notifyHandler,
		db,
		accountHandler,
		config.Get().GCPProjectID,
		config.Get().GCPBucketNameMedia,
		config.Get().GCPBucketNameTmp,
	)

	return storagehandler.NewStorageHandler(reqHandler, fileHandler, config.Get().GCPBucketNameMedia), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "storage-control",
		Short: "Voipbin Storage Management CLI",
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

	cmdFile := &cobra.Command{Use: "file", Short: "File operations"}
	cmdFile.AddCommand(cmdFileCreate())
	cmdFile.AddCommand(cmdFileGet())
	cmdFile.AddCommand(cmdFileList())
	cmdFile.AddCommand(cmdFileDelete())

	cmdAccount := &cobra.Command{Use: "account", Short: "Account operations"}
	cmdAccount.AddCommand(cmdAccountCreate())
	cmdAccount.AddCommand(cmdAccountGet())
	cmdAccount.AddCommand(cmdAccountList())
	cmdAccount.AddCommand(cmdAccountDelete())

	cmdRecording := &cobra.Command{Use: "recording", Short: "Recording operations"}
	cmdRecording.AddCommand(cmdRecordingGet())
	cmdRecording.AddCommand(cmdRecordingDelete())

	cmdRoot.AddCommand(cmdFile)
	cmdRoot.AddCommand(cmdAccount)
	cmdRoot.AddCommand(cmdRecording)
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

// File commands

func cmdFileCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new file record",
		RunE:  runFileCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("owner-id", "", "Owner ID (required)")
	flags.String("reference-type", "", "Reference type: normal, recording (required)")
	flags.String("reference-id", "", "Reference ID (required)")
	flags.String("name", "", "File name (required)")
	flags.String("detail", "", "File detail/description")
	flags.String("filename", "", "Original filename (required)")
	flags.String("bucket-name", "", "GCS bucket name (required)")
	flags.String("filepath", "", "File path in bucket (required)")

	return cmd
}

func runFileCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	ownerID, err := resolveUUID("owner-id", "Owner ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve owner ID")
	}

	referenceID, err := resolveUUID("reference-id", "Reference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve reference ID")
	}

	referenceType := viper.GetString("reference-type")
	if referenceType == "" {
		return fmt.Errorf("reference-type is required")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	filename := viper.GetString("filename")
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	bucketName := viper.GetString("bucket-name")
	if bucketName == "" {
		return fmt.Errorf("bucket-name is required")
	}

	filepath := viper.GetString("filepath")
	if filepath == "" {
		return fmt.Errorf("filepath is required")
	}

	handler, err := initStorageHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.FileCreate(
		context.Background(),
		customerID,
		ownerID,
		file.ReferenceType(referenceType),
		referenceID,
		name,
		viper.GetString("detail"),
		filename,
		bucketName,
		filepath,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}

	return printJSON(res)
}

func cmdFileGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a file by ID",
		RunE:  runFileGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "File ID (required)")

	return cmd
}

func runFileGet(cmd *cobra.Command, args []string) error {
	handler, err := initStorageHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	fileID, err := resolveUUID("id", "File ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve file ID")
	}

	res, err := handler.FileGet(context.Background(), fileID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve file")
	}

	return printJSON(res)
}

func cmdFileList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get file list",
		RunE:  runFileList,
	}

	flags := cmd.Flags()
	flags.Uint64("limit", 100, "Limit the number of files to retrieve")
	flags.String("token", "", "Retrieve files before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")
	flags.String("reference-id", "", "Reference ID to filter")
	flags.String("reference-type", "", "Reference type to filter (normal, recording)")

	return cmd
}

func runFileList(cmd *cobra.Command, args []string) error {
	handler, err := initStorageHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetUint64("limit")
	token := viper.GetString("token")

	filters := map[file.Field]any{
		file.FieldCustomerID: customerID,
		file.FieldDeleted:    false,
	}

	if referenceID := viper.GetString("reference-id"); referenceID != "" {
		refID := uuid.FromStringOrNil(referenceID)
		if refID != uuid.Nil {
			filters[file.FieldReferenceID] = refID
		}
	}

	if referenceType := viper.GetString("reference-type"); referenceType != "" {
		filters[file.FieldReferenceType] = file.ReferenceType(referenceType)
	}

	res, err := handler.FileList(context.Background(), token, limit, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve files")
	}

	return printJSON(res)
}

func cmdFileDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a file",
		RunE:  runFileDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "File ID (required)")

	return cmd
}

func runFileDelete(cmd *cobra.Command, args []string) error {
	handler, err := initStorageHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	fileID, err := resolveUUID("id", "File ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve file ID")
	}

	res, err := handler.FileDelete(context.Background(), fileID)
	if err != nil {
		return errors.Wrap(err, "failed to delete file")
	}

	return printJSON(res)
}

// Account commands

func cmdAccountCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new storage account",
		RunE:  runAccountCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")

	return cmd
}

func runAccountCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	handler, err := initAccountHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(context.Background(), customerID)
	if err != nil {
		return errors.Wrap(err, "failed to create account")
	}

	return printJSON(res)
}

func cmdAccountGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an account by ID",
		RunE:  runAccountGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")

	return cmd
}

func runAccountGet(cmd *cobra.Command, args []string) error {
	handler, err := initAccountHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	accountID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve account ID")
	}

	res, err := handler.Get(context.Background(), accountID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account")
	}

	return printJSON(res)
}

func cmdAccountList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get account list",
		RunE:  runAccountList,
	}

	flags := cmd.Flags()
	flags.Uint64("limit", 100, "Limit the number of accounts to retrieve")
	flags.String("token", "", "Retrieve accounts before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")

	return cmd
}

func runAccountList(cmd *cobra.Command, args []string) error {
	handler, err := initAccountHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetUint64("limit")
	token := viper.GetString("token")

	filters := map[account.Field]any{
		account.FieldCustomerID: customerID,
		account.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), token, limit, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve accounts")
	}

	return printJSON(res)
}

func cmdAccountDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an account",
		RunE:  runAccountDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")

	return cmd
}

func runAccountDelete(cmd *cobra.Command, args []string) error {
	handler, err := initAccountHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	accountID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve account ID")
	}

	res, err := handler.Delete(context.Background(), accountID)
	if err != nil {
		return errors.Wrap(err, "failed to delete account")
	}

	return printJSON(res)
}

// Recording commands

func cmdRecordingGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a recording (compressed zip) by reference ID",
		RunE:  runRecordingGet,
	}

	flags := cmd.Flags()
	flags.String("reference-id", "", "Reference ID (required)")

	return cmd
}

func runRecordingGet(cmd *cobra.Command, args []string) error {
	handler, err := initStorageHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	referenceID, err := resolveUUID("reference-id", "Reference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve reference ID")
	}

	res, err := handler.RecordingGet(context.Background(), referenceID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve recording")
	}

	return printJSON(res)
}

func cmdRecordingDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete all files for a recording by reference ID",
		RunE:  runRecordingDelete,
	}

	flags := cmd.Flags()
	flags.String("reference-id", "", "Reference ID (required)")

	return cmd
}

func runRecordingDelete(cmd *cobra.Command, args []string) error {
	handler, err := initStorageHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	referenceID, err := resolveUUID("reference-id", "Reference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve reference ID")
	}

	err = handler.RecordingDelete(context.Background(), referenceID)
	if err != nil {
		return errors.Wrap(err, "failed to delete recording")
	}

	fmt.Println("{\"status\": \"deleted\"}")
	return nil
}
