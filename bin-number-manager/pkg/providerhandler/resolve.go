package providerhandler

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
)

// resolved is the state of a provider after merging the row, the defaults and the credentials.
type resolved struct {
	name number.ProviderName
	desc Descriptor

	row *provider.Provider // nil if there is no row

	enabled  bool
	priority int

	source provider.CredentialSource
	creds  provider.Credentials

	// hasSecret is true when the credential has a secret (from the DB or the environment).
	hasSecret bool

	// usable is true when the provider is enabled and its credentials are complete.
	usable bool
}

// hasDBSecret returns true if the row holds an encrypted secret.
func hasDBSecret(row *provider.Provider) bool {
	return row != nil && len(row.SecretCiphertext) > 0
}

// requiredConfigMissing returns the first required config key the config does not have.
func requiredConfigMissing(desc Descriptor, config map[string]string) (string, bool) {
	for _, k := range desc.RequiredConfig {
		if config[k] == "" {
			return k, true
		}
	}
	return "", false
}

// resolve merges the registry defaults, the row and the credentials of the provider.
//
// The credential source is atomic: with a secret in the row, the secret and the config come
// from the row only; without it, both come from the environment only. A row secret that cannot
// be decrypted never falls back to the environment, so the provider turns unusable instead of
// silently switching to another account.
func (h *providerHandler) resolve(ctx context.Context, name number.ProviderName) (*resolved, error) {
	desc, ok := h.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider. provider_name: %s: %w", name, dbhandler.ErrNotFound)
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "resolve",
		"provider_name": name,
	})

	row, err := h.db.NumberProviderGet(ctx, name)
	if err != nil {
		if !errors.Is(err, dbhandler.ErrNotFound) {
			// never treat a read error as "no row": the defaults could enable a disabled provider
			return nil, fmt.Errorf("could not get the provider row: %w", err)
		}
		row = nil
	}

	res := &resolved{
		name:     name,
		desc:     desc,
		row:      row,
		enabled:  desc.DefaultEnabled,
		priority: desc.DefaultPriority,
	}
	if row != nil {
		res.enabled = row.Enabled
		res.priority = row.Priority
	}

	if hasDBSecret(row) {
		secret, errDecrypt := h.crypto.Decrypt(row.SecretCiphertext, row.KeyVersion, string(name))
		if errDecrypt != nil {
			log.WithField("key_version", row.KeyVersion).Errorf("Could not decrypt the provider secret. err: %v", errDecrypt)
			res.source = provider.CredentialSourceDBUndecryptable
			res.hasSecret = false
			res.usable = false
			h.observeUnusable(res, "db_undecryptable")
			return res, nil
		}

		res.source = provider.CredentialSourceDB
		res.creds = provider.Credentials{Secret: secret, Config: row.Config}
		res.hasSecret = true

		if key, missing := requiredConfigMissing(desc, row.Config); missing {
			log.Errorf("The provider row misses a required config. key: %s", key)
			res.usable = false
			h.observeUnusable(res, "required_config")
			return res, nil
		}

		res.usable = res.enabled
		return res, nil
	}

	env, ok := h.env[name]
	if !ok || env.Secret == "" {
		res.source = provider.CredentialSourceNone
		res.usable = false
		h.observeUnusable(res, "no_secret")
		return res, nil
	}

	res.source = provider.CredentialSourceEnv
	res.creds = env
	res.hasSecret = true
	res.usable = res.enabled

	return res, nil
}

// observeUnusable counts a resolution of an enabled provider that cannot be used.
func (h *providerHandler) observeUnusable(r *resolved, reason string) {
	if r.enabled {
		promProviderUnusable.WithLabelValues(string(r.name), reason).Inc()
	}
}

// PurchaseCandidates returns the providers to try for a purchase in order.
func (h *providerHandler) PurchaseCandidates(ctx context.Context) ([]*Candidate, error) {
	items := []*resolved{}
	for name := range h.registry {
		r, err := h.resolve(ctx, name)
		if err != nil {
			return nil, err
		}
		if !r.usable || !r.desc.Purchasable {
			continue
		}
		items = append(items, r)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].name < items[j].name
	})

	res := make([]*Candidate, 0, len(items))
	for _, r := range items {
		p, err := r.desc.Factory(r.creds)
		if err != nil {
			return nil, fmt.Errorf("could not create the provider. provider_name: %s: %w", r.name, err)
		}
		res = append(res, &Candidate{Name: r.name, Provider: p})
	}

	return res, nil
}

// ForRelease returns the provider for the release, tag and search operations. It needs only a
// usable secret: the enabled flag and the purchase-only required config are not checked.
func (h *providerHandler) ForRelease(ctx context.Context, name number.ProviderName) (Provider, error) {
	r, err := h.resolve(ctx, name)
	if err != nil {
		return nil, err
	}

	if !r.hasSecret {
		return nil, fmt.Errorf("the provider has no usable credential. provider_name: %s, credential_source: %s", name, r.source)
	}

	p, err := r.desc.Factory(r.creds)
	if err != nil {
		return nil, fmt.Errorf("could not create the provider. provider_name: %s: %w", name, err)
	}

	return p, nil
}

// ForSearch returns the provider for the available number search.
func (h *providerHandler) ForSearch(ctx context.Context) (Provider, error) {
	return h.ForRelease(ctx, number.ProviderNameTelnyx)
}
