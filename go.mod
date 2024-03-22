module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.22

toolchain go1.22.0

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/sys v0.18.0 // indirect
)

require (
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.19.0
	github.com/rabbitmq/amqp091-go v1.9.0
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20240321060456-89e09b2001fc
	gitlab.com/voipbin/bin-manager/billing-manager.git v0.0.0-20240313031507-379b1a425709
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20240322090910-f9e04380bd37
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20240313031908-f098e3fb6f12
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20240313050741-a2ced5030a06
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20240313050825-1c666b883013
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20240313050849-b74c46e8ee3b
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20231117134833-7918f76572d4
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20240313054229-aa9ebb27bdf7
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20240313053651-4edf6033535b
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20240313052650-d3e4c79af4c0
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20240313054209-d19ab08b5a2c
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20240313054753-a427ebcc89b1
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20240313064601-888fe8578646
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20240313054055-78ecba56f1bc
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20240313064944-17d68585dc25
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20240313065038-1498b922bb24
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20240313065019-34db2d786767
	gitlab.com/voipbin/bin-manager/tag-manager.git v0.0.0-20240313070856-7d3433af905d
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20230318110619-25a2d88d1450
	gitlab.com/voipbin/bin-manager/transfer-manager.git v0.0.0-20230419025515-44dea928ef34
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20240313070648-addf67d64996
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20240313071253-ebca1db1437c
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/common v0.51.0 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
