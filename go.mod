module gitlab.com/voipbin/bin-manager/api-manager.git

go 1.18

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-gonic/gin v1.8.2
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-playground/validator/v10 v10.11.1 // indirect
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.7.0
	github.com/gofrs/uuid v4.3.1+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/prometheus/client_golang v1.14.0
	github.com/sirupsen/logrus v1.9.0
	github.com/swaggo/files v1.0.0
	github.com/swaggo/gin-swagger v1.5.3
	github.com/swaggo/swag v1.8.9
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20230121103234-0d96435ecdd3
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20230121111025-34a739cb7e28
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20230102093540-da6a0504a0ae
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20221206165111-75edd8be2cb9
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20221206185501-ee71428c9199
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20230108123249-9f26b43b8ea9
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20230120110509-eb4908c35288
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/text v0.6.0 // indirect
	golang.org/x/tools v0.5.0 // indirect
)

require (
	github.com/gorilla/websocket v1.5.0
	github.com/pebbe/zmq4 v1.2.9
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20221123170257-20a7f901c1fc
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20221207172609-98006dd828f9
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20220926023600-ef42f2bac093
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20230121114959-5eb900e19fcf
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20220722162017-92d117261a6a
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220616065935-caa48b9b0bb5
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20221216125212-e74e4c409422
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220722164233-2ec431be7901
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20221124180605-39b41a7a1ada
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20221029145057-ef1ebd21d097
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220723135740-c87a1ef4af61
	golang.org/x/exp v0.0.0-20230118134722-a68e582fa157
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.8 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/goccy/go-json v0.10.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/streadway/amqp v1.0.0 // indirect
	github.com/ugorji/go/codec v1.2.8 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20221211030023-7909940f4600 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20230117134054-5234eab5c94d // indirect
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0 // indirect
	google.golang.org/genproto v0.0.0-20230119192704-9d59e20e5cd1 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
