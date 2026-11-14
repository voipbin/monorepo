package providerhandler

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/envsecret"
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

	// hasSecret is true when the secret variable of the provider holds a value.
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
// The secret comes from the environment only (the stored ciphertext is not read). A provider
// without a secret has the source none and is not usable. The config is the config of the row,
// where a key without a value is filled from its environment variable.
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

	secret, state := envsecret.Lookup(desc.EnvSecret)
	if state != envsecret.Set {
		res.source = provider.CredentialSourceNone
		res.usable = false
		h.observeUnusable(res, "no_secret")

		return res, nil
	}

	res.source = provider.CredentialSourceEnv
	res.hasSecret = true
	res.creds = provider.Credentials{Secret: secret, Config: envFilledConfig(desc, row)}

	if key, missing := requiredConfigMissing(desc, res.creds.Config); missing {
		log.Errorf("The provider misses a required config. key: %s", key)
		res.usable = false
		h.observeUnusable(res, "required_config")

		return res, nil
	}

	res.usable = res.enabled

	return res, nil
}

// envFilledConfig returns a new config: the config of the row, where a key without a value is
// filled by its environment variable. The row is never written.
func envFilledConfig(desc Descriptor, row *provider.Provider) map[string]string {
	config := map[string]string{}
	if row != nil {
		for k, v := range row.Config {
			config[k] = v
		}
	}

	for key, envName := range desc.EnvConfig {
		if config[key] != "" {
			continue
		}
		if v, state := envsecret.Lookup(envName); state == envsecret.Set {
			config[key] = v
		}
	}

	return config
}

// LogStates logs once the state of each credential variable: set, unset or placeholder. It never
// logs a value.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	for _, name := range credentialEnvNames() {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"env": name, "state": state.String()})
		if state == envsecret.Placeholder {
			entry.Warn("The credential variable holds an unresolved placeholder, so it is treated as not set.")
			continue
		}
		entry.Info("The credential variable state.")
	}
}

// credentialEnvNames returns the credential variable names of the registry in a stable order: per
// provider (by name) the secret variable, then the config variables (by config key).
func credentialEnvNames() []string {
	registry := newRegistry(nil, nil)

	providers := make([]string, 0, len(registry))
	for name := range registry {
		providers = append(providers, string(name))
	}
	sort.Strings(providers)

	res := []string{}
	for _, name := range providers {
		desc := registry[number.ProviderName(name)]
		res = append(res, desc.EnvSecret)

		keys := make([]string, 0, len(desc.EnvConfig))
		for k := range desc.EnvConfig {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			res = append(res, desc.EnvConfig[k])
		}
	}

	return res
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
	for name, desc := range h.registry {
		// a provider that cannot purchase is never resolved here: no lookup and no unusable count
		if !desc.Purchasable {
			continue
		}

		r, err := h.resolve(ctx, name)
		if err != nil {
			return nil, err
		}
		if !r.usable {
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
			// one provider that cannot be created must not stop the purchase at the others
			logrus.WithField("provider_name", r.name).Errorf("Could not create the provider. err: %v", err)
			h.observeUnusable(r, "factory")
			continue
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
