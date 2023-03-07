package contacthandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/common"
)

// ContactGetsByEndpoint returns list of contacts
func (h *contactHandler) ContactGetsByEndpoint(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error) {
	logrus.Debugf("Getting a contact info. endpoint: %s", endpoint)

	target := fmt.Sprintf("%s.%s", endpoint, common.BaseDomainName)
	contacts, err := h.dbAst.AstContactGetsByEndpoint(ctx, target)
	if err != nil {
		logrus.Errorf("Could not get contacts info. endpoint: %s, target_endpoint: %s, err: %v", endpoint, target, err)
		return nil, err
	}
	return contacts, err
}

// ContactRefreshByEndpoint refresh the list of contacts
func (h *contactHandler) ContactRefreshByEndpoint(ctx context.Context, endpoint string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "ContactRefreshByEndpoint",
		"endpoint": endpoint,
	})

	contact := fmt.Sprintf("%s.%s", endpoint, common.BaseDomainName)

	log.Debugf("Refreshing the contacts of the endpoint. endpoint: %s, contact: %s", endpoint, contact)

	if err := h.dbAst.AstContactDeleteFromCache(ctx, contact); err != nil {
		log.Errorf("Could not delete the cache. err: %v", err)
		return errors.Wrap(err, "Could not delete the cache.")
	}

	return nil
}
