module monorepo/bin-timeline-manager

go 1.25.3

replace monorepo/bin-common-handler => ../bin-common-handler

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.43.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.18.1
	github.com/joonix/log v0.0.0-20251205082533-cd78070927ea
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.23.2
	github.com/sirupsen/logrus v1.9.4
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	go.uber.org/mock v0.6.0
	monorepo/bin-common-handler v0.0.0-00010101000000-000000000000
)
