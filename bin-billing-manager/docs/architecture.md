# bin-billing-manager — Architecture

## Component Overview

`bin-billing-manager` is a Class A Standard Go RPC Manager that handles all financial accounting for the VoIPbin platform. It tracks account balances, creates billing records for billable events, validates balance before resource consumption, and integrates with the Paddle payment gateway.

```
cmd/billing-manager/main.go
    ├── pkg/dbhandler         (MySQL + Redis cache)
    ├── pkg/cachehandler      (Redis)
    ├── pkg/accounthandler    (balance management, plan validation)
    ├── pkg/billinghandler    (billing record lifecycle)
    ├── pkg/failedeventhandler (retry queue for failed billing ops)
    ├── pkg/listenhandler     (RabbitMQ RPC — accounts & billings API)
    └── pkg/subscribehandler  (RabbitMQ event consumer — billable events)
```

Key supporting binaries:
- `cmd/billing-control/` — CLI tool for direct DB/cache operations, bypasses RabbitMQ RPC. All output is JSON (stdout); logs to stderr.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/billing-manager` | Config init (Viper+pflag), dependency wiring, daemon start |
| Listen | `pkg/listenhandler` | RabbitMQ RPC request routing; dispatches to accounthandler or billinghandler |
| Subscribe | `pkg/subscribehandler` | Consumes events from call/message/number/customer managers; triggers billing creation |
| Business | `pkg/accounthandler` | Account CRUD, balance add/subtract, plan-type checks, Paddle webhook processing |
| Business | `pkg/billinghandler` | Billing record creation, duration tracking, cost calculation |
| Retry | `pkg/failedeventhandler` | Persists and retries billing operations that fail downstream |
| Data | `pkg/dbhandler` | Parameterized MySQL queries for accounts, billings, failed_events |
| Cache | `pkg/cachehandler` | Redis account cache; invalidated on mutations |
| Models | `models/account` | Account, PaymentType, PaymentMethod, PlanType |
| Models | `models/billing` | Billing, ReferenceType, Status, default unit costs |
| Models | `models/failedevent` | FailedEvent, Status, Field |

## Request Routing

The `listenhandler` consumes from queue `bin-manager.billing-manager.request` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| GET | `/v1/accounts?` | List accounts |
| POST | `/v1/accounts` | Create account |
| GET | `/v1/accounts/{uuid}` | Get account |
| PUT | `/v1/accounts/{uuid}` | Update account |
| DELETE | `/v1/accounts/{uuid}` | Delete account |
| POST | `/v1/accounts/{uuid}/balance_add_force` | Force-add balance |
| POST | `/v1/accounts/{uuid}/balance_subtract_force` | Force-subtract balance |
| POST | `/v1/accounts/{uuid}/is_valid_balance` | Check balance sufficiency |
| GET | `/v1/accounts/{uuid}/is_valid_resource_limit` | Check resource limit |
| PUT | `/v1/accounts/{uuid}/payment_info` | Update payment info |
| GET | `/v1/accounts/{uuid}/paddle_portal_session` | Get Paddle billing portal URL |
| POST | `/v1/hooks/paddle` | Process Paddle subscription/transaction webhook |
| GET | `/v1/billings?` | List billing records |
| GET | `/v1/billings/{uuid}` | Get billing record |
| POST | `/v1/accounts/is_valid_balance_by_customer_id` | Check balance by customer ID |
| GET | `/v1/accounts/is_valid_resource_limit_by_customer_id` | Check resource limit by customer ID |
