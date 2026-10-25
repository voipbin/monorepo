package providerhandler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/secretcrypto"
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", provider.ErrInvalidArgument, fmt.Sprintf(format, args...))
}

// toInfo builds the admin view. It carries no secret material.
func (r *resolved) toInfo() *provider.Info {
	res := &provider.Info{
		ProviderName:     r.name,
		Enabled:          r.enabled,
		Priority:         r.priority,
		CredentialSource: r.source,
		Usable:           r.usable,
	}

	if r.row != nil {
		res.Config = r.row.Config
		res.HasSecret = hasDBSecret(r.row)
		res.KeyVersion = r.row.KeyVersion
		res.UpdatedBy = r.row.UpdatedBy
		res.SecretUpdatedBy = r.row.SecretUpdatedBy
		res.TMUpdate = r.row.TMUpdate
		res.TMSecretUpdate = r.row.TMSecretUpdate
	}

	return res
}

// List returns every provider known to the code, with the defaults for the ones without a row.
func (h *providerHandler) List(ctx context.Context) ([]*provider.Info, error) {
	names := make([]string, 0, len(h.registry))
	for name := range h.registry {
		names = append(names, string(name))
	}
	sort.Strings(names)

	res := make([]*provider.Info, 0, len(names))
	for _, name := range names {
		info, err := h.Get(ctx, number.ProviderName(name))
		if err != nil {
			return nil, err
		}
		res = append(res, info)
	}

	return res, nil
}

// Get returns the admin view of the provider.
func (h *providerHandler) Get(ctx context.Context, name number.ProviderName) (*provider.Info, error) {
	r, err := h.resolve(ctx, name)
	if err != nil {
		return nil, err
	}

	return r.toInfo(), nil
}

// ensureRow makes sure the row exists, creating it with the registry defaults if it is absent.
func (h *providerHandler) ensureRow(ctx context.Context, name number.ProviderName) error {
	desc, ok := h.registry[name]
	if !ok {
		return fmt.Errorf("unknown provider. provider_name: %s: %w", name, dbhandler.ErrNotFound)
	}

	row := &provider.Provider{
		ProviderName: name,
		Enabled:      desc.DefaultEnabled,
		Priority:     desc.DefaultPriority,
		Config:       map[string]string{},
	}
	if err := h.db.NumberProviderEnsure(ctx, row); err != nil {
		return fmt.Errorf("could not create the provider row: %w", err)
	}

	return nil
}

func (h *providerHandler) validateActor(actor uuid.UUID) error {
	if actor == uuid.Nil {
		return invalid("actor_id is required")
	}
	return nil
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// validateConfig checks the config keys and sizes of an update.
func validateConfig(desc Descriptor, config map[string]string) error {
	if len(config) > maxConfigKeys {
		return invalid("too many config keys")
	}

	for k, v := range config {
		if !containsString(desc.AllowedConfig, k) {
			return invalid("config key is not allowed: %s", k)
		}
		if len(k) > maxConfigKeyLen || len(v) > maxConfigValueLen {
			return invalid("config key or value is too long")
		}
	}

	return nil
}

// mergeConfig merges the changes into the config of the row: a key with an empty value is
// deleted. A row that already holds a DB secret must keep its required keys.
func mergeConfig(desc Descriptor, row *provider.Provider, changes map[string]string) (map[string]string, error) {
	merged := map[string]string{}
	for k, v := range row.Config {
		merged[k] = v
	}
	for k, v := range changes {
		if v == "" {
			delete(merged, k)
		} else {
			merged[k] = v
		}
	}

	if hasDBSecret(row) {
		if key, missing := requiredConfigMissing(desc, merged); missing {
			return nil, invalid("the config misses a required key: %s", key)
		}
	}

	return merged, nil
}

// Update changes the enabled flag, the priority and the config of the provider.
func (h *providerHandler) Update(ctx context.Context, name number.ProviderName, actor uuid.UUID, enabled *bool, priority *int, config map[string]string) (*provider.Info, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Update",
		"provider_name": name,
		"actor_id":      actor,
	})

	desc, ok := h.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider. provider_name: %s: %w", name, dbhandler.ErrNotFound)
	}
	if err := h.validateActor(actor); err != nil {
		return nil, err
	}
	if enabled == nil && priority == nil && len(config) == 0 {
		return nil, invalid("nothing to update")
	}
	if priority != nil && (*priority < 0 || *priority > maxPriority) {
		return nil, invalid("priority is out of range")
	}
	if err := validateConfig(desc, config); err != nil {
		return nil, err
	}

	if err := h.ensureRow(ctx, name); err != nil {
		return nil, err
	}
	row, err := h.db.NumberProviderGet(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("could not get the provider row: %w", err)
	}

	fields := map[provider.Field]any{
		provider.FieldUpdatedBy: actor,
	}
	changed := []string{}
	if enabled != nil {
		fields[provider.FieldEnabled] = *enabled
		changed = append(changed, "enabled")
	}
	if priority != nil {
		fields[provider.FieldPriority] = *priority
		changed = append(changed, "priority")
	}
	if len(config) > 0 {
		merged, err := mergeConfig(desc, row, config)
		if err != nil {
			return nil, err
		}

		fields[provider.FieldConfig] = merged
		changed = append(changed, "config")
	}

	if err := h.db.NumberProviderUpdate(ctx, name, fields); err != nil {
		return nil, fmt.Errorf("could not update the provider row: %w", err)
	}
	log.WithField("changed", changed).Info("The provider setting has been changed.")

	h.warnIfNoPurchaseCandidate(ctx)

	return h.Get(ctx, name)
}

// UpdateSecret stores the secret of the provider, encrypted under the latest key.
func (h *providerHandler) UpdateSecret(ctx context.Context, name number.ProviderName, actor uuid.UUID, secret string) (*provider.Info, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateSecret",
		"provider_name": name,
		"actor_id":      actor,
	})

	desc, ok := h.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider. provider_name: %s: %w", name, dbhandler.ErrNotFound)
	}
	if err := h.validateActor(actor); err != nil {
		return nil, err
	}
	// a secret pasted with a trailing newline or spaces would break the Authorization header
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, invalid("secret is required")
	}
	if len(secret) > secretcrypto.MaxPlaintextBytes {
		return nil, invalid("secret is too long")
	}

	// encrypt first, so that a missing key does not leave a row behind
	ciphertext, keyVersion, err := h.crypto.Encrypt(secret, string(name))
	if err != nil {
		return nil, err
	}

	if err := h.ensureRow(ctx, name); err != nil {
		return nil, err
	}
	row, err := h.db.NumberProviderGet(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("could not get the provider row: %w", err)
	}
	if key, missing := requiredConfigMissing(desc, row.Config); missing {
		return nil, invalid("set the config first, it misses a required key: %s", key)
	}

	fields := map[provider.Field]any{
		provider.FieldSecretCiphertext: ciphertext,
		provider.FieldKeyVersion:       keyVersion,
		provider.FieldSecretUpdatedBy:  actor,
		provider.FieldTMSecretUpdate:   h.utilHandler.TimeNow(),
	}
	if err := h.db.NumberProviderUpdate(ctx, name, fields); err != nil {
		return nil, fmt.Errorf("could not update the provider secret: %w", err)
	}
	log.WithField("key_version", keyVersion).Info("The provider secret has been changed.")

	h.warnIfNoPurchaseCandidate(ctx)

	return h.Get(ctx, name)
}

// DeleteSecret removes the DB secret of the provider, so the provider uses the environment credentials again.
func (h *providerHandler) DeleteSecret(ctx context.Context, name number.ProviderName, actor uuid.UUID) (*provider.Info, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "DeleteSecret",
		"provider_name": name,
		"actor_id":      actor,
	})

	if _, ok := h.registry[name]; !ok {
		return nil, fmt.Errorf("unknown provider. provider_name: %s: %w", name, dbhandler.ErrNotFound)
	}
	if err := h.validateActor(actor); err != nil {
		return nil, err
	}

	fields := map[provider.Field]any{
		provider.FieldSecretCiphertext: []byte(nil),
		provider.FieldKeyVersion:       0,
		provider.FieldSecretUpdatedBy:  actor,
		provider.FieldTMSecretUpdate:   h.utilHandler.TimeNow(),
	}
	if err := h.db.NumberProviderUpdate(ctx, name, fields); err != nil {
		// ErrNotFound (no row) is passed through
		if errors.Is(err, dbhandler.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("could not delete the provider secret: %w", err)
	}
	log.Info("The provider secret has been deleted.")

	h.warnIfNoPurchaseCandidate(ctx)

	return h.Get(ctx, name)
}

// warnIfNoPurchaseCandidate logs a warning if no provider can be used for a purchase after a change.
func (h *providerHandler) warnIfNoPurchaseCandidate(ctx context.Context) {
	candidates, err := h.PurchaseCandidates(ctx)
	if err != nil {
		return
	}
	if len(candidates) == 0 {
		logrus.WithField("func", "warnIfNoPurchaseCandidate").Warn("No provider is usable for a number purchase.")
	}
}
