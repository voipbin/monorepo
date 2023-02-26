package contacthandler

import (
	"context"
	"fmt"

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
	logrus.Debugf("Refreshing the contacts of the endpoint. endpoint: %s", endpoint)

	target := fmt.Sprintf("%s.%s", endpoint, common.BaseDomainName)
	return h.dbAst.AstContactDeleteFromCache(ctx, target)
}
