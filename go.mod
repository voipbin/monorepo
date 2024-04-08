module gitlab.com/voipbin/bin-manager/api-manager.git

go 1.22

toolchain go1.22.0

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.7.0 // indirect
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
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20240403030948-51eb7c33cf9a
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20240408033155-50f0cd082334
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20240403034140-ce82222fe7f4
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20240328055052-ec1c723aa183
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20240402051305-cf14186e380d
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20240330083852-ab008a2e3880
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20240405044227-febd49f8b700
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.20.0 // indirect
)

require (
	github.com/gorilla/websocket v1.5.1
	github.com/pebbe/zmq4 v1.2.11
	github.com/pion/rtp v1.8.5
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.9.0
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20240328054741-55144017eccd
	gitlab.com/voipbin/bin-manager/billing-manager.git v0.0.0-20240408051040-600f0028fbab
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20240313031908-f098e3fb6f12
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20240313050741-a2ced5030a06
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20240313050825-1c666b883013
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20240329045829-45dc5f4e4e76
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20231117134833-7918f76572d4
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20240408042746-c45b2b5aa984
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20240328053936-9008e28c2268
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20240313064601-888fe8578646
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20240402021210-adac880b81da
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20240313065038-1498b922bb24
	gitlab.com/voipbin/bin-manager/tag-manager.git v0.0.0-20240313070856-7d3433af905d
	gitlab.com/voipbin/bin-manager/transfer-manager.git v0.0.0-20230419025515-44dea928ef34
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20240313071253-ebca1db1437c
	golang.org/x/exp v0.0.0-20240404231335-c0f41cb1a7a0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.11.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/inflect v0.21.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/runtime v0.28.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-swagger/go-swagger v0.30.5 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/jessevdk/go-flags v1.5.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.0 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.52.2 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.18.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/toqueteos/webbrowser v1.2.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/urfave/cli/v2 v2.27.1 // indirect
	github.com/xrash/smetrics v0.0.0-20240312152122-5f08fbb34913 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20240313052650-d3e4c79af4c0 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20240313070648-addf67d64996 // indirect
	go.mongodb.org/mongo-driver v1.14.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.7.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
