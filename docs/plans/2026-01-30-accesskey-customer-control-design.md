# Accesskey Resource Control in Customer-Control CLI

**Date:** 2026-01-30
**Status:** Approved

## Overview

Add accesskey CRUD commands to the `customer-control` CLI tool, enabling direct database management of accesskeys without going through RabbitMQ RPC.

## Command Structure

```
customer-control accesskey create   # Create new accesskey
customer-control accesskey get      # Get accesskey by ID
customer-control accesskey list     # List accesskeys with filters
customer-control accesskey update   # Update name/detail
customer-control accesskey delete   # Soft-delete accesskey
```

Command group registration:
```go
cmdAccesskey := &cobra.Command{Use: "accesskey", Short: "Accesskey operation"}
cmdAccesskey.AddCommand(cmdAccesskeyCreate())
cmdAccesskey.AddCommand(cmdAccesskeyGet())
cmdAccesskey.AddCommand(cmdAccesskeyList())
cmdAccesskey.AddCommand(cmdAccesskeyUpdate())
cmdAccesskey.AddCommand(cmdAccesskeyDelete())
rootCmd.AddCommand(cmdAccesskey)
```

## Handler Initialization

```go
func initAccesskeyHandler() (accesskeyhandler.AccesskeyHandler, error) {
    db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
    if err != nil {
        return nil, err
    }

    cache, err := initCache()
    if err != nil {
        return nil, err
    }

    dbHandler := dbhandler.NewHandler(db, cache)

    sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
    if err := sockHandler.Connect(); err != nil {
        return nil, err
    }

    reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
    notifyHandler := notifyhandler.NewNotifyHandler(
        sockHandler, reqHandler, commonoutline.QueueNameCustomerEvent, serviceName)

    return accesskeyhandler.NewAccesskeyHandler(reqHandler, dbHandler, notifyHandler), nil
}
```

## Command Flags

### `accesskey create`
| Flag | Required | Description |
|------|----------|-------------|
| `--customer-id` | Yes | UUID of the customer |
| `--name` | Yes | Name of the accesskey |
| `--detail` | No | Description/detail |
| `--expire` | No | Expiration duration (e.g., "720h" for 30 days) |

### `accesskey get`
| Flag | Required | Description |
|------|----------|-------------|
| `--id` | Yes | UUID of the accesskey |

### `accesskey list`
| Flag | Required | Description |
|------|----------|-------------|
| `--customer-id` | No | Filter by customer UUID |
| `--size` | No | Number of results (default: 10) |
| `--token` | No | Pagination token |

### `accesskey update`
| Flag | Required | Description |
|------|----------|-------------|
| `--id` | Yes | UUID of the accesskey |
| `--name` | No | New name |
| `--detail` | No | New detail |

### `accesskey delete`
| Flag | Required | Description |
|------|----------|-------------|
| `--id` | Yes | UUID of the accesskey |

## Implementation

### File to Modify
- `bin-customer-manager/cmd/customer-control/main.go`

### Changes Required

1. Add import for `accesskeyhandler` package
2. Add `initAccesskeyHandler()` function
3. Add 5 command functions with their run handlers:
   - `cmdAccesskeyCreate()` + `runAccesskeyCreate()`
   - `cmdAccesskeyGet()` + `runAccesskeyGet()`
   - `cmdAccesskeyList()` + `runAccesskeyList()`
   - `cmdAccesskeyUpdate()` + `runAccesskeyUpdate()`
   - `cmdAccesskeyDelete()` + `runAccesskeyDelete()`
4. Register commands in `main()` under accesskey subcommand group

### Output Format
All commands output JSON using existing `printJSON()` helper, consistent with customer commands.
