module monorepo/bin-storage-manager

go 1.24.2

replace monorepo/bin-agent-manager => ../bin-agent-manager

replace monorepo/bin-billing-manager => ../bin-billing-manager

replace monorepo/bin-call-manager => ../bin-call-manager

replace monorepo/bin-common-handler => ../bin-common-handler

replace monorepo/bin-campaign-manager => ../bin-campaign-manager

replace monorepo/bin-chat-manager => ../bin-chat-manager

replace monorepo/bin-ai-manager => ../bin-ai-manager

replace monorepo/bin-conference-manager => ../bin-conference-manager

replace monorepo/bin-conversation-manager => ../bin-conversation-manager

replace monorepo/bin-customer-manager => ../bin-customer-manager

replace monorepo/bin-email-manager => ../bin-email-manager

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

require (
	cloud.google.com/go/storage v1.55.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/joonix/log v0.0.0-20230221083239-7988383bab32
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.65.0 // indirect
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/mock v0.5.2
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/oauth2 v0.30.0
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/api v0.240.0
	google.golang.org/genproto v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	monorepo/bin-call-manager v0.0.0-20240403030948-51eb7c33cf9a // indirect
	monorepo/bin-common-handler v0.0.0-20240408033155-50f0cd082334
)

require (
	github.com/go-redsync/redsync/v4 v4.13.0
	github.com/go-sql-driver/mysql v1.9.3
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.11.0
	github.com/smotes/purse v1.0.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.20.1
	monorepo/bin-customer-manager v0.0.0-20240408042746-c45b2b5aa984
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go v0.121.3 // indirect
	cloud.google.com/go/auth v0.16.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.29.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20250501225837-2ac532fd4443 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.37.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	monorepo/bin-agent-manager v0.0.0-20240328054741-55144017eccd // indirect
	monorepo/bin-ai-manager v0.0.0-20240313050825-1c666b883013 // indirect
	monorepo/bin-billing-manager v0.0.0-20240408051040-600f0028fbab // indirect
	monorepo/bin-campaign-manager v0.0.0-20240313031908-f098e3fb6f12 // indirect
	monorepo/bin-chat-manager v0.0.0-20240313050741-a2ced5030a06 // indirect
	monorepo/bin-conference-manager v0.0.0-20240329045829-45dc5f4e4e76 // indirect
	monorepo/bin-conversation-manager v0.0.0-20231117134833-7918f76572d4 // indirect
	monorepo/bin-email-manager v0.0.0-00010101000000-000000000000 // indirect
	monorepo/bin-flow-manager v0.0.0-20240403034140-ce82222fe7f4 // indirect
	monorepo/bin-hook-manager v0.0.0-20240313052650-d3e4c79af4c0 // indirect
	monorepo/bin-message-manager v0.0.0-20240328053936-9008e28c2268 // indirect
	monorepo/bin-number-manager v0.0.0-20240328055052-ec1c723aa183 // indirect
	monorepo/bin-outdial-manager v0.0.0-20240313064601-888fe8578646 // indirect
	monorepo/bin-queue-manager v0.0.0-20240402021210-adac880b81da // indirect
	monorepo/bin-registrar-manager v0.0.0-20240402051305-cf14186e380d // indirect
	monorepo/bin-route-manager v0.0.0-20240313065038-1498b922bb24 // indirect
	monorepo/bin-tag-manager v0.0.0-20240313070856-7d3433af905d // indirect
	monorepo/bin-transcribe-manager v0.0.0-20240405044227-febd49f8b700 // indirect
	monorepo/bin-transfer-manager v0.0.0-20230419025515-44dea928ef34 // indirect
	monorepo/bin-tts-manager v0.0.0-20240313070648-addf67d64996 // indirect
	monorepo/bin-webhook-manager v0.0.0-20240313071253-ebca1db1437c // indirect
)
