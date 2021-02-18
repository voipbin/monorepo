package contacthandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
)

// ContactGetsByEndpoint returns list of contacts
func (h *contactHandler) ContactGetsByEndpoint(ctx context.Context, endpoint string) ([]*models.AstContact, error) {
	logrus.Debugf("Getting a contact info. endpoint: %s", endpoint)

	contacts, err := h.dbAst.AstContactGetsByEndpoint(ctx, endpoint)
	if err != nil {
		logrus.Errorf("Could not get contacts info. endpoint: %s, err: %v", endpoint, err)
		return nil, err
	}
	return contacts, err
}
