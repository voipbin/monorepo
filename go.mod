module gitlab.com/voipbin/bin-manager/flow-manager.git

go 1.18

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.13.0
	github.com/sirupsen/logrus v1.9.0
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220722043822-a8daf3858b87
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220726004520-28d857adb01e
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20220808030401-066b1c9af0a9
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220807174531-0227483c904d
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20220722162017-92d117261a6a
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20220722162946-5c63199f33dd
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220808015830-a4b254f18efb
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220722170308-9bee5d0fc6f2
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220723135740-c87a1ef4af61
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/streadway/amqp v1.0.0 // indirect
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20220722044501-749018171b9b // indirect
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220616065935-caa48b9b0bb5 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20220613130804-0755e1eb84d9 // indirect
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220722163735-36c125e09c26 // indirect
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220722164233-2ec431be7901 // indirect
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220722165215-cbc86d274ceb // indirect
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20220413034054-1271ef0d98c3 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20220628160222-ca61e7d2d60b // indirect
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0 // indirect
	golang.org/x/sys v0.0.0-20220808155132-1c4a2a72c664 // indirect
	google.golang.org/genproto v0.0.0-20220808204814-fd01256a5276 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
