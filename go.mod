module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.17

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v1.0.0
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/prometheus/client_golang v1.12.0
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20220128030351-b09ba2ebff94
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20220127092610-7d5c827992db
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20220127085754-375cf37c6d7f
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220125172215-6244f425bebe
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20220127083953-ef0331841884
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20220128044746-5b1c59714700
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20220128054051-435c02f43a53
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20220128040744-1da444546c76
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20211221033732-78e26f6c23a6
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20220128055248-837e7e1f0a79
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20211221033736-1c9c086d3d00
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20210926125524-0e0a20ea02b2
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
