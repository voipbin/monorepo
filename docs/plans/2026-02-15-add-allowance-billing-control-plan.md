# Add Allowance Commands to billing-control - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add 6 allowance CLI commands (get, list, process-all, ensure, add-tokens, subtract-tokens) to billing-control.

**Architecture:** New `AddTokens` and `SubtractTokens` methods on AllowanceHandler use existing `AllowanceUpdate` with `FieldTokensTotal`. No new DB methods needed. `initHandlers()` is updated to return `allowanceHandler`. All commands follow existing cobra/viper patterns with JSON output.

**Tech Stack:** Go, cobra, viper, squirrel, gomock

---

### Task 1: Add AllowanceHandler AddTokens and SubtractTokens methods

**Files:**
- Modify: `bin-billing-manager/pkg/allowancehandler/main.go` (interface, line 22)
- Modify: `bin-billing-manager/pkg/allowancehandler/allowance.go` (implementation)

**Step 1: Add interface methods in main.go**

After `ProcessAllCycles` (line 22), add:

```go
	AddTokens(ctx context.Context, accountID uuid.UUID, amount int) (*allowance.Allowance, error)
	SubtractTokens(ctx context.Context, accountID uuid.UUID, amount int) (*allowance.Allowance, error)
```

**Step 2: Add implementations in allowance.go**

After `ListByAccountID` (after line 112), add:

```go
// AddTokens adds tokens to the current cycle's total allocation.
func (h *allowanceHandler) AddTokens(ctx context.Context, accountID uuid.UUID, amount int) (*allowance.Allowance, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "AddTokens",
		"account_id": accountID,
		"amount":     amount,
	})

	cycle, err := h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not get current cycle. err: %v", err)
	}

	newTotal := cycle.TokensTotal + amount
	if err := h.db.AllowanceUpdate(ctx, cycle.ID, map[allowance.Field]any{
		allowance.FieldTokensTotal: newTotal,
	}); err != nil {
		return nil, fmt.Errorf("could not update tokens_total. err: %v", err)
	}
	log.Debugf("Added tokens. allowance_id: %s, old_total: %d, new_total: %d", cycle.ID, cycle.TokensTotal, newTotal)

	return h.db.AllowanceGet(ctx, cycle.ID)
}

// SubtractTokens subtracts tokens from the current cycle's total allocation.
func (h *allowanceHandler) SubtractTokens(ctx context.Context, accountID uuid.UUID, amount int) (*allowance.Allowance, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SubtractTokens",
		"account_id": accountID,
		"amount":     amount,
	})

	cycle, err := h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not get current cycle. err: %v", err)
	}

	newTotal := cycle.TokensTotal - amount
	if newTotal < 0 {
		return nil, fmt.Errorf("cannot subtract %d tokens: current total is %d", amount, cycle.TokensTotal)
	}

	if err := h.db.AllowanceUpdate(ctx, cycle.ID, map[allowance.Field]any{
		allowance.FieldTokensTotal: newTotal,
	}); err != nil {
		return nil, fmt.Errorf("could not update tokens_total. err: %v", err)
	}
	log.Debugf("Subtracted tokens. allowance_id: %s, old_total: %d, new_total: %d", cycle.ID, cycle.TokensTotal, newTotal)

	return h.db.AllowanceGet(ctx, cycle.ID)
}
```

**Step 3: Regenerate mocks and run full verification**

Run: `cd bin-billing-manager && go generate ./... && go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 4: Commit**

```bash
git add bin-billing-manager/pkg/allowancehandler/
git commit -m "NOJIRA-Add-allowance-billing-control

- bin-billing-manager: Add AllowanceHandler AddTokens and SubtractTokens methods"
```

---

### Task 2: Update initHandlers and add allowance subcommand group

**Files:**
- Modify: `bin-billing-manager/cmd/billing-control/main.go`

**Step 1: Update initHandlers to return allowanceHandler**

Change `initHandlers` signature (line 524) from:

```go
func initHandlers() (accounthandler.AccountHandler, billinghandler.BillingHandler, error) {
```

to:

```go
func initHandlers() (accounthandler.AccountHandler, billinghandler.BillingHandler, allowancehandler.AllowanceHandler, error) {
```

Update `initBillingHandlers` (line 546) similarly:

```go
func initBillingHandlers(sqlDB *sql.DB, cache cachehandler.CacheHandler) (accounthandler.AccountHandler, billinghandler.BillingHandler, allowancehandler.AllowanceHandler, error) {
```

and change the return statement (line 558) to:

```go
return accHandler, billHandler, allowHandler, nil
```

Update `initHandlers` body (line 535) to:

```go
return initBillingHandlers(db, cache)
```

This is already correct since it just passes through the return.

Update `initCache` error return in `initHandlers` (line 531-533):

```go
	if err != nil {
		return nil, nil, nil, err
	}
```

And the database connect error (line 528):

```go
		return nil, nil, nil, errors.Wrapf(err, "could not connect to the database")
```

**Step 2: Fix all existing callers of initHandlers**

Every existing `run*` function that calls `initHandlers()` currently uses two return values. Update them all to three:

```go
// In every runAccount* function (runAccountCreate, runAccountGet, runAccountList, etc.):
accountHandler, _, _, err := initHandlers()

// In every runBilling* function (runBillingGet, runBillingList):
_, billingHandler, _, err := initHandlers()
```

There are 11 callers total:
- `runAccountCreate` (line 301)
- `runAccountGet` (line 325)
- `runAccountList` (line 344)
- `runAccountUpdate` (line 146)
- `runAccountUpdatePaymentInfo` (line 185)
- `runAccountUpdatePlanType` (line 233)
- `runAccountDelete` (line 371)
- `runAccountAddBalance` (line 390)
- `runAccountSubtractBalance` (line 414)
- `runBillingGet` (line 469)
- `runBillingList` (line 488)

**Step 3: Add allowance subcommand group**

After the billing subcommand block (after line 76), add:

```go
	// Allowance subcommands
	cmdAllowance := &cobra.Command{Use: "allowance", Short: "Allowance operations"}
	cmdAllowance.AddCommand(cmdAllowanceGet())
	cmdAllowance.AddCommand(cmdAllowanceList())
	cmdAllowance.AddCommand(cmdAllowanceProcessAll())
	cmdAllowance.AddCommand(cmdAllowanceEnsure())
	cmdAllowance.AddCommand(cmdAllowanceAddTokens())
	cmdAllowance.AddCommand(cmdAllowanceSubtractTokens())
```

And add it to root (after line 79):

```go
	cmdRoot.AddCommand(cmdAllowance)
```

**Step 4: Add all 6 allowance command definitions and run functions**

Add to the end of main.go (before the helper functions section):

```go
// Allowance commands

func cmdAllowanceGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get current allowance cycle for an account",
		RunE:  runAllowanceGet,
	}

	flags := cmd.Flags()
	flags.String("account-id", "", "Account ID (required)")

	return cmd
}

func cmdAllowanceList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List allowance cycles for an account",
		RunE:  runAllowanceList,
	}

	flags := cmd.Flags()
	flags.String("account-id", "", "Account ID (required)")
	flags.Int("limit", 100, "Limit the number of allowance cycles to retrieve")
	flags.String("token", "", "Pagination token")

	return cmd
}

func cmdAllowanceProcessAll() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process-all",
		Short: "Create missing allowance cycles for all accounts",
		RunE:  runAllowanceProcessAll,
	}

	return cmd
}

func cmdAllowanceEnsure() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: "Ensure current allowance cycle exists for an account",
		RunE:  runAllowanceEnsure,
	}

	flags := cmd.Flags()
	flags.String("account-id", "", "Account ID (required)")

	return cmd
}

func cmdAllowanceAddTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-tokens",
		Short: "Add tokens to current allowance cycle",
		RunE:  runAllowanceAddTokens,
	}

	flags := cmd.Flags()
	flags.String("account-id", "", "Account ID (required)")
	flags.Int("amount", 0, "Number of tokens to add (required)")

	return cmd
}

func cmdAllowanceSubtractTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subtract-tokens",
		Short: "Subtract tokens from current allowance cycle",
		RunE:  runAllowanceSubtractTokens,
	}

	flags := cmd.Flags()
	flags.String("account-id", "", "Account ID (required)")
	flags.Int("amount", 0, "Number of tokens to subtract (required)")

	return cmd
}

func runAllowanceGet(cmd *cobra.Command, args []string) error {
	_, _, allowanceHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	accountID, err := resolveUUID("account-id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	res, err := allowanceHandler.GetCurrentCycle(context.Background(), accountID)
	if err != nil {
		return errors.Wrap(err, "failed to get current allowance cycle")
	}

	return printJSON(res)
}

func runAllowanceList(cmd *cobra.Command, args []string) error {
	_, _, allowanceHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	accountID, err := resolveUUID("account-id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	res, err := allowanceHandler.ListByAccountID(context.Background(), accountID, uint64(limit), token)
	if err != nil {
		return errors.Wrap(err, "failed to list allowance cycles")
	}

	return printJSON(res)
}

func runAllowanceProcessAll(cmd *cobra.Command, args []string) error {
	_, _, allowanceHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	if err := allowanceHandler.ProcessAllCycles(context.Background()); err != nil {
		return errors.Wrap(err, "failed to process all cycles")
	}

	fmt.Println(`{"status":"ok"}`)
	return nil
}

func runAllowanceEnsure(cmd *cobra.Command, args []string) error {
	accountHandler, _, allowanceHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	accountID, err := resolveUUID("account-id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	// look up the account to get customerID and planType
	acc, err := accountHandler.Get(context.Background(), accountID)
	if err != nil {
		return errors.Wrap(err, "failed to get account")
	}

	res, err := allowanceHandler.EnsureCurrentCycle(context.Background(), accountID, acc.CustomerID, acc.PlanType)
	if err != nil {
		return errors.Wrap(err, "failed to ensure allowance cycle")
	}

	return printJSON(res)
}

func runAllowanceAddTokens(cmd *cobra.Command, args []string) error {
	_, _, allowanceHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	accountID, err := resolveUUID("account-id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	amount := viper.GetInt("amount")
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	res, err := allowanceHandler.AddTokens(context.Background(), accountID, amount)
	if err != nil {
		return errors.Wrap(err, "failed to add tokens")
	}

	return printJSON(res)
}

func runAllowanceSubtractTokens(cmd *cobra.Command, args []string) error {
	_, _, allowanceHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	accountID, err := resolveUUID("account-id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	amount := viper.GetInt("amount")
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	res, err := allowanceHandler.SubtractTokens(context.Background(), accountID, amount)
	if err != nil {
		return errors.Wrap(err, "failed to subtract tokens")
	}

	return printJSON(res)
}
```

**Step 5: Run full verification**

Run: `cd bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 6: Commit**

```bash
git add bin-billing-manager/cmd/billing-control/main.go
git commit -m "NOJIRA-Add-allowance-billing-control

- bin-billing-manager: Add 6 allowance commands to billing-control CLI"
```

---

### Task 3: Pull main, check conflicts, push and create PR

**Step 1: Fetch latest main and check for conflicts**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

If conflicts: rebase, resolve, re-run verification.
If clean: proceed.

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Add-allowance-billing-control
```

PR title: `NOJIRA-Add-allowance-billing-control`

PR body:
```
Add allowance commands to billing-control CLI for managing token allowance cycles.

- bin-billing-manager: Add AllowanceHandler AddTokens and SubtractTokens methods
- bin-billing-manager: Add 6 allowance commands to billing-control (get, list, process-all, ensure, add-tokens, subtract-tokens)
```
