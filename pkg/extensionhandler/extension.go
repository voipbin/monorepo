package extensionhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
)

// CreateExtension creates a new extension
// in this func, it creates all releated asterisk resource as well.
func (h *extensionHandler) CreateExtension(ctx context.Context, userID uint64, domainID uuid.UUID, ext string, password string) (*models.Extension, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"user_id":   userID,
			"domain_id": domainID,
			"extension": ext,
		},
	)

	// get domain
	d, err := h.dbBin.DomainGet(ctx, domainID)
	if err != nil {
		log.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	// create id
	commonID := fmt.Sprintf("%s@%s", ext, d.DomainName)

	// create aor
	maxContacts := 1
	aor := &models.AstAOR{
		ID:          &commonID,
		MaxContacts: &maxContacts,
	}
	if err := h.dbAst.AstAORCreate(ctx, aor); err != nil {
		log.Errorf("Could not create AOR. err: %v", err)
		return nil, err
	}

	// create auth
	authType := "userpass"
	auth := &models.AstAuth{
		ID:       &commonID,
		AuthType: &authType,
		Username: &ext,
		Password: &password,
		Realm:    &d.DomainName,
	}
	if err := h.dbAst.AstAuthCreate(ctx, auth); err != nil {
		log.Errorf("Could not create Auth. Err: %v", err)
		return nil, err
	}

	// create endpoint
	endpoint := &models.AstEndpoint{
		ID:   &commonID,
		AORs: &commonID,
		Auth: &commonID,
	}
	if err := h.dbAst.AstEndpointCreate(ctx, endpoint); err != nil {
		log.Errorf("Could not create Endpoint. err: %v", err)
		return nil, err
	}

	// create extension
	extension := &models.Extension{
		ID:     uuid.Must(uuid.NewV4()),
		UserID: userID,

		DomainID: d.ID,

		AORID:      *aor.ID,
		AuthID:     *auth.ID,
		EndpointID: *endpoint.ID,

		Extension: ext,
		Password:  password,
	}
	if err := h.dbBin.ExtensionCreate(ctx, extension); err != nil {
		log.Errorf("Could not create extension. err: %v", err)
		return nil, err
	}

	res, err := h.dbBin.ExtensionGet(ctx, extension.ID)
	if err != nil {
		log.Errorf("Could not get created extension. err: %v", err)
		return nil, err
	}

	return res, nil
}
