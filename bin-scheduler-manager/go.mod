module monorepo/bin-scheduler-manager

go 1.25.3

replace monorepo/bin-agent-manager => ../bin-agent-manager

replace monorepo/bin-billing-manager => ../bin-billing-manager

replace monorepo/bin-call-manager => ../bin-call-manager

replace monorepo/bin-common-handler => ../bin-common-handler

replace monorepo/bin-direct-manager => ../bin-direct-manager

replace monorepo/bin-contact-manager => ../bin-contact-manager

replace monorepo/bin-talk-manager => ../bin-talk-manager

replace monorepo/bin-campaign-manager => ../bin-campaign-manager

replace monorepo/bin-ai-manager => ../bin-ai-manager

replace monorepo/bin-conference-manager => ../bin-conference-manager

replace monorepo/bin-conversation-manager => ../bin-conversation-manager

replace monorepo/bin-customer-manager => ../bin-customer-manager

replace monorepo/bin-email-manager => ../bin-email-manager

replace monorepo/bin-flow-manager => ../bin-flow-manager

replace monorepo/bin-webchat-manager => ../bin-webchat-manager

replace monorepo/bin-hook-manager => ../bin-hook-manager

replace monorepo/bin-message-manager => ../bin-message-manager

replace monorepo/bin-number-manager => ../bin-number-manager

replace monorepo/bin-outdial-manager => ../bin-outdial-manager

replace monorepo/bin-pipecat-manager => ../bin-pipecat-manager

replace monorepo/bin-queue-manager => ../bin-queue-manager

replace monorepo/bin-rag-manager => ../bin-rag-manager

replace monorepo/bin-registrar-manager => ../bin-registrar-manager

replace monorepo/bin-route-manager => ../bin-route-manager

replace monorepo/bin-scheduler-manager => ../bin-scheduler-manager

replace monorepo/bin-storage-manager => ../bin-storage-manager

replace monorepo/bin-tag-manager => ../bin-tag-manager

replace monorepo/bin-timeline-manager => ../bin-timeline-manager

replace monorepo/bin-transcribe-manager => ../bin-transcribe-manager

replace monorepo/bin-transfer-manager => ../bin-transfer-manager

replace monorepo/bin-tts-manager => ../bin-tts-manager

replace monorepo/bin-webhook-manager => ../bin-webhook-manager

require (
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/joonix/log v0.0.0-20251205082533-cd78070927ea
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.9.4
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	monorepo/bin-common-handler v0.0.0-20240408033155-50f0cd082334
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto v0.0.0-20230216225411-c8e22ba71e44 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
