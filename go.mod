module gitlab.com/voipbin/bin-manager/call-manager.git

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
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20210925155534-4eb6f4abd24f
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20210405060100-f0029edb7aa2
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20210323024036-8c44f8dca2de
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20210323024756-a7b0838b069d
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20210909052905-558ef2d8332e
)
