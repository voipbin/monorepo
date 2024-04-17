module gitlab.com/voipbin/bin-manager/hook-manager.git

go 1.22

toolchain go1.22.0

require (
	github.com/gin-contrib/cors v1.7.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-playground/validator/v10 v10.19.0 // indirect
	github.com/go-sql-driver/mysql v1.8.0
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/prometheus/client_golang v1.19.0 // indirect
	github.com/sirupsen/logrus v1.9.3
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20240311045749-400f92951b6e // indirect
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20240313031844-5b54f8cea815
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20240212145953-364923003e09 // indirect
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20240214182731-ed12d5a9070a // indirect
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20240219100316-af55ebba9ea0 // indirect
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20240223190113-d0886960565c // indirect
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20230318110619-25a2d88d1450 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.11.3 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/common v0.50.0 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20240313015159-9fb73d56b217 // indirect
	gitlab.com/voipbin/bin-manager/billing-manager.git v0.0.0-20240313031507-379b1a425709 // indirect
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20240313031908-f098e3fb6f12 // indirect
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20240305053808-9c86ee04f005 // indirect
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20240116170346-73b2d21c2bd4 // indirect
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20240110173312-de5436ad2531 // indirect
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20231117134833-7918f76572d4 // indirect
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20240215075724-ff0374d05dc9 // indirect
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20230705154830-1dd19ba6e804 // indirect
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20230911175755-9140b8487519 // indirect
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20240227083257-1d68ea39943c // indirect
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20231015150838-40ee333c9936 // indirect
	gitlab.com/voipbin/bin-manager/tag-manager.git v0.0.0-20230727164950-43d37418e642 // indirect
	gitlab.com/voipbin/bin-manager/transfer-manager.git v0.0.0-20230419025515-44dea928ef34 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20230716174942-b6a22e86ff12 // indirect
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0 // indirect
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220723135740-c87a1ef4af61 // indirect
	golang.org/x/arch v0.7.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	google.golang.org/genproto v0.0.0-20240311173647-c811ad7063a7 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
