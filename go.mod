module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.17

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v1.0.0
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/prometheus/client_golang v1.12.1
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220413020856-9bff29d04032
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220414221219-afe912d6515b
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220412180445-b5f041da51b3
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220413021200-204155e694ef
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220416151349-0057f244ebe4
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20220412180021-aa7216424a33
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20220413021508-2a83dea8ea6c
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220413022009-130ed4292bdf
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220413022604-0b7ef909e1d9
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220413023153-1b9012469574
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220412173106-dab300c2e2db
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20220413034054-1271ef0d98c3
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220412183503-21306796da28
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20220412175449-ec3a017dbd18
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
