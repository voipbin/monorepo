module gitlab.com/voipbin/bin-manager/flow-manager.git

go 1.16

require (
	github.com/go-redis/redis/v8 v8.11.3
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20210831171004-117e81c6a319
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20210902151352-becc13387646
)
