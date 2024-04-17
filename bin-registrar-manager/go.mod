module gitlab.com/voipbin/bin-manager/registrar-manager.git

go 1.22

toolchain go1.22.0

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.19.0
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/sirupsen/logrus v1.9.3
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20240402014926-8138de897c39
	golang.org/x/sys v0.18.0 // indirect
	google.golang.org/genproto v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

require (
	github.com/pkg/errors v0.9.1
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20240328013407-1d8264a6d809
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/common v0.51.1 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20240328054741-55144017eccd // indirect
	gitlab.com/voipbin/bin-manager/billing-manager.git v0.0.0-20240313031507-379b1a425709 // indirect
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20240401064615-0805c6621fbf // indirect
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20240313031908-f098e3fb6f12 // indirect
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20240313050741-a2ced5030a06 // indirect
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20240313050825-1c666b883013 // indirect
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20240329045829-45dc5f4e4e76 // indirect
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20231117134833-7918f76572d4 // indirect
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20240329004725-f5f463943d89 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20240313052650-d3e4c79af4c0 // indirect
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20240328053936-9008e28c2268 // indirect
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20240328055052-ec1c723aa183 // indirect
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20240313064601-888fe8578646 // indirect
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20240402021210-adac880b81da // indirect
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20240313065038-1498b922bb24 // indirect
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20240330083852-ab008a2e3880 // indirect
	gitlab.com/voipbin/bin-manager/tag-manager.git v0.0.0-20240313070856-7d3433af905d // indirect
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20240401073441-1c99bce2c04e // indirect
	gitlab.com/voipbin/bin-manager/transfer-manager.git v0.0.0-20230419025515-44dea928ef34 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20240313070648-addf67d64996 // indirect
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20240313071253-ebca1db1437c // indirect
)
