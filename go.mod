module gitlab.com/voipbin/bin-manager/storage-manager.git

go 1.14

require (
	cloud.google.com/go/storage v1.10.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.8.1
	gitlab.com/voipbin/bin-manager/api-manager.git v0.0.0-20210402144932-b009a9fdec61
	gitlab.com/voipbin/bin-manager/call-manager.git v0.0.0-20210401181828-39f83862bad2
	gitlab.com/voipbin/bin-manager/common-handler.git v0.0.0-20210314173554-61bfbbbd5633
	gitlab.com/voipbin/bin-manager/flow-manager.git v0.0.0-20210401061929-70b309d80bfe
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	google.golang.org/api v0.33.0
)
