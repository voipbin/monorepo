# bin-transfer-manager Domain

## Domain Entities

### Transfer

A transfer operation record. Stored in MySQL.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `type` | enum | `attended` or `blind` |
| `status` | enum | Current state of the transfer |
| `transferer_call_id` | UUID | Call ID of the party initiating the transfer |
| `transferee_addresses` | JSON | Destination addresses to dial |
| `groupcall_id` | UUID | Groupcall created to reach the transferee |
| `tm_create` | timestamp | Creation time |
| `tm_update` | timestamp | Last update time |
| `tm_delete` | timestamp | Soft-delete (active: `9999-01-01 00:00:00.000000`) |

## Key Business Rules

### Attended Transfer State Machine

1. **Block** (`attendedBlock`): Places existing bridge participants on hold (Music on Hold + muted input). The transferer remains connected.
2. **Execute** (`attendedExecute`): Creates a new groupcall to the transferee's addresses. Transferer speaks privately to transferee.
3. **Transferee answers**: `subscribehandler` receives `groupcall_progressing` → bridges both parties together.
4. **Complete or retrieve**: Transferer can complete (hang up their leg) or retrieve (restore original participants).
5. **Rollback** (`attendedUnblock`): Triggered on `groupcall_hangup` or transfer cancel — removes MOH and mute from original bridge participants.

### Blind Transfer State Machine

1. **Block** (`blindBlock`): Sets `FlagNoAutoLeave` on the confbridge so it survives the transferer's hangup.
2. **Execute** (`blindExecute`): Hangs up the transferer's call immediately, then creates a groupcall to the transferee.
3. **Transferee answers**: `subscribehandler` receives `groupcall_progressing` → bridges transferee with remaining participants.
4. **Rollback** (`blindUnblock`): On `groupcall_hangup` — removes `FlagNoAutoLeave` from confbridge if transfer fails.

### Event-Driven State Transitions

The transfer state machine is driven entirely by call-manager events, not by synchronous RPC responses:

| Event | Action |
|-------|--------|
| `groupcall_progressing` | Transferee answered — bridge parties together |
| `groupcall_hangup` | Transferee disconnected — trigger rollback or finalize |
| `call_hangup` | Transferer hung up — finalize or rollback depending on state |

### Soft Deletes

Active records use `tm_delete = "9999-01-01 00:00:00.000000"`. Deleted records are timestamped with actual deletion time.

### Events Published

This service publishes state change events to `bin-manager.transfer-manager.event`. No specific events are listed in the extractor — consumers include `bin-timeline-manager` for audit.
