module gitlab.com/voipbin/bin-manager/queue-manager.git

go 1.18

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.7.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.14.0
	github.com/sirupsen/logrus v1.9.0
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/agent-manager.git v0.0.0-20230131002231-39f618279ca5
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20230218072504-af51ee98fb87
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20230215072212-a4a31680cc3a
	gitlab.com/voipbin/bin-manager/conference-manager.git v0.0.0-20230213110854-5fda5ad5ace2
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20230218122107-31fb15165405
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/streadway/amqp v1.0.0 // indirect
	gitlab.com/voipbin/bin-manager/campaign-manager.git v0.0.0-20221207172609-98006dd828f9 // indirect
	gitlab.com/voipbin/bin-manager/chat-manager.git v0.0.0-20220926023600-ef42f2bac093 // indirect
	gitlab.com/voipbin/bin-manager/chatbot-manager.git v0.0.0-20230210081649-48f1a33defc6 // indirect
	gitlab.com/voipbin/bin-manager/conversation-manager.git v0.0.0-20220722162017-92d117261a6a // indirect
	gitlab.com/voipbin/bin-manager/customer-manager.git v0.0.0-20220616065935-caa48b9b0bb5 // indirect
	gitlab.com/voipbin/bin-manager/hook-manager.git v0.0.0-20221211030023-7909940f4600 // indirect
	gitlab.com/voipbin/bin-manager/message-manager.git v0.0.0-20230212162339-c9e22d6f99d1 // indirect
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20221206165111-75edd8be2cb9 // indirect
	gitlab.com/voipbin/bin-manager/outdial-manager.git v0.0.0-20220722164233-2ec431be7901 // indirect
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20230127044148-d2db8f41581c // indirect
	gitlab.com/voipbin/bin-manager/route-manager.git v0.0.0-20221029145057-ef1ebd21d097 // indirect
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20230108123249-9f26b43b8ea9 // indirect
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20230218150249-e83abd84dab1 // indirect
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20230204142400-175f62ac7400 // indirect
	gitlab.com/voipbin/bin-manager/user-manager.git v0.0.0-20211201060242-1cc38a3221d0 // indirect
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20220723135740-c87a1ef4af61 // indirect
	golang.org/x/sys v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20230216225411-c8e22ba71e44 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
