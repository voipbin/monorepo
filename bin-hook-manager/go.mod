module monorepo/bin-hook-manager

go 1.24.2

replace monorepo/bin-agent-manager => ../bin-agent-manager

replace monorepo/bin-billing-manager => ../bin-billing-manager

replace monorepo/bin-call-manager => ../bin-call-manager

replace monorepo/bin-common-handler => ../bin-common-handler

replace monorepo/bin-campaign-manager => ../bin-campaign-manager

replace monorepo/bin-chat-manager => ../bin-chat-manager

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

replace monorepo/bin-queue-manager => ../bin-queue-manager

replace monorepo/bin-registrar-manager => ../bin-registrar-manager

replace monorepo/bin-route-manager => ../bin-route-manager

replace monorepo/bin-storage-manager => ../bin-storage-manager

replace monorepo/bin-tag-manager => ../bin-tag-manager

replace monorepo/bin-transcribe-manager => ../bin-transcribe-manager

replace monorepo/bin-transfer-manager => ../bin-transfer-manager

replace monorepo/bin-tts-manager => ../bin-tts-manager

replace monorepo/bin-webhook-manager => ../bin-webhook-manager

require (
	github.com/gin-contrib/cors v1.7.5
	github.com/gin-gonic/gin v1.10.0
	github.com/go-playground/validator/v10 v10.26.0 // indirect
	github.com/go-sql-driver/mysql v1.9.2
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/prometheus/client_golang v1.22.0
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/mock v0.5.1
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	monorepo/bin-call-manager v0.0.0-20240403030948-51eb7c33cf9a // indirect
	monorepo/bin-common-handler v0.0.0-20240408033155-50f0cd082334
	monorepo/bin-flow-manager v0.0.0-20240403034140-ce82222fe7f4 // indirect
	monorepo/bin-number-manager v0.0.0-20240328055052-ec1c723aa183 // indirect
	monorepo/bin-registrar-manager v0.0.0-20240402051305-cf14186e380d // indirect
	monorepo/bin-storage-manager v0.0.0-20240330083852-ab008a2e3880 // indirect
	monorepo/bin-transcribe-manager v0.0.0-20240405044227-febd49f8b700 // indirect
)

require (
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.20.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.13.2 // indirect
	github.com/bytedance/sonic/loader v0.2.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.9 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.63.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.16.0 // indirect
	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0 // indirect
	golang.org/x/net v0.39.0 // indirect
	google.golang.org/genproto v0.0.0-20250425173222-7b384671a197 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	monorepo/bin-agent-manager v0.0.0-20240328054741-55144017eccd // indirect
	monorepo/bin-ai-manager v0.0.0-20240313050825-1c666b883013 // indirect
	monorepo/bin-billing-manager v0.0.0-20240408051040-600f0028fbab // indirect
	monorepo/bin-campaign-manager v0.0.0-20240313031908-f098e3fb6f12 // indirect
	monorepo/bin-chat-manager v0.0.0-20240313050741-a2ced5030a06 // indirect
	monorepo/bin-conference-manager v0.0.0-20240329045829-45dc5f4e4e76 // indirect
	monorepo/bin-conversation-manager v0.0.0-20231117134833-7918f76572d4 // indirect
	monorepo/bin-customer-manager v0.0.0-20240408042746-c45b2b5aa984 // indirect
	monorepo/bin-email-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-message-manager v0.0.0-20240328053936-9008e28c2268 // indirect
	monorepo/bin-outdial-manager v0.0.0-20240313064601-888fe8578646 // indirect
	monorepo/bin-queue-manager v0.0.0-20240402021210-adac880b81da // indirect
	monorepo/bin-route-manager v0.0.0-20240313065038-1498b922bb24 // indirect
	monorepo/bin-tag-manager v0.0.0-20240313070856-7d3433af905d // indirect
	monorepo/bin-transfer-manager v0.0.0-20230419025515-44dea928ef34 // indirect
	monorepo/bin-tts-manager v0.0.0-20240313070648-addf67d64996 // indirect
	monorepo/bin-webhook-manager v0.0.0-20240313071253-ebca1db1437c // indirect
)
