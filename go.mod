module gitlab.com/voipbin/bin-manager/flow-manager.git

go 1.14

require (
	github.com/go-redis/redis/v8 v8.8.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/onsi/ginkgo v1.16.1 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.8.1
	github.com/smotes/purse v1.0.1
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20210412073319-40ad8868dc59
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20210413030010-98987a34c81d
	golang.org/x/net v0.0.0-20210410081132-afb366fc7cd1 // indirect
	golang.org/x/sys v0.0.0-20210412220455-f1c623a9e750 // indirect
	google.golang.org/genproto v0.0.0-20210406143921-e86de6bf7a46 // indirect
)
