# registrar-control CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a command-line tool for managing SIP extensions and trunks in bin-registrar-manager.

**Architecture:** Single-file Cobra CLI following agent-control patterns. Uses registrar-manager's extensionhandler and trunkhandler directly. Supports interactive prompts and flag-based automation with dual output formats (human-readable and JSON).

**Tech Stack:** Go, Cobra, Viper, Survey (prompts), registrar-manager handlers

---

## Task 1: Create Directory Structure and Skeleton

**Files:**
- Create: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Create directory**

```bash
mkdir -p bin-registrar-manager/cmd/registrar-control
```

**Step 2: Create skeleton main.go**

Create `bin-registrar-manager/cmd/registrar-control/main.go`:

```go
package main

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/pkg/errors"

	"monorepo/bin-registrar-manager/internal/config"
)

const serviceName = "registrar-control"

func main() {
	cmd := initCommand()
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "registrar-control",
		Short: "VoIPbin Registrar Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return errors.Wrap(err, "failed to bind flags")
			}
			return config.Bootstrap(cmd)
		},
	}

	// Extension subcommands
	cmdExtension := &cobra.Command{Use: "extension", Short: "Extension operations"}
	cmdRoot.AddCommand(cmdExtension)

	// Trunk subcommands
	cmdTrunk := &cobra.Command{Use: "trunk", Short: "Trunk operations"}
	cmdRoot.AddCommand(cmdTrunk)

	return cmdRoot
}
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds, binary created

**Step 4: Test basic execution**

```bash
./bin/registrar-control --help
```

Expected: Shows help with extension and trunk subcommands

**Step 5: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): add CLI skeleton with extension and trunk subcommands"
```

---

## Task 2: Add Initialization Functions

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add imports and handler initialization functions**

Add to `main.go` after the `initCommand()` function:

```go
func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, errors.Wrap(err, "could not connect to cache")
	}
	return res, nil
}

func initExtensionHandler() (extensionhandler.ExtensionHandler, error) {
	dbBin, err := commondatabasehandler.Connect(config.Get().DatabaseDSNBin)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bin database")
	}

	dbAsterisk, err := commondatabasehandler.Connect(config.Get().DatabaseDSNAsterisk)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to asterisk database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize cache")
	}

	db := dbhandler.NewHandler(dbBin, dbAsterisk, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)

	return extensionhandler.NewExtensionHandler(reqHandler, db, notifyHandler), nil
}

func initTrunkHandler() (trunkhandler.TrunkHandler, error) {
	dbBin, err := commondatabasehandler.Connect(config.Get().DatabaseDSNBin)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bin database")
	}

	dbAsterisk, err := commondatabasehandler.Connect(config.Get().DatabaseDSNAsterisk)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to asterisk database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize cache")
	}

	db := dbhandler.NewHandler(dbBin, dbAsterisk, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)

	return trunkhandler.NewTrunkHandler(reqHandler, db, notifyHandler), nil
}
```

**Step 2: Add required imports**

Add to imports section:

```go
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-registrar-manager/internal/config"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): add handler initialization functions"
```

---

## Task 3: Add Helper Functions

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add resolveUUID helper**

Add after handler initialization functions:

```go
func resolveUUID(flagName string, label string) (uuid.UUID, error) {
	res := uuid.FromStringOrNil(viper.GetString(flagName))
	if res == uuid.Nil {
		tmp := ""
		prompt := &survey.Input{Message: fmt.Sprintf("%s (Required):", label)}
		if err := survey.AskOne(prompt, &tmp, survey.WithValidator(survey.Required)); err != nil {
			return uuid.Nil, errors.Wrap(err, "input canceled")
		}

		res = uuid.FromStringOrNil(tmp)
		if res == uuid.Nil {
			return uuid.Nil, fmt.Errorf("invalid format for %s: '%s' is not a valid UUID", label, tmp)
		}
	}

	return res, nil
}
```

**Step 2: Add resolveString helper**

```go
func resolveString(flagName string, label string, required bool) (string, error) {
	res := viper.GetString(flagName)
	if res == "" && required {
		prompt := &survey.Input{Message: fmt.Sprintf("%s (Required):", label)}
		if err := survey.AskOne(prompt, &res, survey.WithValidator(survey.Required)); err != nil {
			return "", errors.Wrap(err, "input canceled")
		}
	}
	return res, nil
}
```

**Step 3: Add formatOutput helper**

```go
func formatOutput(data interface{}, format string) error {
	if format == "json" {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return errors.Wrap(err, "failed to marshal JSON")
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable format (caller provides specific formatting)
	return nil
}
```

**Step 4: Add confirmDelete helper**

```go
func confirmDelete(resourceType string, id uuid.UUID, details string) (bool, error) {
	if viper.GetBool("force") {
		return true, nil
	}

	fmt.Printf("\n--- %s Information ---\n", resourceType)
	fmt.Print(details)
	fmt.Println("------------------------")

	confirm := false
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Are you sure you want to delete %s %s?", resourceType, id),
	}
	if err := survey.AskOne(prompt, &confirm); err != nil {
		return false, errors.Wrap(err, "confirmation canceled")
	}

	return confirm, nil
}
```

**Step 5: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 6: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): add helper functions for UUID resolution, prompts, and output"
```

---

## Task 4: Implement Extension Create Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdExtensionCreate function**

Add after helper functions:

```go
func cmdExtensionCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new extension",
		RunE:  runExtensionCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID")
	flags.String("extension_number", "", "Extension number")
	flags.String("username", "", "Username")
	flags.String("password", "", "Password")
	flags.String("domain", "", "Domain name")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runExtensionCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	username, err := resolveString("username", "Username", true)
	if err != nil {
		return errors.Wrap(err, "failed to resolve username")
	}

	password := viper.GetString("password")
	if password == "" {
		prompt := &survey.Password{Message: "Password (Required):"}
		if err := survey.AskOne(prompt, &password, survey.WithValidator(survey.Required)); err != nil {
			return errors.Wrap(err, "failed to get password")
		}
	}

	extensionNumber := viper.GetString("extension_number")
	domain := viper.GetString("domain")

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Create(context.Background(), customerID, extensionNumber, username, password, domain)
	if err != nil {
		return errors.Wrap(err, "failed to create extension")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Println("\n--- Extension Created ---")
	fmt.Printf("ID:                %s\n", res.ID)
	fmt.Printf("Customer ID:       %s\n", res.CustomerID)
	fmt.Printf("Extension Number:  %s\n", res.ExtensionNumber)
	fmt.Printf("Username:          %s\n", res.Username)
	fmt.Printf("Domain:            %s\n", res.DomainName)
	fmt.Println("-------------------------")

	jsonData, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Println("-----------------------")

	logrus.WithField("res", res).Infof("Created extension")
	return nil
}
```

**Step 2: Register command in initCommand**

In `initCommand()`, replace the extension subcommand section:

```go
// Extension subcommands
cmdExtension := &cobra.Command{Use: "extension", Short: "Extension operations"}
cmdExtension.AddCommand(cmdExtensionCreate())
cmdRoot.AddCommand(cmdExtension)
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Test command help**

```bash
./bin/registrar-control extension create --help
```

Expected: Shows create command flags

**Step 5: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement extension create command"
```

---

## Task 5: Implement Extension Get Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdExtensionGet function**

```go
func cmdExtensionGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an extension by ID",
		RunE:  runExtensionGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Extension ID")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runExtensionGet(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Extension ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve extension ID")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	fmt.Printf("\nRetrieving Extension ID: %s...\n", id)
	res, err := handler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve extension")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Println("\n--- Extension Information ---")
	fmt.Printf("ID:                %s\n", res.ID)
	fmt.Printf("Customer ID:       %s\n", res.CustomerID)
	fmt.Printf("Extension Number:  %s\n", res.ExtensionNumber)
	fmt.Printf("Username:          %s\n", res.Username)
	fmt.Printf("Domain:            %s\n", res.DomainName)
	fmt.Println("-----------------------------")

	jsonData, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Println("-----------------------")

	return nil
}
```

**Step 2: Register command**

In `initCommand()`, update extension subcommand:

```go
cmdExtension.AddCommand(cmdExtensionCreate())
cmdExtension.AddCommand(cmdExtensionGet())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement extension get command"
```

---

## Task 6: Implement Extension List Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdExtensionList function**

```go
func cmdExtensionList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List extensions",
		RunE:  runExtensionList,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID filter")
	flags.String("domain", "", "Domain filter")
	flags.String("username", "", "Username filter")
	flags.String("extension_number", "", "Extension number filter")
	flags.Int("limit", 100, "Limit number of results")
	flags.String("token", "", "Pagination token")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runExtensionList(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Build filters
	filters := make(map[string]interface{})
	filters["customer_id"] = customerID

	if domain := viper.GetString("domain"); domain != "" {
		filters["domain"] = domain
	}
	if username := viper.GetString("username"); username != "" {
		filters["username"] = username
	}
	if extNum := viper.GetString("extension_number"); extNum != "" {
		filters["extension_number"] = extNum
	}

	limit := uint64(viper.GetInt("limit"))
	token := viper.GetString("token")

	fmt.Printf("\nRetrieving extensions... limit: %d, token: %s, filters: %v\n", limit, token, filters)
	res, err := handler.Gets(context.Background(), limit, token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve extensions")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Printf("Success! Extensions count: %d\n", len(res))
	for _, ext := range res {
		fmt.Printf(" - [%s] %s@%s (number: %s)\n", ext.ID, ext.Username, ext.DomainName, ext.ExtensionNumber)
	}

	return nil
}
```

**Step 2: Register command**

```go
cmdExtension.AddCommand(cmdExtensionList())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement extension list command"
```

---

## Task 7: Implement Extension Update Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdExtensionUpdate function**

```go
func cmdExtensionUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an extension",
		RunE:  runExtensionUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Extension ID")
	flags.String("password", "", "New password")
	flags.String("username", "", "New username")
	flags.String("extension_number", "", "New extension number")
	flags.String("domain", "", "New domain")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runExtensionUpdate(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Extension ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve extension ID")
	}

	// Check at least one update field is provided
	hasUpdate := false
	updates := make(map[string]interface{})

	if password := viper.GetString("password"); password != "" {
		updates["password"] = password
		hasUpdate = true
	}
	if username := viper.GetString("username"); username != "" {
		updates["username"] = username
		hasUpdate = true
	}
	if extNum := viper.GetString("extension_number"); extNum != "" {
		updates["extension_number"] = extNum
		hasUpdate = true
	}
	if domain := viper.GetString("domain"); domain != "" {
		updates["domain"] = domain
		hasUpdate = true
	}

	if !hasUpdate {
		return fmt.Errorf("at least one field must be provided for update: --password, --username, --extension_number, or --domain")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Update(context.Background(), id, updates)
	if err != nil {
		return errors.Wrap(err, "failed to update extension")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Println("\n--- Extension Updated ---")
	fmt.Printf("ID:                %s\n", res.ID)
	fmt.Printf("Customer ID:       %s\n", res.CustomerID)
	fmt.Printf("Extension Number:  %s\n", res.ExtensionNumber)
	fmt.Printf("Username:          %s\n", res.Username)
	fmt.Printf("Domain:            %s\n", res.DomainName)
	fmt.Println("-------------------------")

	logrus.WithField("res", res).Infof("Updated extension")
	return nil
}
```

**Step 2: Register command**

```go
cmdExtension.AddCommand(cmdExtensionUpdate())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement extension update command"
```

---

## Task 8: Implement Extension Delete Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdExtensionDelete function**

```go
func cmdExtensionDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an extension",
		RunE:  runExtensionDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Extension ID")
	flags.Bool("force", false, "Skip confirmation prompt")

	return cmd
}

func runExtensionDelete(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Extension ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve extension ID")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Get extension details for confirmation
	ext, err := handler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve extension")
	}

	details := fmt.Sprintf("ID:                %s\nCustomer ID:       %s\nExtension Number:  %s\nUsername:          %s\nDomain:            %s\n",
		ext.ID, ext.CustomerID, ext.ExtensionNumber, ext.Username, ext.DomainName)

	confirmed, err := confirmDelete("Extension", id, details)
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Println("Deletion canceled")
		return nil
	}

	fmt.Printf("\nDeleting Extension ID: %s...\n", id)
	res, err := handler.Delete(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to delete extension")
	}

	logrus.WithField("res", res).Infof("Deleted extension")
	fmt.Println("Extension deleted successfully")
	return nil
}
```

**Step 2: Register command**

```go
cmdExtension.AddCommand(cmdExtensionDelete())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement extension delete command with confirmation"
```

---

## Task 9: Implement Trunk Create Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdTrunkCreate function**

```go
func cmdTrunkCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new trunk",
		RunE:  runTrunkCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID")
	flags.String("domain", "", "Domain name")
	flags.String("name", "", "Trunk name")
	flags.String("username", "", "Username for authentication")
	flags.String("password", "", "Password for authentication")
	flags.String("allowed_ips", "", "Comma-separated list of allowed IPs")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runTrunkCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	domain, err := resolveString("domain", "Domain", true)
	if err != nil {
		return errors.Wrap(err, "failed to resolve domain")
	}

	name := viper.GetString("name")
	username := viper.GetString("username")
	password := viper.GetString("password")
	allowedIPs := viper.GetString("allowed_ips")

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Parse allowed IPs if provided
	var ipList []string
	if allowedIPs != "" {
		ipList = strings.Split(allowedIPs, ",")
		for i := range ipList {
			ipList[i] = strings.TrimSpace(ipList[i])
		}
	}

	res, err := handler.Create(context.Background(), customerID, domain, name, username, password, ipList)
	if err != nil {
		return errors.Wrap(err, "failed to create trunk")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Println("\n--- Trunk Created ---")
	fmt.Printf("ID:           %s\n", res.ID)
	fmt.Printf("Customer ID:  %s\n", res.CustomerID)
	fmt.Printf("Name:         %s\n", res.Name)
	fmt.Printf("Domain:       %s\n", res.DomainName)
	fmt.Printf("Username:     %s\n", res.Username)
	fmt.Println("---------------------")

	jsonData, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Println("-----------------------")

	logrus.WithField("res", res).Infof("Created trunk")
	return nil
}
```

**Step 2: Add strings import**

Add to imports:

```go
"strings"
```

**Step 3: Register command in initCommand**

Update trunk subcommand:

```go
// Trunk subcommands
cmdTrunk := &cobra.Command{Use: "trunk", Short: "Trunk operations"}
cmdTrunk.AddCommand(cmdTrunkCreate())
cmdRoot.AddCommand(cmdTrunk)
```

**Step 4: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 5: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement trunk create command"
```

---

## Task 10: Implement Trunk Get Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdTrunkGet function**

```go
func cmdTrunkGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a trunk by ID",
		RunE:  runTrunkGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Trunk ID")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runTrunkGet(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Trunk ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve trunk ID")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	fmt.Printf("\nRetrieving Trunk ID: %s...\n", id)
	res, err := handler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve trunk")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Println("\n--- Trunk Information ---")
	fmt.Printf("ID:           %s\n", res.ID)
	fmt.Printf("Customer ID:  %s\n", res.CustomerID)
	fmt.Printf("Name:         %s\n", res.Name)
	fmt.Printf("Domain:       %s\n", res.DomainName)
	fmt.Printf("Username:     %s\n", res.Username)
	fmt.Println("-------------------------")

	jsonData, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Println("-----------------------")

	return nil
}
```

**Step 2: Register command**

```go
cmdTrunk.AddCommand(cmdTrunkGet())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement trunk get command"
```

---

## Task 11: Implement Trunk List Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdTrunkList function**

```go
func cmdTrunkList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List trunks",
		RunE:  runTrunkList,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID filter")
	flags.String("domain", "", "Domain filter")
	flags.String("name", "", "Name filter")
	flags.Int("limit", 100, "Limit number of results")
	flags.String("token", "", "Pagination token")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runTrunkList(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Build filters
	filters := make(map[string]interface{})
	filters["customer_id"] = customerID

	if domain := viper.GetString("domain"); domain != "" {
		filters["domain"] = domain
	}
	if name := viper.GetString("name"); name != "" {
		filters["name"] = name
	}

	limit := uint64(viper.GetInt("limit"))
	token := viper.GetString("token")

	fmt.Printf("\nRetrieving trunks... limit: %d, token: %s, filters: %v\n", limit, token, filters)
	res, err := handler.Gets(context.Background(), limit, token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve trunks")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Printf("Success! Trunks count: %d\n", len(res))
	for _, trunk := range res {
		fmt.Printf(" - [%s] %s (%s)\n", trunk.ID, trunk.Name, trunk.DomainName)
	}

	return nil
}
```

**Step 2: Register command**

```go
cmdTrunk.AddCommand(cmdTrunkList())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement trunk list command"
```

---

## Task 12: Implement Trunk Update Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdTrunkUpdate function**

```go
func cmdTrunkUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a trunk",
		RunE:  runTrunkUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Trunk ID")
	flags.String("name", "", "New name")
	flags.String("domain", "", "New domain")
	flags.String("username", "", "New username")
	flags.String("password", "", "New password")
	flags.String("allowed_ips", "", "New comma-separated list of allowed IPs")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runTrunkUpdate(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Trunk ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve trunk ID")
	}

	// Check at least one update field is provided
	hasUpdate := false
	updates := make(map[string]interface{})

	if name := viper.GetString("name"); name != "" {
		updates["name"] = name
		hasUpdate = true
	}
	if domain := viper.GetString("domain"); domain != "" {
		updates["domain"] = domain
		hasUpdate = true
	}
	if username := viper.GetString("username"); username != "" {
		updates["username"] = username
		hasUpdate = true
	}
	if password := viper.GetString("password"); password != "" {
		updates["password"] = password
		hasUpdate = true
	}
	if allowedIPs := viper.GetString("allowed_ips"); allowedIPs != "" {
		ipList := strings.Split(allowedIPs, ",")
		for i := range ipList {
			ipList[i] = strings.TrimSpace(ipList[i])
		}
		updates["allowed_ips"] = ipList
		hasUpdate = true
	}

	if !hasUpdate {
		return fmt.Errorf("at least one field must be provided for update: --name, --domain, --username, --password, or --allowed_ips")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Update(context.Background(), id, updates)
	if err != nil {
		return errors.Wrap(err, "failed to update trunk")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Println("\n--- Trunk Updated ---")
	fmt.Printf("ID:           %s\n", res.ID)
	fmt.Printf("Customer ID:  %s\n", res.CustomerID)
	fmt.Printf("Name:         %s\n", res.Name)
	fmt.Printf("Domain:       %s\n", res.DomainName)
	fmt.Printf("Username:     %s\n", res.Username)
	fmt.Println("---------------------")

	logrus.WithField("res", res).Infof("Updated trunk")
	return nil
}
```

**Step 2: Register command**

```go
cmdTrunk.AddCommand(cmdTrunkUpdate())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement trunk update command"
```

---

## Task 13: Implement Trunk Delete Command

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-control/main.go`

**Step 1: Add cmdTrunkDelete function**

```go
func cmdTrunkDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a trunk",
		RunE:  runTrunkDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Trunk ID")
	flags.Bool("force", false, "Skip confirmation prompt")

	return cmd
}

func runTrunkDelete(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Trunk ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve trunk ID")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Get trunk details for confirmation
	trunk, err := handler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve trunk")
	}

	details := fmt.Sprintf("ID:           %s\nCustomer ID:  %s\nName:         %s\nDomain:       %s\nUsername:     %s\n",
		trunk.ID, trunk.CustomerID, trunk.Name, trunk.DomainName, trunk.Username)

	confirmed, err := confirmDelete("Trunk", id, details)
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Println("Deletion canceled")
		return nil
	}

	fmt.Printf("\nDeleting Trunk ID: %s...\n", id)
	res, err := handler.Delete(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to delete trunk")
	}

	logrus.WithField("res", res).Infof("Deleted trunk")
	fmt.Println("Trunk deleted successfully")
	return nil
}
```

**Step 2: Register command**

```go
cmdTrunk.AddCommand(cmdTrunkDelete())
```

**Step 3: Test build**

```bash
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control
```

Expected: Build succeeds

**Step 4: Test all commands help**

```bash
./bin/registrar-control extension --help
./bin/registrar-control trunk --help
```

Expected: All extension and trunk subcommands listed

**Step 5: Commit**

```bash
git add cmd/registrar-control/main.go
git commit -m "feat(registrar-control): implement trunk delete command with confirmation"
```

---

## Task 14: Final Testing and Documentation

**Step 1: Run verification workflow**

```bash
cd bin-registrar-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All checks pass

**Step 2: Test binary compilation**

```bash
go build -o bin/registrar-control ./cmd/registrar-control
./bin/registrar-control --help
```

Expected: Help shows extension and trunk commands

**Step 3: Update CLAUDE.md**

Add to `bin-registrar-manager/CLAUDE.md` after the "Common Commands" section:

```markdown
### Build registrar-control CLI

```bash
# From monorepo root
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control

# Test CLI
./bin/registrar-control --help
./bin/registrar-control extension --help
./bin/registrar-control trunk --help
```

### Using registrar-control

```bash
# Extension management
registrar-control extension create --customer_id <uuid> --username user1001
registrar-control extension get --id <uuid>
registrar-control extension list --customer_id <uuid>
registrar-control extension update --id <uuid> --password newpass
registrar-control extension delete --id <uuid>

# Trunk management
registrar-control trunk create --customer_id <uuid> --domain trunk.example.com
registrar-control trunk get --id <uuid>
registrar-control trunk list --customer_id <uuid>
registrar-control trunk update --id <uuid> --name "Updated Name"
registrar-control trunk delete --id <uuid> --force
```
```

**Step 4: Commit documentation**

```bash
git add CLAUDE.md
git commit -m "docs(registrar-control): add CLI usage documentation to CLAUDE.md"
```

**Step 5: Create final commit with complete implementation**

```bash
git log --oneline -15
```

Expected: Series of logical commits for each feature

---

## Completion Checklist

- [x] CLI skeleton with Cobra structure
- [x] Initialization functions for handlers
- [x] Helper functions (UUID resolution, prompts, output formatting)
- [x] Extension commands (create, get, list, update, delete)
- [x] Trunk commands (create, get, list, update, delete)
- [x] Interactive prompts for required fields
- [x] JSON output format support
- [x] Delete confirmation prompts with --force flag
- [x] Verification workflow passed
- [x] Documentation updated

## Notes

- All commands follow agent-control patterns
- Uses registrar-manager's config, handlers, and models directly
- Single main.go file keeps implementation simple
- No unit tests required (handlers already tested)
- Manual integration testing recommended with development environment
