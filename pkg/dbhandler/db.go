package dbhandler

import "github.com/sirupsen/logrus"

// numberGetFromRow gets the number from the row.
func (h *handler) Close() {
	log := logrus.WithField("func", "Close")

	log.Debug("Closing database connection.")
	h.db.Close()
}
