package sock

type QueueType string

const (
	QueueTypeNormal   QueueType = "normal"
	QueueTypeVolatile QueueType = "volatile"
)
