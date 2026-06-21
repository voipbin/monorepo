package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-agent-manager/internal/config"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// agentAddressBackfill is one agent's address set read straight from the legacy
// agent_agents.addresses JSON column (the source of truth during the EXPAND
// phase, before the child table is populated).
type agentAddressBackfill struct {
	agentID    uuid.UUID
	customerID uuid.UUID
	addresses  []commonaddress.Address
}

func cmdBackfillAddresses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backfill-addresses",
		Short: "One-time backfill: populate agent_addresses from the legacy agent_agents.addresses JSON column",
		Long: "Reads every non-deleted agent's addresses directly from the legacy " +
			"agent_agents.addresses JSON column and writes them into the normalized " +
			"agent_addresses child table via the same store path the live code uses " +
			"(db.AgentSetAddresses), so there is zero drift. The operation is idempotent: " +
			"it replaces (delete-then-insert) each agent's child rows, so it is safe to " +
			"re-run.\n\n" +
			"This is the EXPAND-phase populate step. The read paths already source addresses " +
			"from the (initially empty) child table, so until this backfill completes, agents " +
			"appear to have no addresses. Run it immediately after deploying the migration + " +
			"the agent-manager image.\n\n" +
			"MUST be run inside a maintenance window with the agent-manager and call-manager " +
			"RPC consumers stopped: the backfill assumes no concurrent agent mutation or " +
			"by-address lookup.\n\n" +
			"Defaults to --dry-run=true (preview only). Re-run with --dry-run=false to apply. " +
			"The apply pass hard-fails if any canonical (customer_id, type, target) collision " +
			"is detected (two agents owning one address): resolve the duplicate agents first, " +
			"then re-run, otherwise the later UNIQUE-index promotion would fail.\n\n" +
			"Note: applying bumps each migrated agent's tm_update (the store path stamps it).",
		RunE: runBackfillAddresses,
	}

	cmd.Flags().Bool("dry-run", true, "Preview changes without writing (default true)")

	return cmd
}

func runBackfillAddresses(cmd *cobra.Command, args []string) error {
	dryRun := viper.GetBool("dry-run")

	ctx := context.Background()

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrap(err, "could not connect to the database")
	}
	defer func() { _ = sqlDB.Close() }()

	cache := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := cache.Connect(); errConnect != nil {
		return errors.Wrap(errConnect, "could not connect to the cache")
	}

	db := dbhandler.NewHandler(sqlDB, cache)

	// 1. total non-deleted agent count, for the completeness reconciliation.
	total, err := countNonDeletedAgents(ctx, sqlDB)
	if err != nil {
		return errors.Wrap(err, "could not count agents")
	}

	// 2. read every non-deleted agent's legacy JSON addresses into memory FIRST
	// (fully draining + closing the read) before issuing any write. Holding the
	// read rows open while writing would block on a single-connection pool, the
	// same deadlock class fixed in the dbhandler read path.
	records, visited, err := readLegacyAddresses(ctx, sqlDB)
	if err != nil {
		return errors.Wrap(err, "could not read legacy addresses")
	}

	// 3. completeness reconciliation: never write on a short/over read.
	if visited != total {
		return fmt.Errorf("read %d agents but %d non-deleted agents exist; aborting (no writes performed)", visited, total)
	}

	// 4. build the collision map from the canonical (customer,type,target) of
	// every agent's addresses, so a duplicate that would later break the UNIQUE
	// promotion is caught up front.
	collisions := detectCollisions(records)

	// 5. report
	fmt.Printf("mode: %s\n", dryRunLabel(dryRun))
	fmt.Printf("agents scanned: %d (reconciled with table count %d)\n", visited, total)
	withAddrs := 0
	for _, r := range records {
		if len(r.addresses) > 0 {
			withAddrs++
		}
	}
	fmt.Printf("agents with addresses to backfill: %d\n", withAddrs)
	reportCollisions(collisions)

	// 6. dry-run stops here
	if dryRun {
		fmt.Println("\ndry-run: no changes written. Re-run with --dry-run=false to apply.")
		return nil
	}

	// 7. collision gate: hard-fail, no override.
	if len(collisions) > 0 {
		return fmt.Errorf("aborting apply: %d canonical collision(s) detected; resolve the duplicate agents, then re-run (no writes performed)", len(collisions))
	}

	// 8. apply: replace each agent's child rows via the live store path (drift-0,
	// idempotent). Agents with no addresses need no child rows and are skipped.
	applied := 0
	for _, r := range records {
		if len(r.addresses) == 0 {
			continue
		}
		if errSet := db.AgentSetAddresses(ctx, r.agentID, r.addresses); errSet != nil {
			return errors.Wrapf(errSet, "could not set addresses for agent %s (applied %d before failure)", r.agentID, applied)
		}
		applied++
	}

	fmt.Printf("\napplied: %d agent(s) backfilled.\n", applied)
	return nil
}

// readLegacyAddresses reads id, customer_id and the legacy addresses JSON for
// every non-deleted agent, fully into memory. It returns the records and the
// number of agents visited (for reconciliation). The result rows are drained
// and closed before returning, so no read cursor is held across a later write.
func readLegacyAddresses(ctx context.Context, sqlDB *sql.DB) ([]agentAddressBackfill, int, error) {
	rows, err := sqlDB.QueryContext(ctx,
		"SELECT id, customer_id, addresses FROM agent_agents WHERE tm_delete IS NULL",
	)
	if err != nil {
		return nil, 0, errors.Wrap(err, "could not query agents")
	}
	defer func() { _ = rows.Close() }()

	records := []agentAddressBackfill{}
	visited := 0
	for rows.Next() {
		var (
			idRaw         []byte
			customerIDRaw []byte
			addressesRaw  sql.NullString
		)
		if errScan := rows.Scan(&idRaw, &customerIDRaw, &addressesRaw); errScan != nil {
			return nil, 0, errors.Wrap(errScan, "could not scan agent row")
		}
		visited++

		agentID, errID := uuid.FromBytes(idRaw)
		if errID != nil {
			return nil, 0, errors.Wrap(errID, "could not parse agent id")
		}
		customerID, errCust := uuid.FromBytes(customerIDRaw)
		if errCust != nil {
			return nil, 0, errors.Wrapf(errCust, "could not parse customer id for agent %s", agentID)
		}

		addresses := []commonaddress.Address{}
		if addressesRaw.Valid && strings.TrimSpace(addressesRaw.String) != "" {
			if errJSON := json.Unmarshal([]byte(addressesRaw.String), &addresses); errJSON != nil {
				return nil, 0, errors.Wrapf(errJSON, "could not parse addresses JSON for agent %s", agentID)
			}
		}

		records = append(records, agentAddressBackfill{
			agentID:    agentID,
			customerID: customerID,
			addresses:  addresses,
		})
	}
	if errRows := rows.Err(); errRows != nil {
		return nil, 0, errors.Wrap(errRows, "rows iteration error")
	}

	return records, visited, nil
}

// detectCollisions reduces the records to the set of canonical
// (customer_id, type, target) keys owned by more than one agent. It reuses
// collisionKey / dedupeUUIDs from normalize_addresses.go so the operational
// semantics match the existing backfill tool.
func detectCollisions(records []agentAddressBackfill) map[string][]uuid.UUID {
	owners := map[string][]uuid.UUID{}
	for _, r := range records {
		for _, addr := range r.addresses {
			key := collisionKey(r.customerID, addr)
			owners[key] = append(owners[key], r.agentID)
		}
	}

	collisions := map[string][]uuid.UUID{}
	for key, ids := range owners {
		uniqueOwners := dedupeUUIDs(ids)
		if len(uniqueOwners) > 1 {
			collisions[key] = uniqueOwners
		}
	}
	return collisions
}
