package resourcehandler

import (
	"context"
	"monorepo/bin-agent-manager/models/resource"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Gets returns agents
func (h *resourceHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*resource.Resource, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.ResourceGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get retrieves a single resource by its ID from the database.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// id (uuid.UUID): The unique identifier of the resource to retrieve.
//
// Returns:
// (*resource.Resource, error): A pointer to the retrieved resource and any error that occurred during the operation.
// If the operation is successful, the error will be nil.
// If the resource is not found, the error will be of type *sql.ErrNoRows.
func (h *resourceHandler) Get(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	// Create a logrus logger with a field for the function name.
	log := logrus.WithField("func", "Get")

	// Call the ResourceGet method of the h.db (database handler) to retrieve the resource.
	res, err := h.db.ResourceGet(ctx, id)
	if err != nil {
		// Log the error with a message indicating the failure to get the resource.
		log.Errorf("Could not get resource info. err: %v", err)
		// Return nil and the error.
		return nil, err
	}

	// Return the retrieved resource and nil error.
	return res, nil
}

// Create creates a new resource.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// customerID (uuid.UUID): The ID of the customer associated with the resource.
// ownerID (uuid.UUID): The ID of the owner(agent) associated with the resource.
// referenceType (resource.Type): The type of reference for the resource.
// data (interface{}): The data associated with the resource.
//
// Returns:
// (*resource.Resource, error): A pointer to the created resource and any error that occurred during the operation.
// If the operation is successful, the error will be nil.
func (h *resourceHandler) Create(ctx context.Context, customerID uuid.UUID, ownerID uuid.UUID, referenceType resource.ReferenceType, referenceID uuid.UUID, data interface{}) (*resource.Resource, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"customer_id":    customerID,
		"owner_id":       ownerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"data":           data,
	})
	log.Debug("Creating a new resource.")

	id := h.utilHandler.UUIDCreate()
	tmp := &resource.Resource{
		ID:            id,
		CustomerID:    customerID,
		OwnerID:       ownerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Data:          data,
	}

	if errCreate := h.db.ResourceCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create an agent. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "could not create an agent")
	}

	res, err := h.db.ResourceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created resource info. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishEvent(ctx, resource.EventTypeResourceCreated, res)

	return res, nil
}

// Delete deletes the resource.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// id (uuid.UUID): The unique identifier of the resource to delete.
//
// Returns:
// (*resource.Resource, error): A pointer to the deleted resource and any error that occurred during the operation.
// If the operation is successful, the error will be nil.
// If the resource is not found, the error will be of type *sql.ErrNoRows.
func (h *resourceHandler) Delete(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	// Create a logrus logger with fields for the function name and resource ID.
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"resource_id": id,
	})

	// Call the ResourceDelete method of the h.db (database handler) to delete the resource.
	if errDelete := h.db.ResourceDelete(ctx, id); errDelete != nil {
		// Log the error with a message indicating the failure to delete the resource.
		log.Errorf("Could not delete the agent. err: %v", errDelete)
		// Wrap the error and return it.
		return nil, errors.Wrap(errDelete, "could not delete the agent")
	}

	// Call the ResourceGet method of the h.db (database handler) to retrieve the deleted resource.
	res, err := h.db.ResourceGet(ctx, id)
	if err != nil {
		// Log the error with a message indicating the failure to get the deleted resource info.
		log.Errorf("Could not get deleted resource info. err: %v", err)
		// Wrap the error and return it.
		return nil, errors.Wrapf(err, "Could not get deleted resource info.")
	}

	// Publish a resource deleted event using the notifyHandler.
	h.notifyHandler.PublishEvent(ctx, resource.EventTypeResourceDeleted, res)

	// Return the deleted resource and nil error.
	return res, nil
}

// UpdateData updates the data of a resource in the database.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// id (uuid.UUID): The unique identifier of the resource to update.
// data (interface{}): The new data to be set for the resource.
//
// Returns:
// (*resource.Resource, error): A pointer to the updated resource and any error that occurred during the operation.
// If the operation is successful, the error will be nil.
// If the resource is not found, the error will be of type *sql.ErrNoRows.
func (h *resourceHandler) UpdateData(ctx context.Context, id uuid.UUID, data interface{}) (*resource.Resource, error) {
	// Create a logrus logger with fields for the function name and resource ID.
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateData",
		"resource_id": id,
	})

	// Call the ResourceSetData method of the h.db (database handler) to update the resource data.
	if errSet := h.db.ResourceSetData(ctx, id, data); errSet != nil {
		// Log the error with a message indicating the failure to update the resource data.
		log.Errorf("Could not update the resource data. err: %v", errSet)
		// Return nil and the error.
		return nil, errSet
	}

	// Call the ResourceGet method of the h.db (database handler) to retrieve the updated resource.
	res, err := h.db.ResourceGet(ctx, id)
	if err != nil {
		// Log the error with a message indicating the failure to get the updated resource info.
		log.Errorf("Could not get deleted resource info. err: %v", err)
		// Wrap the error and return it.
		return nil, errors.Wrapf(err, "Could not get deleted resource info.")
	}

	// Publish a resource updated event using the notifyHandler.
	h.notifyHandler.PublishEvent(ctx, resource.EventTypeResourceUpdated, res)

	// Return the updated resource and nil error.
	return res, nil
}
