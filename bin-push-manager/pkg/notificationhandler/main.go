package notificationhandler

//go:generate mockgen -package notificationhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/pkg/dbhandler"
	"monorepo/bin-push-manager/pkg/pushhandler"
)

// deliveryTimeout is the longest delivery attempt to one device.
const deliveryTimeout = 5 * time.Second

// listPageToken is the page token of the internal list: a time after every record.
const listPageToken = "9999-12-31T23:59:59.999999Z"

// listSize is above the per-agent device cap.
const listSize = 1000

// NotificationHandler interface
type NotificationHandler interface {
	// Send sends the notification to the live devices of the agent, or to one of them when deviceID is set.
	Send(ctx context.Context, customerID, agentID, deviceID uuid.UUID, kind notification.Kind, data map[string]string) (*notification.SendResult, error)
}

type notificationHandler struct {
	db     dbhandler.DBHandler
	sender pushhandler.Sender

	// now returns the current time. It is a field so the tests can fix it.
	now func() time.Time
}

// result labels of the notification metric
const (
	resultSent          = "sent"
	resultUnregistered  = "unregistered"
	resultFailed        = "failed"
	resultNotConfigured = "not_configured"
)

var (
	metricsNamespace = "push_manager"

	promNotificationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "notification_total",
			Help:      "Total number of notification deliveries by kind and result",
		},
		[]string{"kind", "result"},
	)
)

func init() {
	prometheus.MustRegister(promNotificationTotal)
}

// NewNotificationHandler return NotificationHandler interface
func NewNotificationHandler(db dbhandler.DBHandler, sender pushhandler.Sender) NotificationHandler {
	return &notificationHandler{
		db:     db,
		sender: sender,
		now:    time.Now,
	}
}
