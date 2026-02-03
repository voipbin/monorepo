package contact

// Event types for contact-manager events published to RabbitMQ.
// These events are published when contacts are created, updated, or deleted.
const (
	// EventTypeContactCreated is published when a new contact is created.
	EventTypeContactCreated string = "contact_created"

	// EventTypeContactUpdated is published when a contact is modified.
	EventTypeContactUpdated string = "contact_updated"

	// EventTypeContactDeleted is published when a contact is soft-deleted.
	EventTypeContactDeleted string = "contact_deleted"
)
