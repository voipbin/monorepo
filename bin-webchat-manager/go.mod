module monorepo/bin-webchat-manager

go 1.25.3

replace monorepo/bin-common-handler => ../bin-common-handler

replace monorepo/bin-direct-manager => ../bin-direct-manager

replace monorepo/bin-flow-manager => ../bin-flow-manager

require (
	github.com/Masterminds/squirrel v1.5.4
	github.com/go-redis/redis/v8 v8.11.5
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/mattn/go-sqlite3 v1.14.48
	github.com/sirupsen/logrus v1.9.4
	github.com/smotes/purse v1.0.1
	go.uber.org/mock v0.6.0
	monorepo/bin-common-handler v0.0.0-20240408033155-50f0cd082334
)

require (
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
