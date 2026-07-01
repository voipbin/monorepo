package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcontact "monorepo/bin-contact-manager/models/contact"
)

// ContactAddressList returns a paged list of addresses for the authenticated customer.
func (h *serviceHandler) ContactAddressList(
	ctx context.Context,
	a *auth.AuthIdentity,
	filters map[string]any,
	pageToken string,
	pageSize uint64,
) ([]cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactAddressList",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// inject customer_id so the backend scopes the query
	filters["customer_id"] = a.CustomerID

	res, err := h.reqHandler.ContactV1ContactAddressList(ctx, a.CustomerID, filters, pageToken, pageSize)
	if err != nil {
		log.Infof("Could not list contact addresses. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ContactAddressGet returns a single address by ID, scoped to the customer.
func (h *serviceHandler) ContactAddressGet(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactAddressGet",
		"customer_id": a.CustomerID,
		"address_id":  addressID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressGet(ctx, a.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}

	// tenant isolation check
	if res.CustomerID != a.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return res, nil
}

// ContactAddressCreateIndependent creates an address via the /contact_addresses endpoint.
// If contactID is uuid.Nil, creates an unresolved address (contact_id IS NULL) scoped to
// the authenticated customer, skipping the contact-ownership lookup entirely.
func (h *serviceHandler) ContactAddressCreateIndependent(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	addrType string,
	target string,
	isPrimary bool,
	name string,
	detail string,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactAddressCreateIndependent",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if contactID == uuid.Nil {
		// Unresolved creation: no contact to look up, scope directly to a.CustomerID.
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}

		res, err := h.reqHandler.ContactV1ContactAddressCreate(ctx, a.CustomerID, uuid.Nil, addrType, target, isPrimary, name, detail)
		if err != nil {
			log.Infof("Could not create unresolved contact address. err: %v", err)
			return nil, err
		}

		return res, nil
	}

	// verify contact ownership
	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressCreate(ctx, ct.CustomerID, contactID, addrType, target, isPrimary, name, detail)
	if err != nil {
		log.Infof("Could not create contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ContactAddressClaim attaches a currently-unresolved address to a contact.
// Tenant isolation (both the address AND the target contact must belong to
// a.CustomerID) is enforced here, per the "auth/ownership only in
// bin-api-manager" rule.
func (h *serviceHandler) ContactAddressClaim(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
	contactID uuid.UUID,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactAddressClaim",
		"customer_id": a.CustomerID,
		"address_id":  addressID,
		"contact_id":  contactID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Tenant isolation: AddressGet scopes by customerID
	addr, err := h.reqHandler.ContactV1ContactAddressGet(ctx, a.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}
	if addr.CustomerID != a.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Tenant isolation: verify the target contact also belongs to this customer
	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}
	if ct.CustomerID != a.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressClaim(ctx, a.CustomerID, addressID, contactID)
	if err != nil {
		log.Infof("Could not claim contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ContactAddressUpdateIndependent updates an address via the /contact_addresses/{id} endpoint.
func (h *serviceHandler) ContactAddressUpdateIndependent(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
	fields map[string]any,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactAddressUpdateIndependent",
		"customer_id": a.CustomerID,
		"address_id":  addressID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// Tenant isolation: AddressGet scopes by customerID
	addr, err := h.reqHandler.ContactV1ContactAddressGet(ctx, a.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, addr.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressUpdate(ctx, a.CustomerID, addr.ContactID, addressID, fields)
	if err != nil {
		log.Infof("Could not update contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ContactAddressDeleteIndependent deletes an address via the /contact_addresses/{id} endpoint.
func (h *serviceHandler) ContactAddressDeleteIndependent(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactAddressDeleteIndependent",
		"customer_id": a.CustomerID,
		"address_id":  addressID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// Tenant isolation: AddressGet scopes by customerID
	addr, err := h.reqHandler.ContactV1ContactAddressGet(ctx, a.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, addr.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressDelete(ctx, a.CustomerID, addr.ContactID, addressID)
	if err != nil {
		log.Infof("Could not delete contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentContactAddressList lists addresses for the service agent's customer.
func (h *serviceHandler) ServiceAgentContactAddressList(
	ctx context.Context,
	a *auth.AuthIdentity,
	filters map[string]any,
	pageToken string,
	pageSize uint64,
) ([]cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactAddressList",
		"customer_id": a.CustomerID,
	})

	agent, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, agent.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressList(ctx, agent.CustomerID, filters, pageToken, pageSize)
	if err != nil {
		log.Infof("Could not list contact addresses. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentContactAddressGet returns a single address for the service agent.
func (h *serviceHandler) ServiceAgentContactAddressGet(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ServiceAgentContactAddressGet",
		"address_id": addressID,
	})

	agent, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.ContactV1ContactAddressGet(ctx, agent.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}

	if res.CustomerID != agent.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return res, nil
}

// ServiceAgentContactAddressCreateIndependent creates an address for the service agent.
// If contactID is uuid.Nil, creates an unresolved address (contact_id IS NULL) scoped to
// the agent's customer, skipping the contact-ownership lookup entirely.
func (h *serviceHandler) ServiceAgentContactAddressCreateIndependent(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	addrType string,
	target string,
	isPrimary bool,
	name string,
	detail string,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ServiceAgentContactAddressCreateIndependent",
		"contact_id": contactID,
	})

	agent, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return nil, err
	}

	if contactID == uuid.Nil {
		// Unresolved creation: no contact to look up, scope directly to agent.CustomerID.
		res, err := h.reqHandler.ContactV1ContactAddressCreate(ctx, agent.CustomerID, uuid.Nil, addrType, target, isPrimary, name, detail)
		if err != nil {
			log.Infof("Could not create unresolved contact address. err: %v", err)
			return nil, err
		}

		return res, nil
	}

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if ct.CustomerID != agent.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressCreate(ctx, agent.CustomerID, contactID, addrType, target, isPrimary, name, detail)
	if err != nil {
		log.Infof("Could not create contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentContactAddressClaim attaches a currently-unresolved address to a contact,
// for the service agent surface. Tenant isolation (both the address AND the target
// contact must belong to the agent's customer) is enforced here.
func (h *serviceHandler) ServiceAgentContactAddressClaim(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
	contactID uuid.UUID,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ServiceAgentContactAddressClaim",
		"address_id": addressID,
		"contact_id": contactID,
	})

	agent, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return nil, err
	}

	// Tenant isolation: AddressGet scopes by customerID
	addr, err := h.reqHandler.ContactV1ContactAddressGet(ctx, agent.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}
	if addr.CustomerID != agent.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Tenant isolation: verify the target contact also belongs to this customer
	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}
	if ct.CustomerID != agent.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressClaim(ctx, agent.CustomerID, addressID, contactID)
	if err != nil {
		log.Infof("Could not claim contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentContactAddressUpdateIndependent updates an address for the service agent.
func (h *serviceHandler) ServiceAgentContactAddressUpdateIndependent(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
	fields map[string]any,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ServiceAgentContactAddressUpdateIndependent",
		"address_id": addressID,
	})

	agent, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return nil, err
	}

	// Tenant isolation: AddressGet scopes by customerID
	addr, err := h.reqHandler.ContactV1ContactAddressGet(ctx, agent.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}

	if addr.CustomerID != agent.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressUpdate(ctx, agent.CustomerID, addr.ContactID, addressID, fields)
	if err != nil {
		log.Infof("Could not update contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentContactAddressDeleteIndependent deletes an address for the service agent.
func (h *serviceHandler) ServiceAgentContactAddressDeleteIndependent(
	ctx context.Context,
	a *auth.AuthIdentity,
	addressID uuid.UUID,
) (*cmcontact.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ServiceAgentContactAddressDeleteIndependent",
		"address_id": addressID,
	})

	agent, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return nil, err
	}

	// Tenant isolation: AddressGet scopes by customerID
	addr, err := h.reqHandler.ContactV1ContactAddressGet(ctx, agent.CustomerID, addressID)
	if err != nil {
		log.Infof("Could not get contact address. err: %v", err)
		return nil, err
	}

	if addr.CustomerID != agent.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ContactAddressDelete(ctx, agent.CustomerID, addr.ContactID, addressID)
	if err != nil {
		log.Infof("Could not delete contact address. err: %v", err)
		return nil, err
	}

	return res, nil
}
