package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-tts-manager/internal/config"
	"monorepo/bin-tts-manager/models/tts"
	"monorepo/bin-tts-manager/pkg/ttshandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = outline.ServiceNameTTSManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (ttshandler.TTSHandler, error) {
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, outline.QueueNameTTSEvent, serviceName)

	handler := ttshandler.NewTTSHandler(
		config.Get().AWSAccessKey,
		config.Get().AWSSecretKey,
		"/tmp/tts-media", // Default media bucket directory for CLI
		"localhost",      // Default local address for CLI
		reqHandler,
		notifyHandler,
	)

	if handler == nil {
		return nil, errors.New("failed to create TTS handler")
	}

	return handler, nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "tts-control",
		Short: "Voipbin TTS Management CLI",
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

	cmdSub := &cobra.Command{Use: "tts", Short: "TTS operation"}
	cmdSub.AddCommand(cmdCreate())

	cmdRoot.AddCommand(cmdSub)
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
		Short: "Create a new TTS audio file",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("call_id", "", "Call ID (required)")
	flags.String("text", "", "Text to synthesize (required)")
	flags.String("lang", "en-US", "Language code (default: en-US)")
	flags.String("gender", string(tts.GenderFemale), "Voice gender: male, female, neutral (default: female)")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	callID, err := resolveUUID("call_id", "Call ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve call ID")
	}

	text := viper.GetString("text")
	if text == "" {
		return fmt.Errorf("text is required")
	}

	lang := viper.GetString("lang")
	if lang == "" {
		lang = "en-US"
	}

	genderStr := viper.GetString("gender")
	gender := tts.Gender(genderStr)

	// Validate gender
	switch gender {
	case tts.GenderMale, tts.GenderFemale, tts.GenderNeutral:
		// Valid gender
	default:
		return fmt.Errorf("invalid gender: %s (must be male, female, or neutral)", genderStr)
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		callID,
		text,
		lang,
		gender,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create TTS")
	}

	return printJSON(res)
}
