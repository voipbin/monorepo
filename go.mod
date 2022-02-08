module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.17

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v1.0.0
	golang.org/x/sys v0.0.0-20220207234003-57398862261d // indirect
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/prometheus/client_golang v1.12.1
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220208193438-0e648b83c246
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220203012553-7a6af2df4d9e
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220203063008-305be30288a4
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220203044108-296babaad5c9
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220207024948-b975c9ff24f5
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220208185140-cfc6013ffd01
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220129111440-7128c74e331c
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220128172756-b3561f5b2c80
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20211221033732-78e26f6c23a6
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220206205721-6f56cc4c3c1e
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20211221033736-1c9c086d3d00
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220203062028-e58cb4d7c71e
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
