module gitlab.com/voipbin/bin-manager/api-manager.git

go 1.20

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gin-contrib/cors v1.5.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-playground/validator/v10 v10.16.0 // indirect
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.7.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/prometheus/client_golang v1.17.0
	github.com/sirupsen/logrus v1.9.3
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/swag v1.16.2
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20231130142821-0d377dc837c5
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20231125074622-ad630d730b13
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20230728174519-d347587ee005
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20230728171820-cdbeafa2cf1c
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20231005165516-1c0255c49028
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20230108123249-9f26b43b8ea9
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20230318110619-25a2d88d1450
	golang.org/x/crypto v0.16.0 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.16.0 // indirect
)

require (
	github.com/gorilla/websocket v1.5.1
	github.com/pebbe/zmq4 v1.2.10
	github.com/pkg/errors v0.9.1
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20231201025803-cbb8d2994a0c
	gitlab.com/voipbin/bin-manager/billing-manager.git v0.0.0-20230909145437-b3154bc05285
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20231129031206-5f69dfb5e088
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20220926023600-ef42f2bac093
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20230722152405-d36870c690da
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20230911172339-3bc1d3fa3578
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20231117134833-7918f76572d4
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20231119133841-1f299dcad73a
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20230705154830-1dd19ba6e804
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20230911175755-9140b8487519
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20230903182459-1d2f626b38fd
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20231015150838-40ee333c9936
	gitlab.com/voipbin/bin-manager/tag-manager.git v0.0.0-20230727164950-43d37418e642
	gitlab.com/voipbin/bin-manager/transfer-manager.git v0.0.0-20230419025515-44dea928ef34
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220723135740-c87a1ef4af61
	golang.org/x/exp v0.0.0-20231127185646-65229373498e
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.10.2 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.9 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20231130225603-8939d436e934 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20230716174942-b6a22e86ff12 // indirect
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0 // indirect
	golang.org/x/arch v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20231127180814-3a041ad873d4 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
