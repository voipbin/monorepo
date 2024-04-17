package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
)

// getSerialize returns cached serialized info.
func (h *handler) getSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(tmp), &data); err != nil {
		return err
	}
	return nil
}

// setSerialize sets the info into the cache after serialization.
func (h *handler) setSerialize(ctx context.Context, key string, data interface{}) error {
	return h.setSerializeWithExpiration(ctx, key, data, time.Hour*24)
}

// setSerialize sets the info into the cache after serialization.
func (h *handler) setSerializeWithExpiration(ctx context.Context, key string, data interface{}, expiration time.Duration) error {
	tmp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, expiration).Err(); err != nil {
		return err
	}
	return nil
}

// delKey deletes the given key from the cache.
func (h *handler) delKey(ctx context.Context, key string) error {
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}

// AstEndpointGet returns cached AstEndpoint info
func (h *handler) AstEndpointGet(ctx context.Context, id string) (*astendpoint.AstEndpoint, error) {
	key := fmt.Sprintf("ast_endpoint:%s", id)

	var res astendpoint.AstEndpoint
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AstEndpointSet sets the astendpoint.AstEndpoint info into the cache.
func (h *handler) AstEndpointSet(ctx context.Context, e *astendpoint.AstEndpoint) error {
	key := fmt.Sprintf("ast_endpoint:%s", *e.ID)

	if err := h.setSerializeWithExpiration(ctx, key, e, time.Minute*3); err != nil {
		return err
	}

	return nil
}

// AstEndpointDel deletes the astendpoint.AstEndpoint info from the cache.
func (h *handler) AstEndpointDel(ctx context.Context, id string) error {
	key := fmt.Sprintf("ast_endpoint:%s", id)

	return h.delKey(ctx, key)
}

// AstAuthGet returns cached AstAuth info
func (h *handler) AstAuthGet(ctx context.Context, id string) (*astauth.AstAuth, error) {
	key := fmt.Sprintf("ast_auth:%s", id)

	var res astauth.AstAuth
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AstAuthSet sets the astauth.AstAuth info into the cache.
func (h *handler) AstAuthSet(ctx context.Context, e *astauth.AstAuth) error {
	key := fmt.Sprintf("ast_auth:%s", *e.ID)

	if err := h.setSerializeWithExpiration(ctx, key, e, time.Minute*3); err != nil {
		return err
	}

	return nil
}

// AstAuthDel deletes the astauth.AstAuth info from the cache.
func (h *handler) AstAuthDel(ctx context.Context, id string) error {
	key := fmt.Sprintf("ast_auth:%s", id)

	return h.delKey(ctx, key)
}

// AstAORGet returns cached AstAOR info
func (h *handler) AstAORGet(ctx context.Context, id string) (*astaor.AstAOR, error) {
	key := fmt.Sprintf("ast_aor:%s", id)

	var res astaor.AstAOR
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AstAORSet sets the astaor.AstAOR info into the cache.
func (h *handler) AstAORSet(ctx context.Context, e *astaor.AstAOR) error {
	key := fmt.Sprintf("ast_aor:%s", *e.ID)

	if err := h.setSerializeWithExpiration(ctx, key, e, time.Minute*3); err != nil {
		return err
	}

	return nil
}

// AstAORDel deletes the astaor.AstAOR info from the cache.
func (h *handler) AstAORDel(ctx context.Context, id string) error {
	key := fmt.Sprintf("ast_aor:%s", id)

	return h.delKey(ctx, key)
}

// ExtensionGet returns cached Domain info
func (h *handler) ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
	key := fmt.Sprintf("registrar:extension:%s", id)

	var res extension.Extension
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ExtensionGetByEndpointID returns cached extension info of the given extension
func (h *handler) ExtensionGetByEndpointID(ctx context.Context, endpointID string) (*extension.Extension, error) {
	key := fmt.Sprintf("registrar:extension_endpoint_id:%s", endpointID)

	var res extension.Extension
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ExtensionGetByCustomerIDANDExtension returns cached extension info of the given customer id and extension
func (h *handler) ExtensionGetByCustomerIDANDExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error) {
	key := fmt.Sprintf("registrar:customer_id_extension:%s:%s", customerID, ext)

	var res extension.Extension
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ExtensionSet sets the extension info into the cache.
func (h *handler) ExtensionSet(ctx context.Context, e *extension.Extension) error {
	key := fmt.Sprintf("registrar:extension:%s", e.ID)
	if err := h.setSerialize(ctx, key, e); err != nil {
		return err
	}

	keyEndpointID := fmt.Sprintf("registrar:extension_endpoint_id:%s", e.EndpointID)
	if err := h.setSerialize(ctx, keyEndpointID, e); err != nil {
		return err
	}

	keyCustomerIDExtension := fmt.Sprintf("registrar:customer_id_extension:%s:%s", e.CustomerID, e.Extension)
	if err := h.setSerialize(ctx, keyCustomerIDExtension, e); err != nil {
		return err
	}

	return nil
}

// AstContactsGet returns cached contacts info of the given endpoint
func (h *handler) AstContactsGet(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error) {
	key := fmt.Sprintf("ast_contacts:%s", endpoint)

	var tmp []astcontact.AstContact
	if err := h.getSerialize(ctx, key, &tmp); err != nil {
		return nil, err
	}

	res := []*astcontact.AstContact{}
	for _, c := range tmp {
		res = append(res, &c)
	}

	return res, nil
}

// AstContactsSet sets the contacts info into the cache.
func (h *handler) AstContactsSet(ctx context.Context, endpoint string, contacts []*astcontact.AstContact) error {
	key := fmt.Sprintf("ast_contacts:%s", endpoint)

	if err := h.setSerializeWithExpiration(ctx, key, contacts, time.Minute*3); err != nil {
		return err
	}

	return nil
}

// AstContactsDel deletes the contacts info from the cache.
func (h *handler) AstContactsDel(ctx context.Context, endpoint string) error {
	key := fmt.Sprintf("ast_contacts:%s", endpoint)

	return h.delKey(ctx, key)
}

// TrunkSet sets the Trunk info into the cache.
func (h *handler) TrunkSet(ctx context.Context, e *trunk.Trunk) error {
	key := fmt.Sprintf("registrar:trunk:%s", e.ID)
	if err := h.setSerialize(ctx, key, e); err != nil {
		return err
	}

	keyName := fmt.Sprintf("registrar:trunk_domain_name:%s", e.DomainName)
	if err := h.setSerialize(ctx, keyName, e); err != nil {
		return err
	}

	return nil
}

// TrunkGet returns cached Trunk info
func (h *handler) TrunkGet(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {
	key := fmt.Sprintf("registrar:trunk:%s", id)

	var res trunk.Trunk
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TrunkGetByDomainName returns cached Trunk info
func (h *handler) TrunkGetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error) {
	key := fmt.Sprintf("registrar:trunk_domain_name:%s", domainName)

	var res trunk.Trunk
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TrunkDel deletes the trunk info from the cache.
func (h *handler) TrunkDel(ctx context.Context, id uuid.UUID, name string) error {
	key := fmt.Sprintf("registrar:trunk:%s", id)
	if errDel := h.delKey(ctx, key); errDel != nil {
		return errDel
	}

	keyName := fmt.Sprintf("registrar:trunk_domain_name:%s", name)
	if errDel := h.delKey(ctx, keyName); errDel != nil {
		return errDel
	}

	return nil
}
