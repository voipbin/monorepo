package queue

// list of call queue types
const (
	EventTypeQueueCreated string = "queue_created" // the queue has created
	EventTypeQueueUpdated string = "queue_updated" // the queue has updated
	EventTypeQueueDeleted string = "queue_deleted" // the queue had deleted
)
