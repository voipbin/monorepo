module gitlab.com/voipbin/bin-manager/call-manager.git

go 1.16

require (
	github.com/go-redis/redis/v8 v8.8.2
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.8.1
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20210821180258-3162580740d1
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20210405060100-f0029edb7aa2
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20210323024036-8c44f8dca2de
	gitlab.com/voipbin/bin-manager/tts-manager.git v0.0.0-20210323024756-a7b0838b069d
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20210909052905-558ef2d8332e
	golang.org/x/net v0.0.0-20210415231046-e915ea6b2b7d // indirect
	golang.org/x/sys v0.0.0-20210419170143-37df388d1f33 // indirect
	google.golang.org/genproto v0.0.0-20210416161957-9910b6c460de // indirect
)
