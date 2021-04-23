module gitlab.com/voipbin/bin-manager/api-manager.git

go 1.14

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-gonic/gin v1.6.3
	github.com/go-playground/validator/v10 v10.5.0 // indirect
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
	github.com/swaggo/files v0.0.0-20190704085106-630677cd5c14
	github.com/swaggo/gin-swagger v1.3.0
	github.com/swaggo/swag v1.7.0
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20210404020257-e89ee0f0ee04
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20210401061929-70b309d80bfe
	gitlab.com/voipbin/bin-manager/number-manager.git v0.0.0-20210405060100-f0029edb7aa2
	gitlab.com/voipbin/bin-manager/registrar-manager.git v0.0.0-20210323024036-8c44f8dca2de
	gitlab.com/voipbin/bin-manager/storage-manager.git v0.0.0-20210408192237-4079c56261c5
	gitlab.com/voipbin/bin-manager/transcribe-manager.git v0.0.0-20210422140528-113e05de23e1
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	google.golang.org/genproto v0.0.0-20210406143921-e86de6bf7a46 // indirect
)
