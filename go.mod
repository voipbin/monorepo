module gitlab.com/voipbin/bin-manager/transcribe-manager.git

go 1.22

toolchain go1.22.0

require (
	cloud.google.com/go v0.112.2 // indirect
	cloud.google.com/go/speech v1.23.0
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pion/rtp v1.8.5
	github.com/prometheus/client_golang v1.19.0
	github.com/prometheus/common v0.52.2 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/sirupsen/logrus v1.9.3
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20240403030948-51eb7c33cf9a
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20240402062712-7120d40ad766
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20240330083852-ab008a2e3880 // indirect
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20240313071253-ebca1db1437c // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/oauth2 v0.19.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0
	google.golang.org/api v0.172.0
	google.golang.org/genproto v0.0.0-20240401170217-c3f982113cda // indirect
)

require google.golang.org/protobuf v1.33.0 // indirect

require (
	github.com/pkg/errors v0.9.1
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20240328013407-1d8264a6d809
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20240403034140-ce82222fe7f4
)

require (
	cloud.google.com/go/compute v1.25.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/longrunning v0.5.6 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dongri/phonenumber v0.1.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.3 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.2.1 // indirect
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20240328054741-55144017eccd // indirect
	gitlab.com/voipbin/bin-manager/billing-manager.git v0.0.0-20240313031507-379b1a425709 // indirect
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20240313031908-f098e3fb6f12 // indirect
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20240313050741-a2ced5030a06 // indirect
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20240313050825-1c666b883013 // indirect
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20240329045829-45dc5f4e4e76 // indirect
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20231117134833-7918f76572d4 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20240313052650-d3e4c79af4c0 // indirect
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20240328053936-9008e28c2268 // indirect
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20240328055052-ec1c723aa183 // indirect
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20240313064601-888fe8578646 // indirect
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20240402021210-adac880b81da // indirect
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20240402051305-cf14186e380d // indirect
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20240313065038-1498b922bb24 // indirect
	gitlab.com/voipbin/bin-manager/tag-manager.git v0.0.0-20240313070856-7d3433af905d // indirect
	gitlab.com/voipbin/bin-manager/transfer-manager.git v0.0.0-20230419025515-44dea928ef34 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20240313070648-addf67d64996 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/grpc v1.63.0 // indirect
)
