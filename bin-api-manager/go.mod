module monorepo/bin-api-manager

go 1.22

toolchain go1.22.0

replace monorepo/bin-agent-manager => ../bin-agent-manager

replace monorepo/bin-call-manager => ../bin-call-manager

replace monorepo/bin-common-handler => ../bin-common-handler

replace monorepo/bin-billing-manager => ../bin-billing-manager

replace monorepo/bin-campaign-manager => ../bin-campaign-manager

replace monorepo/bin-chat-manager => ../bin-chat-manager

replace monorepo/bin-chatbot-manager => ../bin-chatbot-manager

replace monorepo/bin-conference-manager => ../bin-conference-manager

replace monorepo/bin-conversation-manager => ../bin-conversation-manager

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

replace monorepo/bin-customer-manager => ../bin-customer-manager

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gin-contrib/cors v1.7.2
	github.com/gin-gonic/gin v1.10.0
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/prometheus/client_golang v1.19.1
	github.com/sirupsen/logrus v1.9.3
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/swag v1.16.3
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/tools v0.21.0 // indirect
)

require (
	github.com/gorilla/websocket v1.5.1
	github.com/pebbe/zmq4 v1.2.11
	github.com/pion/rtp v1.8.6
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.9.0
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842
	monorepo/bin-agent-manager v0.0.0-20240328054741-55144017eccd
	monorepo/bin-billing-manager v0.0.0-20240408051040-600f0028fbab
	monorepo/bin-call-manager v0.0.0-20240403030948-51eb7c33cf9a
	monorepo/bin-campaign-manager v0.0.0-20240313031908-f098e3fb6f12
	monorepo/bin-chat-manager v0.0.0-20240313050741-a2ced5030a06
	monorepo/bin-chatbot-manager v0.0.0-20240313050825-1c666b883013
	monorepo/bin-common-handler v0.0.0-20240408033155-50f0cd082334
	monorepo/bin-conference-manager v0.0.0-20240329045829-45dc5f4e4e76
	monorepo/bin-conversation-manager v0.0.0-20231117134833-7918f76572d4
	monorepo/bin-customer-manager v0.0.0-20240408042746-c45b2b5aa984
	monorepo/bin-flow-manager v0.0.0-20240403034140-ce82222fe7f4
	monorepo/bin-message-manager v0.0.0-20240328053936-9008e28c2268
	monorepo/bin-number-manager v0.0.0-20240328055052-ec1c723aa183
	monorepo/bin-outdial-manager v0.0.0-20240313064601-888fe8578646
	monorepo/bin-queue-manager v0.0.0-20240402021210-adac880b81da
	monorepo/bin-registrar-manager v0.0.0-20240402051305-cf14186e380d
	monorepo/bin-route-manager v0.0.0-20240313065038-1498b922bb24
	monorepo/bin-storage-manager v0.0.0-20240330083852-ab008a2e3880
	monorepo/bin-tag-manager v0.0.0-20240313070856-7d3433af905d
	monorepo/bin-transcribe-manager v0.0.0-20240405044227-febd49f8b700
	monorepo/bin-transfer-manager v0.0.0-20230419025515-44dea928ef34
	monorepo/bin-webhook-manager v0.0.0-20240313071253-ebca1db1437c
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.53.0 // indirect
	github.com/prometheus/procfs v0.14.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20240509183442-62759503f434 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	monorepo/bin-hook-manager v0.0.0-20240313052650-d3e4c79af4c0 // indirect
	monorepo/bin-tts-manager v0.0.0-20240313070648-addf67d64996 // indirect
)
