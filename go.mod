module gitlab.com/voipbin/bin-manager/api-manager.git

go 1.22

toolchain go1.22.0

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gin-contrib/cors v1.7.1
	github.com/gin-gonic/gin v1.9.1
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-playground/validator/v10 v10.19.0 // indirect
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/prometheus/client_golang v1.19.0
	github.com/sirupsen/logrus v1.9.3
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/swag v1.16.3
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20240328053508-a6603b09c7cd
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20240329081555-926615748334
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20240329004725-f5f463943d89
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20240328055052-ec1c723aa183
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20240313064944-17d68585dc25
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20240313065019-34db2d786767
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20240329081015-d32619da0dcd
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.19.0 // indirect
)

require (
	github.com/gorilla/websocket v1.5.1
	github.com/pebbe/zmq4 v1.2.11
	github.com/pion/rtp v1.8.5
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.9.0
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20240328054741-55144017eccd
	gitlab.com/voipbin/bin-manager/billing-manager.git v0.0.0-20240313031507-379b1a425709
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20240313031908-f098e3fb6f12
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20240313050741-a2ced5030a06
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20240313050825-1c666b883013
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20240329045829-45dc5f4e4e76
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20231117134833-7918f76572d4
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20240328013407-1d8264a6d809
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20240328053936-9008e28c2268
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20240313064601-888fe8578646
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20240313054055-78ecba56f1bc
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20240313065038-1498b922bb24
	gitlab.com/voipbin/bin-manager/tag-manager.git v0.0.0-20240313070856-7d3433af905d
	gitlab.com/voipbin/bin-manager/transfer-manager.git v0.0.0-20230419025515-44dea928ef34
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20240313071253-ebca1db1437c
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.11.3 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
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
	github.com/pelletier/go-toml/v2 v2.2.0 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/common v0.51.1 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20240313052650-d3e4c79af4c0 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20240313070648-addf67d64996 // indirect
	golang.org/x/arch v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20240325203815-454cdb8f5daa // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
