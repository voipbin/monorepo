module monorepo/bin-common-handler

go 1.22

toolchain go1.22.0

replace monorepo/bin-agent-manager => ../bin-agent-manager

replace monorepo/bin-billing-manager => ../bin-billing-manager

replace monorepo/bin-call-manager => ../bin-call-manager

replace monorepo/bin-common-handler => ../bin-common-handler

replace monorepo/bin-campaign-manager => ../bin-campaign-manager

replace monorepo/bin-chat-manager => ../bin-chat-manager

replace monorepo/bin-chatbot-manager => ../bin-chatbot-manager

replace monorepo/bin-conference-manager => ../bin-conference-manager

replace monorepo/bin-conversation-manager => ../bin-conversation-manager

replace monorepo/bin-customer-manager => ../bin-customer-manager

replace monorepo/bin-flow-manager => ../bin-flow-manager

replace monorepo/bin-hook-manager => ../bin-hook-manager

replace monorepo/bin-message-manager => ../bin-message-manager

replace monorepo/bin-number-manager => ../bin-number-manager

replace monorepo/bin-outdial-manager => ../bin-outdial-manager

replace monorepo/bin-queue-manager => ../bin-queue-manager

replace monorepo/bin-registrar-manager => ../bin-registrar-manager

replace monorepo/bin-route-manager => ../bin-route-manager

replace monorepo/bin-storage-manager => ../bin-storage-manager

replace monorepo/bin-tag-manager => ../bin-tag-manager

replace monorepo/bin-transcribe-manager => ../bin-transcribe-manager

replace monorepo/bin-transfer-manager => ../bin-transfer-manager

replace monorepo/bin-tts-manager => ../bin-tts-manager

replace monorepo/bin-webhook-manager => ../bin-webhook-manager

require (
	github.com/golang/mock v1.6.0
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/sys v0.20.0 // indirect
)

require (
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.19.1
	github.com/rabbitmq/amqp091-go v1.10.0
	monorepo/bin-agent-manager v0.0.0-20240328054741-55144017eccd
	monorepo/bin-billing-manager v0.0.0-20240408051040-600f0028fbab
	monorepo/bin-call-manager v0.0.0-20240403030948-51eb7c33cf9a
	monorepo/bin-campaign-manager v0.0.0-20240313031908-f098e3fb6f12
	monorepo/bin-chat-manager v0.0.0-20240313050741-a2ced5030a06
	monorepo/bin-chatbot-manager v0.0.0-20240313050825-1c666b883013
	monorepo/bin-conference-manager v0.0.0-20240329045829-45dc5f4e4e76
	monorepo/bin-conversation-manager v0.0.0-20231117134833-7918f76572d4
	monorepo/bin-customer-manager v0.0.0-20240408042746-c45b2b5aa984
	monorepo/bin-flow-manager v0.0.0-20240403034140-ce82222fe7f4
	monorepo/bin-hook-manager v0.0.0-20240313052650-d3e4c79af4c0
	monorepo/bin-message-manager v0.0.0-20240328053936-9008e28c2268
	monorepo/bin-number-manager v0.0.0-20240328055052-ec1c723aa183
	monorepo/bin-outdial-manager v0.0.0-20240313064601-888fe8578646
	monorepo/bin-queue-manager v0.0.0-20240402021210-adac880b81da
	monorepo/bin-registrar-manager v0.0.0-20240402051305-cf14186e380d
	monorepo/bin-route-manager v0.0.0-20240313065038-1498b922bb24
	monorepo/bin-storage-manager v0.0.0-20240330083852-ab008a2e3880
	monorepo/bin-tag-manager v0.0.0-20240313070856-7d3433af905d
	monorepo/bin-transcribe-manager v0.0.0-20240405044227-febd49f8b700
	monorepo/bin-transfer-manager v0.0.0-20230419025515-44dea928ef34
	monorepo/bin-tts-manager v0.0.0-20240313070648-addf67d64996
	monorepo/bin-webhook-manager v0.0.0-20240313071253-ebca1db1437c
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.53.0 // indirect
	github.com/prometheus/procfs v0.15.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)
