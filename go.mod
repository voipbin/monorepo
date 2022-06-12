module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.17

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v1.0.0
	golang.org/x/sys v0.0.0-20220610221304-9f5ed59c137d // indirect
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/prometheus/client_golang v1.12.2
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220611213710-f484abced1e9
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220611211540-aca4fc4de5a8
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20220611222844-e5c44b372689
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220412180445-b5f041da51b3
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220607052644-b8feae22071c
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220610055038-1e876c49a5c3
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20220412180021-aa7216424a33
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20220612082606-32ea7b93d6bd
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220508083304-215537bdcdaa
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220611220718-0fcb5e204d02
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220524124918-fc99b31aebd4
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220424171422-5a3b77610aae
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20220413034054-1271ef0d98c3
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220412183503-21306796da28
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20220412175449-ec3a017dbd18
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220607060321-69fc9046521a
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.34.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
