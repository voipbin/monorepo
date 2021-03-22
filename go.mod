module gitlab.com/voipbin/bin-manager/call-manager.git

go 1.14

require (
	github.com/go-redis/redis/v8 v8.7.1
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.8.1
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20210322032029-24e9a68b7a4c
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20210321130210-a1303884d876
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20210321121353-4946f2db3798
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20210317181219-9039fab1cff3
)
