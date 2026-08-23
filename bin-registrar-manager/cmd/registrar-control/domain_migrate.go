package main

// VOIP-1385 domain migration batch (uuid realm -> short label realm).
//
// The batch is an identity-preserving RE-PROVISION: it never calls the
// extension Create/Delete handlers (those would mint a new extension uuid,
// regenerate the direct hash, fire created/deleted webhook storms, and re-run
// billing limit checks). Per extension it rewrites ONLY the Asterisk ps_* rows
// and the registrar_sip_auths realm via dbhandler (which maintains the Redis
// mirrors), updates the registrar_extensions row in place, and publishes
// exactly ONE extension_updated event. id, extension, username, password,
// direct_id, and direct_hash stay untouched. No billing RPC is made.

import (
	"bufio"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

const (
	// migrationTokenLayout matches the common utilhandler ISO8601 layout used
	// as the tm_create pagination cursor by the list RPCs/queries.
	migrationTokenLayout = "2006-01-02T15:04:05.000000Z"

	// migrationPageSize is the customer/domain page size for the scans.
	migrationPageSize = 1000

	// migrationExtensionListLimit mirrors the customer_deleted cascade limit.
	migrationExtensionListLimit = 1000

	// migrationShortLabelLength marks an already-migrated domain row (the
	// 4-char base36 label; backfilled legacy rows carry the 36-char uuid).
	migrationShortLabelLength = 4
)

// Asterisk provisioning defaults. MUST stay identical to the (unexported)
// defaults in pkg/extensionhandler/main.go — the re-provisioned ps_* rows have
// to match what the extension-create path would write.
const (
	migrationDefaultMaxContacts    = 3
	migrationDefaultRemoveExisting = "yes"
	migrationDefaultAuthType       = "userpass"
)

// extensionMigrationRecord is one extension's before/after state in the log.
type extensionMigrationRecord struct {
	ID            uuid.UUID `json:"id"`
	Extension     string    `json:"extension"`
	OldRealm      string    `json:"old_realm"`
	NewRealm      string    `json:"new_realm"`
	OldEndpointID string    `json:"old_endpoint_id"`
	NewEndpointID string    `json:"new_endpoint_id"`
}

// migrationRecord is one customer's migration log line (the rollback source).
//
// LOG-BEFORE-EXECUTE CONTRACT: the record is built from the FULL list of
// still-pending extensions and appended (and flushed) to the log BEFORE any
// per-extension mutation starts. A crash mid-customer therefore never loses
// rollback coverage: the crashed run's record already lists every extension it
// was about to touch, and the resume run pre-logs only the ones still pending
// (the rest are covered by the earlier record). Replaying a pre-logged but
// never-migrated extension is safe — reprovisionExtension no-ops when the
// extension already carries the target realm.
type migrationRecord struct {
	CustomerID uuid.UUID                  `json:"customer_id"`
	OldLabel   string                     `json:"old_label,omitempty"`
	NewLabel   string                     `json:"new_label,omitempty"`
	OldRealm   string                     `json:"old_realm,omitempty"`
	NewRealm   string                     `json:"new_realm,omitempty"`
	Extensions []extensionMigrationRecord `json:"extensions,omitempty"`

	// Status is set only on marker lines (operator visibility); rollback
	// ignores any line with a non-empty status.
	Status string `json:"status,omitempty"`
}

// migrationStatusCompleted marks a customer's migration as fully applied
// (appended after the last extension succeeds; informational only).
const migrationStatusCompleted = "completed"

// extensionPendingForRealm reports whether the extension still needs a
// re-provision to carry targetRealm. Single source of truth shared by the
// pre-execution record builder and reprovisionExtension's idempotent skip.
func extensionPendingForRealm(ext *extension.Extension, targetRealm string) bool {
	newEndpoint := fmt.Sprintf("%s@%s", ext.Extension, targetRealm)
	return ext.Realm != targetRealm || ext.EndpointID != newEndpoint
}

// buildExtensionMigrationRecord captures the extension's TRUE current state
// (before any mutation) as the rollback entry for a move to targetRealm.
func buildExtensionMigrationRecord(ext *extension.Extension, targetRealm string) extensionMigrationRecord {
	oldRealm := ext.Realm
	if oldRealm == "" {
		oldRealm = extensionLegacyRealm(ext)
	}

	return extensionMigrationRecord{
		ID:            ext.ID,
		Extension:     ext.Extension,
		OldRealm:      oldRealm,
		NewRealm:      targetRealm,
		OldEndpointID: ext.EndpointID,
		NewEndpointID: fmt.Sprintf("%s@%s", ext.Extension, targetRealm),
	}
}

// writeMigrationLogLine appends one JSON line to the migration log and flushes
// it to disk when the writer supports Sync (os.File). The log is the rollback
// source: the line MUST be durable before the mutations it covers begin.
func writeMigrationLogLine(w io.Writer, record *migrationRecord) error {
	line, err := json.Marshal(record)
	if err != nil {
		return errors.Wrap(err, "could not marshal the migration log line")
	}

	if _, errWrite := fmt.Fprintf(w, "%s\n", line); errWrite != nil {
		return errors.Wrap(errWrite, "could not write the migration log line")
	}

	if s, ok := w.(interface{ Sync() error }); ok {
		if errSync := s.Sync(); errSync != nil {
			return errors.Wrap(errSync, "could not sync the migration log")
		}
	}

	return nil
}

// ============================================================================
// domain-migrate
// ============================================================================

func cmdDomainMigrate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain-migrate",
		Short: "Migrate customer SIP domains from uuid realms to short 4-char labels (identity-preserving re-provision)",
		Long: "Per customer: moves the registrar_customer_domains row to a fresh 4-char base36 label and " +
			"re-provisions every extension under the new realm WITHOUT touching extension identity " +
			"(id, extension, username, password, direct_id, direct_hash are untouched; exactly one " +
			"extension_updated event per extension; no billing RPC).\n\n" +
			"Idempotent and resumable: re-running skips customers/extensions that already carry the short realm.\n\n" +
			"Defaults to a dry-run preview. Pass --execute with --log <path> to apply; the JSON-lines log " +
			"is the rollback source for domain-migrate-rollback.",
		RunE: runDomainMigrate,
	}

	cmd.Flags().Bool("dry-run", true, "Preview only (default)")
	cmd.Flags().Bool("execute", false, "Apply the migration (requires --log)")
	cmd.Flags().String("customer-id", "", "Migrate only this customer (optional)")
	cmd.Flags().String("log", "", "JSON-lines log path (rollback source; required with --execute)")

	return cmd
}

func runDomainMigrate(cmd *cobra.Command, args []string) error {
	execute := viper.GetBool("execute")
	if execute && cmd.Flags().Changed("dry-run") && viper.GetBool("dry-run") {
		return fmt.Errorf("--dry-run and --execute are mutually exclusive")
	}

	var onlyCustomerID uuid.UUID
	if v := viper.GetString("customer-id"); v != "" {
		onlyCustomerID = uuid.FromStringOrNil(v)
		if onlyCustomerID == uuid.Nil {
			return fmt.Errorf("invalid format for customer-id: '%s' is not a valid UUID", v)
		}
	}

	var logWriter io.Writer
	if execute {
		logPath := viper.GetString("log")
		if logPath == "" {
			return fmt.Errorf("--log is required with --execute (the log is the rollback source)")
		}

		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return errors.Wrap(err, "could not open the log file")
		}
		defer func() { _ = f.Close() }()
		logWriter = f
	}

	deps, err := initDomainMigrationDeps()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	migrated, skipped, err := domainMigrate(context.Background(), deps.dbAst, deps.dbBin, deps.customerDomainHandler, deps.notifyHandler, execute, onlyCustomerID, logWriter, os.Stdout)
	if err != nil {
		return errors.Wrap(err, "domain migration failed")
	}

	mode := "DRY-RUN"
	if execute {
		mode = "EXECUTE"
	}
	fmt.Printf("domain-migrate [%s] done. migrated: %d, skipped(already short): %d\n", mode, migrated, skipped)

	return nil
}

// domainMigrate migrates all (or one) customers' domains to short labels.
// Returns (migrated, skipped) customer counts.
func domainMigrate(
	ctx context.Context,
	dbAst dbhandler.DBHandler,
	dbBin dbhandler.DBHandler,
	customerDomainHandler customerdomainhandler.CustomerDomainHandler,
	notifyHandler notifyhandler.NotifyHandler,
	execute bool,
	onlyCustomerID uuid.UUID,
	logWriter io.Writer,
	out io.Writer,
) (int, int, error) {
	customerIDs, err := migrationTargetCustomerIDs(ctx, dbBin, onlyCustomerID)
	if err != nil {
		return 0, 0, err
	}

	migrated := 0
	skipped := 0
	for _, customerID := range customerIDs {
		// the log line is written INSIDE migrateCustomerDomain, BEFORE the
		// per-extension re-provisioning starts (log-before-execute)
		_, skip, errMigrate := migrateCustomerDomain(ctx, dbAst, dbBin, customerDomainHandler, notifyHandler, customerID, execute, logWriter, out)
		if errMigrate != nil {
			// fail fast: interruption safety comes from per-extension idempotency —
			// a re-run converges (already-migrated customers/extensions are
			// skipped) and the pre-logged record already covers this customer
			return migrated, skipped, errors.Wrapf(errMigrate, "could not migrate customer. customer_id: %s", customerID)
		}
		if skip {
			skipped++
			continue
		}

		migrated++
	}

	return migrated, skipped, nil
}

// migrationTargetCustomerIDs returns the customer ids to migrate: the single
// requested customer, or every customer with a registrar_customer_domains row
// (the backfill guarantees full coverage before the batch runs).
func migrationTargetCustomerIDs(ctx context.Context, dbBin dbhandler.DBHandler, onlyCustomerID uuid.UUID) ([]uuid.UUID, error) {
	if onlyCustomerID != uuid.Nil {
		return []uuid.UUID{onlyCustomerID}, nil
	}

	res := []uuid.UUID{}
	token := ""
	for {
		domains, err := dbBin.CustomerDomainList(ctx, migrationPageSize, token, map[customerdomain.Field]any{})
		if err != nil {
			return nil, errors.Wrap(err, "could not list customer domains")
		}

		for _, cd := range domains {
			res = append(res, cd.CustomerID)
		}

		if uint64(len(domains)) < migrationPageSize {
			break
		}

		last := domains[len(domains)-1]
		if last.TMCreate == nil {
			return nil, fmt.Errorf("customer domain %s has a nil tm_create; cannot page safely", last.CustomerID)
		}
		token = last.TMCreate.UTC().Format(migrationTokenLayout)
	}

	return res, nil
}

// migrateCustomerDomain migrates one customer. Returns (record, skipped, error).
// Skipped means the customer already carries a short realm and every extension
// is already provisioned under it (idempotent resume).
//
// LOG-BEFORE-EXECUTE: on --execute the rollback record — covering ALL
// still-pending extensions with their true current old realms — is appended
// and flushed to logWriter BEFORE the first per-extension mutation, so an
// interrupted run never loses rollback coverage.
func migrateCustomerDomain(
	ctx context.Context,
	dbAst dbhandler.DBHandler,
	dbBin dbhandler.DBHandler,
	customerDomainHandler customerdomainhandler.CustomerDomainHandler,
	notifyHandler notifyhandler.NotifyHandler,
	customerID uuid.UUID,
	execute bool,
	logWriter io.Writer,
	out io.Writer,
) (*migrationRecord, bool, error) {
	// current domain row (backfilled uuid row, an already-short row on resume, or missing)
	oldLabel := customerID.String()
	oldRealm := ""
	var row *customerdomain.CustomerDomain

	tmp, err := dbBin.CustomerDomainGet(ctx, customerID)
	if err == nil {
		row = tmp
		oldLabel = row.DomainLabel
		oldRealm = row.Realm
	} else if !stderrors.Is(err, dbhandler.ErrNotFound) {
		return nil, false, errors.Wrap(err, "could not get customer domain")
	}

	// customer's active extensions
	filters := map[extension.Field]any{
		extension.FieldCustomerID: customerID,
		extension.FieldDeleted:    false,
	}
	exts, err := dbBin.ExtensionList(ctx, migrationExtensionListLimit, "", filters)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not list extensions")
	}

	alreadyShort := row != nil && len(row.DomainLabel) == migrationShortLabelLength
	if alreadyShort {
		// resume path: the row's realm is already the short one. Recover the
		// TRUE old realm from a not-yet-migrated extension (the row no longer
		// carries it); if every extension is done, the customer is complete.
		pending := false
		for _, e := range exts {
			if extensionPendingForRealm(e, row.Realm) {
				pending = true
				if e.Realm != "" {
					oldRealm = e.Realm
				} else {
					oldRealm = extensionLegacyRealm(e)
				}
				oldLabel = firstDomainLabel(oldRealm)
				break
			}
		}
		if !pending {
			return nil, true, nil
		}
	}

	if oldRealm == "" {
		// no domain row: post-cutover there is no computed legacy fallback, so
		// the only source for the customer's current realm is the STORED
		// extension data (stored realm, else the endpoint id's domain part)
		for _, e := range exts {
			if e.Realm != "" {
				oldRealm = e.Realm
			} else {
				oldRealm = extensionLegacyRealm(e)
			}
			if oldRealm != "" {
				oldLabel = firstDomainLabel(oldRealm)
				break
			}
		}
	}
	if oldRealm == "" && len(exts) > 0 {
		// extensions exist but none carries a recoverable realm: refuse rather
		// than write a rollback record that cannot restore them
		return nil, false, fmt.Errorf("could not determine the current realm from the stored extension data. customer_id: %s", customerID)
	}

	if !execute {
		_, _ = fmt.Fprintf(out, "[dry-run] would migrate customer. customer_id: %s, old_label: %s, old_realm: %s, extensions: %d\n", customerID, oldLabel, oldRealm, len(exts))
		return &migrationRecord{
			CustomerID: customerID,
			OldLabel:   oldLabel,
			OldRealm:   oldRealm,
		}, false, nil
	}

	// 1. move the domain row to a fresh short label (create when missing,
	//    update the backfilled row, no-op when already short). This is the
	//    only mutation preceding the log line; the record below captures the
	//    row's pre-migration label/realm, so it stays fully roll-back-able.
	domain, err := customerDomainHandler.MigrateToShortDomain(ctx, customerID)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not migrate the customer domain row")
	}

	// 2. build the rollback record from the FULL current pending list BEFORE
	//    any per-extension mutation. Extensions already on the target realm
	//    were migrated (and pre-logged) by a previous run — coverage across
	//    the accumulated log lines is complete.
	record := &migrationRecord{
		CustomerID: customerID,
		OldLabel:   oldLabel,
		NewLabel:   domain.DomainLabel,
		OldRealm:   oldRealm,
		NewRealm:   domain.Realm,
		Extensions: []extensionMigrationRecord{},
	}
	pending := []*extension.Extension{}
	for _, e := range exts {
		if !extensionPendingForRealm(e, domain.Realm) {
			continue
		}
		pending = append(pending, e)
		record.Extensions = append(record.Extensions, buildExtensionMigrationRecord(e, domain.Realm))
	}

	// 3. LOG BEFORE EXECUTE: append + flush the record so a crash during the
	//    re-provision loop below never loses rollback coverage. A customer with
	//    no previous domain state at all (no row, no extensions: oldRealm is
	//    empty) gets no rollback record — there is nothing to restore, and an
	//    empty old_realm would fail the rollback replay validation.
	if logWriter != nil && oldRealm != "" {
		if errLog := writeMigrationLogLine(logWriter, record); errLog != nil {
			return nil, false, errors.Wrap(errLog, "could not write the migration log record")
		}
	}

	// 4. identity-preserving re-provision of every pending extension
	for _, e := range pending {
		if _, errProvision := reprovisionExtension(ctx, dbAst, dbBin, notifyHandler, e, domain.Realm); errProvision != nil {
			return nil, false, errors.Wrapf(errProvision, "could not re-provision extension. extension_id: %s", e.ID)
		}
	}

	// 5. completion marker (operator visibility only; rollback ignores it)
	if logWriter != nil {
		marker := &migrationRecord{
			CustomerID: customerID,
			Status:     migrationStatusCompleted,
		}
		if errLog := writeMigrationLogLine(logWriter, marker); errLog != nil {
			return nil, false, errors.Wrap(errLog, "could not write the migration log completion marker")
		}
	}

	_, _ = fmt.Fprintf(out, "migrated customer. customer_id: %s, old_realm: %s, new_realm: %s, extensions: %d\n", customerID, oldRealm, domain.Realm, len(record.Extensions))

	return record, false, nil
}

// reprovisionExtension moves one extension to targetRealm while PRESERVING its
// identity: id, extension, username, password, direct_id, and direct_hash are
// untouched. Only the ps_* rows are deleted+created (new ids, same
// username/password), the sipauth realm is updated, and the extension row is
// updated in place. Publishes exactly ONE extension_updated event. No billing
// RPC. Returns nil when the extension already carries targetRealm (idempotent).
func reprovisionExtension(
	ctx context.Context,
	dbAst dbhandler.DBHandler,
	dbBin dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	ext *extension.Extension,
	targetRealm string,
) (*extensionMigrationRecord, error) {
	newEndpoint := fmt.Sprintf("%s@%s", ext.Extension, targetRealm)

	if !extensionPendingForRealm(ext, targetRealm) {
		// already provisioned under the target realm. This also makes the
		// rollback replay of a pre-logged but never-migrated extension a
		// clean no-op (log-before-execute safety).
		return nil, nil
	}

	oldEndpoint := ext.EndpointID
	oldRealm := ext.Realm
	if oldRealm == "" {
		oldRealm = extensionLegacyRealm(ext)
	}

	// 1. delete the OLD ps_* rows (stored ids), plus a defensive delete of the
	//    NEW ids so a re-run interrupted after the creates converges instead of
	//    failing on duplicate keys. Deleting absent rows is a no-op.
	for _, id := range []string{ext.EndpointID, newEndpoint} {
		if id == "" {
			continue
		}
		if errDelete := dbAst.AstEndpointDelete(ctx, id); errDelete != nil {
			return nil, errors.Wrapf(errDelete, "could not delete the ast endpoint. id: %s", id)
		}
	}
	for _, id := range []string{ext.AuthID, newEndpoint} {
		if id == "" {
			continue
		}
		if errDelete := dbAst.AstAuthDelete(ctx, id); errDelete != nil {
			return nil, errors.Wrapf(errDelete, "could not delete the ast auth. id: %s", id)
		}
	}
	for _, id := range []string{ext.AORID, newEndpoint} {
		if id == "" {
			continue
		}
		if errDelete := dbAst.AstAORDelete(ctx, id); errDelete != nil {
			return nil, errors.Wrapf(errDelete, "could not delete the ast aor. id: %s", id)
		}
	}

	// 2. create the NEW ps_* rows (same username/password, new realm/ids).
	//    Same shapes/defaults as the extension-create path.
	maxContacts := migrationDefaultMaxContacts
	removeExisting := migrationDefaultRemoveExisting
	aor := &astaor.AstAOR{
		ID:             &newEndpoint,
		MaxContacts:    &maxContacts,
		RemoveExisting: &removeExisting,
	}
	if errCreate := dbAst.AstAORCreate(ctx, aor); errCreate != nil {
		return nil, errors.Wrap(errCreate, "could not create the ast aor")
	}

	authType := migrationDefaultAuthType
	username := ext.Username
	password := ext.Password
	realm := targetRealm
	auth := &astauth.AstAuth{
		ID:       &newEndpoint,
		AuthType: &authType,
		Username: &username,
		Password: &password,
		Realm:    &realm,
	}
	if errCreate := dbAst.AstAuthCreate(ctx, auth); errCreate != nil {
		return nil, errors.Wrap(errCreate, "could not create the ast auth")
	}

	endpoint := &astendpoint.AstEndpoint{
		ID:   &newEndpoint,
		AORs: &newEndpoint,
		Auth: &newEndpoint,
	}
	if errCreate := dbAst.AstEndpointCreate(ctx, endpoint); errCreate != nil {
		return nil, errors.Wrap(errCreate, "could not create the ast endpoint")
	}

	// 3. sipauth realm (Kamailio byte-matches this against the From domain)
	sipFields := map[sipauth.Field]any{
		sipauth.FieldRealm: targetRealm,
	}
	if errUpdate := dbBin.SIPAuthUpdate(ctx, ext.ID, sipFields); errUpdate != nil {
		return nil, errors.Wrap(errUpdate, "could not update the sip auth realm")
	}

	// 4. purge stale ps_contacts of the OLD endpoint (row delete + contact
	//    cache invalidation). Done before the extension update so an
	//    interrupted run still purges on the re-run.
	if oldEndpoint != "" && oldEndpoint != newEndpoint {
		if errPurge := dbAst.AstContactDeleteByEndpoint(ctx, oldEndpoint); errPurge != nil {
			return nil, errors.Wrapf(errPurge, "could not purge the stale contacts. endpoint: %s", oldEndpoint)
		}
	}

	// 5. update the extension row IN PLACE: realm, domain_name, endpoint_id,
	//    aor_id, auth_id only. id/extension/username/password/direct_* untouched.
	fields := map[extension.Field]any{
		extension.FieldRealm:      targetRealm,
		extension.FieldDomainName: targetRealm,
		extension.FieldEndpointID: newEndpoint,
		extension.FieldAORID:      newEndpoint,
		extension.FieldAuthID:     newEndpoint,
	}
	if errUpdate := dbBin.ExtensionUpdate(ctx, ext.ID, fields); errUpdate != nil {
		return nil, errors.Wrap(errUpdate, "could not update the extension")
	}

	// 6. publish exactly ONE extension_updated event (no deleted/created
	//    lifecycle events, no billing re-check)
	res, err := dbBin.ExtensionGet(ctx, ext.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get the updated extension")
	}
	notifyHandler.PublishEvent(ctx, extension.EventTypeExtensionUpdated, res)

	return &extensionMigrationRecord{
		ID:            ext.ID,
		Extension:     ext.Extension,
		OldRealm:      oldRealm,
		NewRealm:      targetRealm,
		OldEndpointID: oldEndpoint,
		NewEndpointID: newEndpoint,
	}, nil
}

// ============================================================================
// domain-migrate-rollback
// ============================================================================

func cmdDomainMigrateRollback() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain-migrate-rollback",
		Short: "Roll back a domain migration by replaying the migration log in reverse",
		Long: "Reads the JSON-lines log written by domain-migrate --execute and re-provisions each " +
			"customer back to its old label/realm with the same identity-preserving mechanics " +
			"(one extension_updated event per extension, no billing RPC).\n\n" +
			"Defaults to a dry-run preview. Pass --execute to apply.",
		RunE: runDomainMigrateRollback,
	}

	cmd.Flags().String("log", "", "JSON-lines log path written by domain-migrate --execute (required)")
	cmd.Flags().Bool("execute", false, "Apply the rollback (default: dry-run preview)")

	return cmd
}

func runDomainMigrateRollback(cmd *cobra.Command, args []string) error {
	execute := viper.GetBool("execute")

	logPath := viper.GetString("log")
	if logPath == "" {
		return fmt.Errorf("--log is required")
	}

	f, err := os.Open(logPath)
	if err != nil {
		return errors.Wrap(err, "could not open the log file")
	}
	defer func() { _ = f.Close() }()

	records, err := parseMigrationLog(f)
	if err != nil {
		return errors.Wrap(err, "could not parse the migration log")
	}

	deps, err := initDomainMigrationDeps()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	rolledBack, err := domainMigrateRollback(context.Background(), deps.dbAst, deps.dbBin, deps.customerDomainHandler, deps.notifyHandler, execute, records, os.Stdout)
	if err != nil {
		return errors.Wrap(err, "domain migration rollback failed")
	}

	mode := "DRY-RUN"
	if execute {
		mode = "EXECUTE"
	}
	fmt.Printf("domain-migrate-rollback [%s] done. customers: %d\n", mode, rolledBack)

	return nil
}

// parseMigrationLog reads the JSON-lines migration log.
func parseMigrationLog(r io.Reader) ([]*migrationRecord, error) {
	res := []*migrationRecord{}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		record := &migrationRecord{}
		if err := json.Unmarshal([]byte(line), record); err != nil {
			return nil, errors.Wrapf(err, "invalid log line %d", lineNumber)
		}

		// marker/status lines (e.g. {"customer_id":..., "status":"completed"})
		// are operator information, not rollback records — ignore gracefully
		if record.Status != "" {
			continue
		}

		if record.CustomerID == uuid.Nil {
			return nil, fmt.Errorf("invalid log line %d: missing customer_id", lineNumber)
		}

		res = append(res, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "could not read the log")
	}

	return res, nil
}

// domainMigrateRollback replays the migration records in REVERSE order,
// re-provisioning each customer back to its old label/realm with the same
// identity-preserving mechanics. Idempotent: already-rolled-back extensions
// are skipped.
func domainMigrateRollback(
	ctx context.Context,
	dbAst dbhandler.DBHandler,
	dbBin dbhandler.DBHandler,
	customerDomainHandler customerdomainhandler.CustomerDomainHandler,
	notifyHandler notifyhandler.NotifyHandler,
	execute bool,
	records []*migrationRecord,
	out io.Writer,
) (int, error) {
	rolledBack := 0

	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]

		if record.OldLabel == "" || record.OldRealm == "" {
			return rolledBack, fmt.Errorf("invalid record for customer %s: missing old_label/old_realm", record.CustomerID)
		}

		if !execute {
			_, _ = fmt.Fprintf(out, "[dry-run] would roll back customer. customer_id: %s, realm: %s -> %s, extensions: %d\n", record.CustomerID, record.NewRealm, record.OldRealm, len(record.Extensions))
			rolledBack++
			continue
		}

		// restore the domain row to the old label/realm
		if _, err := customerDomainHandler.UpdateByCustomerID(ctx, record.CustomerID, record.OldLabel, record.OldRealm); err != nil {
			return rolledBack, errors.Wrapf(err, "could not restore the customer domain row. customer_id: %s", record.CustomerID)
		}

		// re-provision each extension back to its old realm
		for _, e := range record.Extensions {
			ext, err := dbBin.ExtensionGet(ctx, e.ID)
			if err != nil {
				if stderrors.Is(err, dbhandler.ErrNotFound) {
					_, _ = fmt.Fprintf(out, "warning: extension %s no longer exists; skipping\n", e.ID)
					continue
				}
				return rolledBack, errors.Wrapf(err, "could not get the extension. extension_id: %s", e.ID)
			}

			if _, errProvision := reprovisionExtension(ctx, dbAst, dbBin, notifyHandler, ext, e.OldRealm); errProvision != nil {
				return rolledBack, errors.Wrapf(errProvision, "could not roll back the extension. extension_id: %s", e.ID)
			}
		}

		_, _ = fmt.Fprintf(out, "rolled back customer. customer_id: %s, realm: %s -> %s, extensions: %d\n", record.CustomerID, record.NewRealm, record.OldRealm, len(record.Extensions))
		rolledBack++
	}

	return rolledBack, nil
}

// ============================================================================
// helpers
// ============================================================================

// extensionLegacyRealm recovers the realm of a NULL-realm extension row from
// its STORED fields only: the domain part of the stored endpoint id
// (<ext>@<realm>). Returns "" when the endpoint id carries no domain part —
// post-cutover there is no computed fallback anymore.
func extensionLegacyRealm(ext *extension.Extension) string {
	if idx := strings.LastIndex(ext.EndpointID, "@"); idx >= 0 && idx < len(ext.EndpointID)-1 {
		return ext.EndpointID[idx+1:]
	}

	return ""
}

// firstDomainLabel returns the first dot-separated label of the given realm.
func firstDomainLabel(realm string) string {
	if idx := strings.Index(realm, "."); idx > 0 {
		return realm[:idx]
	}

	return realm
}
