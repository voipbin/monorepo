module gitlab.com/voipbin/bin-manager/webhook-manager.git

go 1.17

require (
	github.com/go-redis/redis/v8 v8.11.4
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/common v0.32.1 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20220224004013-7f378ac0e217
	golang.org/x/sys v0.0.0-20220224120231-95c6836cb0e7 // indirect
	google.golang.org/genproto v0.0.0-20220222213610-43724f9ea8cf // indirect
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220203044108-296babaad5c9
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/streadway/amqp v1.0.0 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.2.1 // indirect
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220214010042-9d735c473204 // indirect
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220224052357-6eefe99bd198 // indirect
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220214020613-3467a95bad81 // indirect
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220224051508-e0cda3ac4ec5 // indirect
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220208185140-cfc6013ffd01 // indirect
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220214021001-a55b20ebbcdc // indirect
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220209071252-a365e400801c // indirect
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20211221033732-78e26f6c23a6 // indirect
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220206205721-6f56cc4c3c1e // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20220220065756-9f1522273672 // indirect
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
