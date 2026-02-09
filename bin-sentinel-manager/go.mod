module monorepo/bin-sentinel-manager

go 1.25.3

replace monorepo/bin-agent-manager => ../bin-agent-manager

replace monorepo/bin-billing-manager => ../bin-billing-manager

replace monorepo/bin-call-manager => ../bin-call-manager

replace monorepo/bin-common-handler => ../bin-common-handler

replace monorepo/bin-contact-manager => ../bin-contact-manager

replace monorepo/bin-talk-manager => ../bin-talk-manager

replace monorepo/bin-campaign-manager => ../bin-campaign-manager

replace monorepo/bin-ai-manager => ../bin-ai-manager

replace monorepo/bin-conference-manager => ../bin-conference-manager

replace monorepo/bin-conversation-manager => ../bin-conversation-manager

replace monorepo/bin-customer-manager => ../bin-customer-manager

replace monorepo/bin-email-manager => ../bin-email-manager

replace monorepo/bin-flow-manager => ../bin-flow-manager

replace monorepo/bin-hook-manager => ../bin-hook-manager

replace monorepo/bin-message-manager => ../bin-message-manager

replace monorepo/bin-number-manager => ../bin-number-manager

replace monorepo/bin-outdial-manager => ../bin-outdial-manager

replace monorepo/bin-pipecat-manager => ../bin-pipecat-manager

replace monorepo/bin-queue-manager => ../bin-queue-manager

replace monorepo/bin-rag-manager => ../bin-rag-manager

replace monorepo/bin-registrar-manager => ../bin-registrar-manager

replace monorepo/bin-route-manager => ../bin-route-manager

replace monorepo/bin-storage-manager => ../bin-storage-manager

replace monorepo/bin-tag-manager => ../bin-tag-manager

replace monorepo/bin-timeline-manager => ../bin-timeline-manager

replace monorepo/bin-transcribe-manager => ../bin-transcribe-manager

replace monorepo/bin-transfer-manager => ../bin-transfer-manager

replace monorepo/bin-tts-manager => ../bin-tts-manager

replace monorepo/bin-webhook-manager => ../bin-webhook-manager

require (
	github.com/joonix/log v0.0.0-20251205082533-cd78070927ea
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.23.2
	github.com/sirupsen/logrus v1.9.4
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	go.uber.org/mock v0.6.0
	k8s.io/api v0.36.0-alpha.0
	k8s.io/apimachinery v0.36.0-alpha.0
	k8s.io/client-go v0.36.0-alpha.0
	monorepo/bin-common-handler v0.0.0-20240408033155-50f0cd082334
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/ClickHouse/ch-go v0.71.0 // indirect
	github.com/ClickHouse/clickhouse-go/v2 v2.43.0 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-openapi/jsonpointer v0.22.4 // indirect
	github.com/go-openapi/jsonreference v0.21.4 // indirect
	github.com/go-openapi/swag v0.25.4 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.4 // indirect
	github.com/go-openapi/swag/conv v0.25.4 // indirect
	github.com/go-openapi/swag/fileutils v0.25.4 // indirect
	github.com/go-openapi/swag/jsonname v0.25.4 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.4 // indirect
	github.com/go-openapi/swag/loading v0.25.4 // indirect
	github.com/go-openapi/swag/mangling v0.25.4 // indirect
	github.com/go-openapi/swag/netutils v0.25.4 // indirect
	github.com/go-openapi/swag/stringutils v0.25.4 // indirect
	github.com/go-openapi/swag/typeutils v0.25.4 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.4 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/paulmach/orb v0.12.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/sashabaranov/go-openai v1.41.2 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/term v0.39.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20251222181119-0a764e51fe1b // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20251125145642-4e65d59e963e // indirect
	k8s.io/utils v0.0.0-20260108192941-914a6e750570 // indirect
	monorepo/bin-agent-manager v0.0.0-20240328054741-55144017eccd // indirect
	monorepo/bin-ai-manager v0.0.0-20240313050825-1c666b883013 // indirect
	monorepo/bin-billing-manager v0.0.0-20240408051040-600f0028fbab // indirect
	monorepo/bin-call-manager v0.0.0-20240403030948-51eb7c33cf9a // indirect
	monorepo/bin-campaign-manager v0.0.0-20240313031908-f098e3fb6f12 // indirect
	monorepo/bin-conference-manager v0.0.0-20240329045829-45dc5f4e4e76 // indirect
	monorepo/bin-contact-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-conversation-manager v0.0.0-20231117134833-7918f76572d4 // indirect
	monorepo/bin-customer-manager v0.0.0-20240408042746-c45b2b5aa984 // indirect
	monorepo/bin-email-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-flow-manager v0.0.0-20240403034140-ce82222fe7f4 // indirect
	monorepo/bin-hook-manager v0.0.0-20240313052650-d3e4c79af4c0 // indirect
	monorepo/bin-message-manager v0.0.0-20240328053936-9008e28c2268 // indirect
	monorepo/bin-number-manager v0.0.0-20240328055052-ec1c723aa183 // indirect
	monorepo/bin-outdial-manager v0.0.0-20240313064601-888fe8578646 // indirect
	monorepo/bin-pipecat-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-queue-manager v0.0.0-20240402021210-adac880b81da // indirect
	monorepo/bin-rag-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-registrar-manager v0.0.0-20240402051305-cf14186e380d // indirect
	monorepo/bin-route-manager v0.0.0-20240313065038-1498b922bb24 // indirect
	monorepo/bin-storage-manager v0.0.0-20240330083852-ab008a2e3880 // indirect
	monorepo/bin-tag-manager v0.0.0-20240313070856-7d3433af905d // indirect
	monorepo/bin-talk-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-timeline-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-transcribe-manager v0.0.0-20240405044227-febd49f8b700 // indirect
	monorepo/bin-transfer-manager v0.0.0-20230419025515-44dea928ef34 // indirect
	monorepo/bin-tts-manager v0.0.0-20240313070648-addf67d64996 // indirect
	monorepo/bin-webhook-manager v0.0.0-20240313071253-ebca1db1437c // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.1 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
