module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.18

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.9.0
	github.com/streadway/amqp v1.0.0
	golang.org/x/sys v0.5.0 // indirect
)

require (
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/prometheus/client_golang v1.14.0
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20230131002231-39f618279ca5
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20230207053459-e6665298a1f4
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20221207172609-98006dd828f9
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20220926023600-ef42f2bac093
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20230208064515-e645d468b8ae
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20230206053408-48096fff3918
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20220722162017-92d117261a6a
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220616065935-caa48b9b0bb5
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20230203154425-8d2aff84eb6e
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20221211030023-7909940f4600
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20221216125212-e74e4c409422
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20221206165111-75edd8be2cb9
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220722164233-2ec431be7901
	gitlab.com/voipbin/bin-manager/queue-manager.git v0.0.0-20221124180605-39b41a7a1ada
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20230127044148-d2db8f41581c
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20221029145057-ef1ebd21d097
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20230108123249-9f26b43b8ea9
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20230131002226-28fcd0534b57
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20230204142400-175f62ac7400
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220723135740-c87a1ef4af61
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
