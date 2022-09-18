module gitlab.com/voipbin/bin-manager/api-manager.git

go 1.18

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-gonic/gin v1.8.1
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-playground/validator/v10 v10.11.1 // indirect
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.3.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/prometheus/client_golang v1.13.0
	github.com/sirupsen/logrus v1.9.0
	github.com/swaggo/files v0.0.0-20220728132757-551d4a08d97a
	github.com/swaggo/gin-swagger v1.5.3
	github.com/swaggo/swag v1.8.6
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220917172109-24211b12153a
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20220917165454-26e09c92aad7
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220917174709-a6be8747debf
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220722163735-36c125e09c26
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220722165215-cbc86d274ceb
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20220413034054-1271ef0d98c3
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220722170308-9bee5d0fc6f2
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90 // indirect
	golang.org/x/net v0.0.0-20220909164309-bea034e7d591 // indirect
	golang.org/x/sys v0.0.0-20220915200043-7b5979e65e41 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.12 // indirect
)

require (
	github.com/gorilla/websocket v1.5.0
	github.com/pebbe/zmq4 v1.2.9
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220722043822-a8daf3858b87
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20220722044501-749018171b9b
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220807174531-0227483c904d
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20220722162017-92d117261a6a
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220616065935-caa48b9b0bb5
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20220722162946-5c63199f33dd
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220722164233-2ec431be7901
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220808015830-a4b254f18efb
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220723135740-c87a1ef4af61
	golang.org/x/exp v0.0.0-20220916125017-b168a2c6b86b
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/spec v0.20.7 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/streadway/amqp v1.0.0 // indirect
	github.com/ugorji/go/codec v1.2.7 // indirect
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20220916145027-abf17f1587b3 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20220910161114-850aece1ef56 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20220628160222-ca61e7d2d60b // indirect
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0 // indirect
	google.golang.org/genproto v0.0.0-20220916172020-2692e8806bfa // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
