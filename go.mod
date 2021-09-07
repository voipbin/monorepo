module gitlab.com/voipbin/bin-manager/transcribe-manager.git

go 1.14

require (
	cloud.google.com/go v0.94.0 // indirect
	cloud.google.com/go/speech v0.1.0
	cloud.google.com/go/storage v1.16.1
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/go-redis/redis/v8 v8.11.3
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pion/rtp v1.7.2
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20210905153006-e26b10b17032
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20210409195628-27c9b796b068
	gitlab.com/voipbin/bin-manager/webhook-manager.git v0.0.0-20210410172355-d1894606b9df
	golang.org/x/net v0.0.0-20210825183410-e898025ed96a // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	golang.org/x/sys v0.0.0-20210831042530-f4d43177bf5e // indirect
	golang.org/x/text v0.3.7
	google.golang.org/api v0.56.0
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2
)
