package domainhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
)

// CreateDomain creates a new domain and returns a created domain info
func (h *domainHandler) CreateDomain(ctx context.Context, userID uint64, domainName string) (*models.Domain, error) {
	// create domain
	d := &models.Domain{
		ID:     uuid.Must(uuid.NewV4()),
		UserID: userID,
	}

	if err := h.dbBin.DomainCreate(ctx, d); err != nil {
		logrus.Errorf("Could not create a domain info. err: %v", err)
		return nil, err
	}

	res, err := h.dbBin.DomainGet(ctx, d.ID)
	if err != nil {
		logrus.Errorf("Could not get created domain info. err: %v", err)
		return nil, err
	}

	return res, nil
}
