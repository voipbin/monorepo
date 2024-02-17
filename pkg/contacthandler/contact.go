package contacthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/common"
)

// ContactGetsByExtension returns list of contacts
func (h *contactHandler) ContactGetsByExtension(ctx context.Context, customerID uuid.UUID, ext string) ([]*astcontact.AstContact, error) {
	logrus.Debugf("Getting a contact info. endpoint: %s", ext)

	endpoint := common.GenerateEndpointExtension(customerID, ext)
	contacts, err := h.dbAst.AstContactGetsByEndpoint(ctx, endpoint)
	if err != nil {
		logrus.Errorf("Could not get contacts info. endpoint: %s, target_endpoint: %s, err: %v", ext, endpoint, err)
		return nil, err
	}

	return contacts, err
}

// ContactRefreshByEndpoint refresh the list of contacts
func (h *contactHandler) ContactRefreshByEndpoint(ctx context.Context, customerID uuid.UUID, ext string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactRefreshByEndpoint",
		"customer_id": customerID,
		"extension":   ext,
	})

	endpoint := common.GenerateEndpointExtension(customerID, ext)
	log.Debugf("Refreshing the contacts of the endpoint. endpoint: %s", endpoint)

	if err := h.dbAst.AstContactDeleteFromCache(ctx, endpoint); err != nil {
		log.Errorf("Could not delete the cache. err: %v", err)
		return errors.Wrap(err, "Could not delete the cache.")
	}

	return nil
}
