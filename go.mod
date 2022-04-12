module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.17

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v1.0.0
	golang.org/x/sys v0.0.0-20220412071739-889880a91fd5 // indirect
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/prometheus/client_golang v1.12.1
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220412120516-38c3ee4e32b7
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220412121149-1b04f3e8a36c
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220412121423-ed62e05a34bb
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220412120836-3caad2b4e7fb
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220412115545-9d379a3e42c5
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20220412122943-0ecb90b64b09
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20220412140153-99c896dcbdbf
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220412151528-78ae6d9f855e
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220412155728-fc0951619811
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220412160021-568002c53a38
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220412173106-dab300c2e2db
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20211221033732-78e26f6c23a6
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220412173758-1f0485c0f75e
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20220412174440-db2ee7ce5472
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220412173416-8ce0e99f0e5f
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.33.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
