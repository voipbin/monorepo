package speaking

// Status represents the status of a speaking session
type Status string

const (
	StatusInitiating Status = "initiating"
	StatusActive     Status = "active"
	StatusStopped    Status = "stopped"
)
