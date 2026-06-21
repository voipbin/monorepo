package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-agent-manager/internal/config"
	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// backfillTokenLayout matches commonutilhandler ISO8601 layout used by AgentList
// as the tm_create pagination cursor, so advancing the token stays consistent
// with how the default token is generated.
const backfillTokenLayout = "2006-01-02T15:04:05.000000Z"

// backfillPageSize is intentionally large: a wide page makes a same-microsecond
// tm_create straddle across a page boundary vanishingly unlikely, and the count
// reconciliation turns "unlikely" into "verified".
const backfillPageSize = 1000

// addressChange records one agent whose stored addresses are not yet canonical.
type addressChange struct {
	agentID    uuid.UUID
	original   []commonaddress.Address
	normalized []commonaddress.Address
}

func cmdNormalizeAddresses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "normalize-addresses",
		Short: "One-time backfill: rewrite every agent's stored addresses to canonical form",
		Long: "Scans all non-deleted agents, normalizes each address target via the shared " +
			"commonaddress.NormalizeTarget (the same function the store/lookup paths use, so there " +
			"is zero drift), and rewrites the agents whose stored values are not yet canonical.\n\n" +
			"MUST be run inside a maintenance window with the agent-manager and call-manager RPC " +
			"consumers stopped (see the design doc): the backfill assumes no concurrent agent " +
			"mutation or by-address lookup.\n\n" +
			"Defaults to --dry-run=true (preview only). Re-run with --dry-run=false to apply. " +
			"The apply pass hard-fails if any canonical collision is detected; resolve the duplicate " +
			"agents first, then re-run.",
		RunE: runNormalizeAddresses,
	}

	cmd.Flags().Bool("dry-run", true, "Preview changes without writing (default true)")

	return cmd
}

func runNormalizeAddresses(cmd *cobra.Command, args []string) error {
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

	// 1. total non-deleted agent count (for completeness reconciliation)
	total, err := countNonDeletedAgents(ctx, sqlDB)
	if err != nil {
		return errors.Wrap(err, "could not count agents")
	}

	// 2. full scan: collect changes + a global collision map, count visited
	changes, collisions, visited, err := scanAgents(ctx, db)
	if err != nil {
		return errors.Wrap(err, "could not scan agents")
	}

	// 3. completeness reconciliation: a page-boundary skip would leave an agent
	// raw and route-miss forever; fail loudly rather than silently miss it.
	if visited != total {
		return fmt.Errorf("scan completeness check failed: visited %d agents but %d non-deleted agents exist; aborting (no writes performed)", visited, total)
	}

	// 4. always report
	fmt.Printf("mode: %s\n", dryRunLabel(dryRun))
	fmt.Printf("agents scanned: %d (reconciled with table count %d)\n", visited, total)
	fmt.Printf("agents needing normalization: %d\n", len(changes))
	reportChanges(changes)
	reportCollisions(collisions)

	// 5. dry-run stops here
	if dryRun {
		fmt.Println("\ndry-run: no changes written. Re-run with --dry-run=false to apply.")
		return nil
	}

	// 6. collision gate: hard-fail, no override. Committing a collision would
	// leave two agents owning one canonical target -> nondeterministic routing.
	if len(collisions) > 0 {
		return fmt.Errorf("aborting apply: %d canonical collision(s) detected; resolve the duplicate agents, then re-run (no writes performed)", len(collisions))
	}

	// 7. apply: dump original then write canonical for each changed agent
	applied := 0
	for _, c := range changes {
		fmt.Printf("apply agent_id=%s original=%s\n", c.agentID, formatAddresses(c.original))
		if errSet := db.AgentSetAddresses(ctx, c.agentID, c.normalized); errSet != nil {
			return errors.Wrapf(errSet, "could not set addresses for agent %s (applied %d before failure)", c.agentID, applied)
		}
		applied++
	}

	fmt.Printf("\napplied: %d agent(s) normalized.\n", applied)
	return nil
}

// countNonDeletedAgents returns the number of non-deleted agents directly,
// for the scan completeness reconciliation.
func countNonDeletedAgents(ctx context.Context, sqlDB *sql.DB) (int, error) {
	var n int
	row := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_agents WHERE tm_delete IS NULL")
	if err := row.Scan(&n); err != nil {
		return 0, errors.Wrap(err, "could not scan count")
	}
	return n, nil
}

// scanAgents walks every non-deleted agent via the existing AgentList token
// pagination, building the change set and a GLOBAL collision map keyed on
// (customer_id, type, canonical_target) populated from every agent's
// post-normalization addresses (so cross-page and already-canonical collisions
// are caught).
func scanAgents(ctx context.Context, db dbhandler.DBHandler) ([]addressChange, map[string][]uuid.UUID, int, error) {
	filters := map[agent.Field]any{
		agent.FieldDeleted: false,
	}

	changes := []addressChange{}
	canonicalOwners := map[string][]uuid.UUID{}
	visited := 0

	token := "" // empty -> AgentList defaults to current time (newest first)
	for {
		agents, err := db.AgentList(ctx, backfillPageSize, token, filters)
		if err != nil {
			return nil, nil, 0, errors.Wrap(err, "could not list agents")
		}
		if len(agents) == 0 {
			break
		}

		for _, a := range agents {
			visited++

			normalized := normalizeAddresses(a.Addresses)

			// record canonical ownership for collision detection
			for _, addr := range normalized {
				key := collisionKey(a.CustomerID, addr)
				canonicalOwners[key] = append(canonicalOwners[key], a.ID)
			}

			if addressesDiffer(a.Addresses, normalized) {
				changes = append(changes, addressChange{
					agentID:    a.ID,
					original:   a.Addresses,
					normalized: normalized,
				})
			}
		}

		if len(agents) < backfillPageSize {
			break
		}

		// advance the cursor to the oldest tm_create of this page
		last := agents[len(agents)-1]
		if last.TMCreate == nil {
			// defensive: cannot advance safely; stop to avoid an infinite loop
			break
		}
		token = last.TMCreate.UTC().Format(backfillTokenLayout)
	}

	// reduce the ownership map to genuine collisions (a canonical owned by >1 agent)
	collisions := map[string][]uuid.UUID{}
	for key, owners := range canonicalOwners {
		uniqueOwners := dedupeUUIDs(owners)
		if len(uniqueOwners) > 1 {
			collisions[key] = uniqueOwners
		}
	}

	return changes, collisions, visited, nil
}

// normalizeAddresses returns a normalized copy of addresses (per element, keyed
// on each element's own type). Loss-proof: the error is discarded because the
// first return value is always safe to store.
func normalizeAddresses(addresses []commonaddress.Address) []commonaddress.Address {
	out := make([]commonaddress.Address, len(addresses))
	copy(out, addresses)
	for i := range out {
		out[i].Target, _ = commonaddress.NormalizeTarget(out[i].Type, out[i].Target)
	}
	return out
}

func addressesDiffer(a, b []commonaddress.Address) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i].Type != b[i].Type || a[i].Target != b[i].Target {
			return true
		}
	}
	return false
}

func collisionKey(customerID uuid.UUID, addr commonaddress.Address) string {
	return fmt.Sprintf("%s\x00%s\x00%s", customerID, addr.Type, addr.Target)
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	out := []uuid.UUID{}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func dryRunLabel(dryRun bool) string {
	if dryRun {
		return "dry-run (preview only)"
	}
	return "apply"
}

func reportChanges(changes []addressChange) {
	if len(changes) == 0 {
		return
	}
	fmt.Println("\nplanned changes (agent_id: type raw -> canonical):")
	for _, c := range changes {
		for i := range c.original {
			if c.original[i].Type == c.normalized[i].Type && c.original[i].Target == c.normalized[i].Target {
				continue
			}
			fmt.Printf("  %s: %s %q -> %q\n", c.agentID, c.original[i].Type, c.original[i].Target, c.normalized[i].Target)
		}
	}
}

func reportCollisions(collisions map[string][]uuid.UUID) {
	if len(collisions) == 0 {
		fmt.Println("\ncollisions: none")
		return
	}
	fmt.Printf("\ncollisions: %d (a canonical (customer,type,target) owned by >1 agent)\n", len(collisions))

	keys := make([]string, 0, len(collisions))
	for k := range collisions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts := strings.SplitN(k, "\x00", 3)
		owners := collisions[k]
		ownerStrs := make([]string, len(owners))
		for i, o := range owners {
			ownerStrs[i] = o.String()
		}
		fmt.Printf("  customer=%s type=%s target=%q owners=[%s]\n", parts[0], parts[1], parts[2], strings.Join(ownerStrs, ", "))
	}
}

func formatAddresses(addresses []commonaddress.Address) string {
	parts := make([]string, len(addresses))
	for i, a := range addresses {
		parts[i] = fmt.Sprintf("{%s:%s}", a.Type, a.Target)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
