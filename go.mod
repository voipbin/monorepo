module gitlab.com/voipbin/bin-manager/stt-manager.git

go 1.14

require (
	cloud.google.com/go v0.70.0
	cloud.google.com/go/storage v1.10.0
	github.com/go-redis/redis/v8 v8.8.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.8.1
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20210401181828-39f83862bad2
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20210408192237-4079c56261c5
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20210410172355-d1894606b9df
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/text v0.3.6
	google.golang.org/api v0.33.0
	google.golang.org/genproto v0.0.0-20210402141018-6c239bbf2bb1
)
