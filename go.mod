module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.17

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v1.0.0
	golang.org/x/sys v0.0.0-20220128215802-99c3d69c2c27 // indirect
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/prometheus/client_golang v1.12.1
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220131015406-68c5d0a1a790
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220129114754-16a5f2f6812f
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220129071825-44f3b25d509f
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220127020659-07e0a18cc4f7
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220129073757-8b16b0bb6f7b
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220129111655-419853d35f8c
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220129111440-7128c74e331c
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220128172756-b3561f5b2c80
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20211221033732-78e26f6c23a6
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220129091806-b8d5c0eb6df5
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20211221033736-1c9c086d3d00
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220131055855-db615a133f71
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
