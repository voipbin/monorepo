package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-hook-manager/internal/config"
	"monorepo/bin-hook-manager/pkg/servicehandler"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameHookManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (servicehandler.ServiceHandler, error) {
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	return servicehandler.NewServiceHandler(reqHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "hook-control",
		Short: "Voipbin Hook Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}

			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bootstrap config"))
	}

	cmdRoot.AddCommand(cmdSendEmail())
	cmdRoot.AddCommand(cmdSendMessage())
	cmdRoot.AddCommand(cmdSendConversation())

	return cmdRoot
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

func cmdSendEmail() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-email",
		Short: "Send a test email webhook",
		RunE:  runSendEmail,
	}

	flags := cmd.Flags()
	flags.String("uri", "", "Webhook URI (required)")
	flags.String("data", "", "Webhook payload data as JSON string (required)")
	flags.String("file", "", "Read webhook payload from file path (alternative to --data)")

	return cmd
}

func runSendEmail(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	uri := viper.GetString("uri")
	if uri == "" {
		return fmt.Errorf("uri is required")
	}

	data, err := getPayloadData()
	if err != nil {
		return errors.Wrap(err, "failed to get payload data")
	}

	if err := handler.Email(context.Background(), uri, data); err != nil {
		return errors.Wrap(err, "failed to send email webhook")
	}

	response := map[string]string{
		"status":  "sent",
		"type":    "email",
		"uri":     uri,
		"payload": string(data),
	}

	return printJSON(response)
}

func cmdSendMessage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-message",
		Short: "Send a test message webhook",
		RunE:  runSendMessage,
	}

	flags := cmd.Flags()
	flags.String("uri", "", "Webhook URI (required)")
	flags.String("data", "", "Webhook payload data as JSON string (required)")
	flags.String("file", "", "Read webhook payload from file path (alternative to --data)")

	return cmd
}

func runSendMessage(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	uri := viper.GetString("uri")
	if uri == "" {
		return fmt.Errorf("uri is required")
	}

	data, err := getPayloadData()
	if err != nil {
		return errors.Wrap(err, "failed to get payload data")
	}

	if err := handler.Message(context.Background(), uri, data); err != nil {
		return errors.Wrap(err, "failed to send message webhook")
	}

	response := map[string]string{
		"status":  "sent",
		"type":    "message",
		"uri":     uri,
		"payload": string(data),
	}

	return printJSON(response)
}

func cmdSendConversation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-conversation",
		Short: "Send a test conversation webhook",
		RunE:  runSendConversation,
	}

	flags := cmd.Flags()
	flags.String("uri", "", "Webhook URI (required)")
	flags.String("data", "", "Webhook payload data as JSON string (required)")
	flags.String("file", "", "Read webhook payload from file path (alternative to --data)")

	return cmd
}

func runSendConversation(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	uri := viper.GetString("uri")
	if uri == "" {
		return fmt.Errorf("uri is required")
	}

	data, err := getPayloadData()
	if err != nil {
		return errors.Wrap(err, "failed to get payload data")
	}

	if err := handler.Conversation(context.Background(), uri, data); err != nil {
		return errors.Wrap(err, "failed to send conversation webhook")
	}

	response := map[string]string{
		"status":  "sent",
		"type":    "conversation",
		"uri":     uri,
		"payload": string(data),
	}

	return printJSON(response)
}

// getPayloadData retrieves payload data from either --data flag or --file flag
func getPayloadData() ([]byte, error) {
	dataStr := viper.GetString("data")
	filePath := viper.GetString("file")

	if dataStr == "" && filePath == "" {
		return nil, fmt.Errorf("either --data or --file is required")
	}

	if dataStr != "" && filePath != "" {
		return nil, fmt.Errorf("cannot specify both --data and --file")
	}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file: %s", filePath)
		}
		return data, nil
	}

	return []byte(dataStr), nil
}
