module gitlab.com/voipbin/bin-manager/common-handler.git

go 1.14

require (
	github.com/go-redis/redis/v8 v8.0.0
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang/mock v1.4.4
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.6.0
	github.com/smotes/purse v1.0.1
	github.com/streadway/amqp v1.0.0
	gitlab.com/voipbin/bin-manager/api-manager.git v0.0.0-20200914235904-f3d84b2adfb7
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20200914235843-f7422c21d446
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20200914235906-3d3077deda96
)
