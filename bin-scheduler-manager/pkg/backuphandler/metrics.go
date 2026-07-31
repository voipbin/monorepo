package backuphandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

const promNamespace = "scheduler_manager"

// promBackupLastBytes exports the size of the most recent successful dump
// (design §5.7/§7 — a doctor check flags 0 or missing).
var promBackupLastBytes = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: promNamespace,
		Name:      "backup_last_bytes",
		Help:      "Size in bytes of the most recent successful database backup dump.",
	},
)

func init() {
	prometheus.MustRegister(promBackupLastBytes)
}
