# Database Patterns Checklist

> **Quick Reference:** Essential database patterns and gotchas to verify when working with models and database operations.

## Pre-Commit Checklist

Use this checklist when adding or modifying database models:

### Model Definition

- [ ] All `uuid.UUID` fields have `,uuid` db tag
- [ ] Soft delete field: `TMDelete string \`db:"tm_delete"\``
- [ ] All fields have appropriate `db:"column_name"` tags
- [ ] JSON tags match API expectations

### Database Operations

- [ ] Use Squirrel query builder (not raw SQL)
- [ ] Active record check: `WHERE tm_delete = ?` with `DefaultTimeStamp`
- [ ] UUID fields converted to binary for MySQL

## Critical: UUID Field Tags

**ALWAYS add `,uuid` to uuid.UUID fields:**

```go
// CORRECT
type Call struct {
    ID           uuid.UUID `json:"id" db:"id,uuid"`
    CustomerID   uuid.UUID `json:"customer_id" db:"customer_id,uuid"`
    FlowID       uuid.UUID `json:"flow_id" db:"flow_id,uuid"`
    Name         string    `json:"name" db:"name"`
    TMCreate     string    `json:"tm_create" db:"tm_create"`
    TMUpdate     string    `json:"tm_update" db:"tm_update"`
    TMDelete     string    `json:"tm_delete" db:"tm_delete"`
}

// WRONG - Missing ,uuid tag
type Call struct {
    ID           uuid.UUID `json:"id" db:"id"`           // BUG: queries will fail
    CustomerID   uuid.UUID `json:"customer_id" db:"customer_id"` // BUG
}
```

### Why This Matters

1. `commondatabasehandler.PrepareFields()` needs `,uuid` to convert UUID → binary
2. Without it, UUIDs are passed as strings → no database matches
3. Results in silent failures: list endpoints return empty arrays

### Symptoms of Missing UUID Tag

- GET with filters returns `[]` when data exists
- POST works but GET by ID fails
- No errors in logs, just empty results

## Soft Delete Pattern

All tables use soft deletes with `tm_delete` timestamp:

```go
// Active records have this timestamp
const DefaultTimeStamp = "9999-01-01 00:00:00.000000"

// Querying active records
query := sq.Select("*").
    From("call_calls").
    Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp})

// Soft delete
query := sq.Update("call_calls").
    Set("tm_delete", time.Now().Format("2006-01-02 15:04:05.000000")).
    Where(sq.Eq{"id": id})
```

## Squirrel Query Builder

### SELECT

```go
query := sq.Select(
    "id",
    "customer_id",
    "name",
    "tm_create",
).From("call_calls").
    Where(sq.Eq{"id": id}).
    Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp})

sql, args, err := query.ToSql()
```

### INSERT

```go
query := sq.Insert("call_calls").
    Columns("id", "customer_id", "name", "tm_create", "tm_update", "tm_delete").
    Values(
        id.Bytes(),
        customerID.Bytes(),
        name,
        now,
        now,
        databasehandler.DefaultTimeStamp,
    )
```

### UPDATE

```go
query := sq.Update("call_calls").
    Set("name", newName).
    Set("tm_update", time.Now().Format("2006-01-02 15:04:05.000000")).
    Where(sq.Eq{"id": id.Bytes()}).
    Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp})
```

### DELETE (Soft)

```go
query := sq.Update("call_calls").
    Set("tm_delete", time.Now().Format("2006-01-02 15:04:05.000000")).
    Where(sq.Eq{"id": id.Bytes()})
```

## Filter Handling

### Using ApplyFields

```go
// From bin-common-handler/pkg/databasehandler/
filters := map[string]any{
    "customer_id": customerID,  // uuid.UUID
    "status":      "active",    // string
}

query := sq.Select("*").From("call_calls")
query = databasehandler.ApplyFields(query, filters)
```

### Manual Filter Application

```go
query := sq.Select("*").From("call_calls")

if customerID, ok := filters["customer_id"].(uuid.UUID); ok {
    query = query.Where(sq.Eq{"customer_id": customerID.Bytes()})
}

if status, ok := filters["status"].(string); ok {
    query = query.Where(sq.Eq{"status": status})
}
```

## Pagination Pattern

```go
const (
    DefaultPageSize = 100
    MaxPageSize     = 1000
)

func (h *dbHandler) List(ctx context.Context, size, token int, filters map[string]any) ([]*Model, error) {
    if size <= 0 || size > MaxPageSize {
        size = DefaultPageSize
    }

    query := sq.Select("*").
        From("table_name").
        Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp}).
        OrderBy("tm_create DESC").
        Limit(uint64(size)).
        Offset(uint64(token))

    query = databasehandler.ApplyFields(query, filters)
    // ...
}
```

## Timestamp Format

```go
// Standard format for MySQL DATETIME(6)
const TimeFormat = "2006-01-02 15:04:05.000000"

now := time.Now().Format(TimeFormat)
```

## Common Database Queries

### Get by ID

```go
func (h *dbHandler) Get(ctx context.Context, id uuid.UUID) (*Model, error) {
    query := sq.Select("*").
        From("table_name").
        Where(sq.Eq{"id": id.Bytes()}).
        Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp})

    sql, args, err := query.ToSql()
    if err != nil {
        return nil, errors.Wrap(err, "could not build query")
    }

    row := h.db.QueryRowContext(ctx, sql, args...)
    // scan row...
}
```

### Check Existence

```go
func (h *dbHandler) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
    query := sq.Select("COUNT(*)").
        From("table_name").
        Where(sq.Eq{"id": id.Bytes()}).
        Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp})

    var count int
    // execute and scan count
    return count > 0, nil
}
```

### Get by Unique Field

```go
func (h *dbHandler) GetByName(ctx context.Context, customerID uuid.UUID, name string) (*Model, error) {
    query := sq.Select("*").
        From("table_name").
        Where(sq.Eq{"customer_id": customerID.Bytes()}).
        Where(sq.Eq{"name": name}).
        Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp})
    // ...
}
```

## Scanning Results

### Single Row

```go
var m Model
err := row.Scan(
    &m.ID,
    &m.CustomerID,
    &m.Name,
    &m.TMCreate,
    &m.TMUpdate,
    &m.TMDelete,
)
```

### Multiple Rows

```go
rows, err := h.db.QueryContext(ctx, sql, args...)
if err != nil {
    return nil, err
}
defer rows.Close()

var results []*Model
for rows.Next() {
    var m Model
    if err := rows.Scan(&m.ID, &m.Name); err != nil {
        return nil, err
    }
    results = append(results, &m)
}

if err := rows.Err(); err != nil {
    return nil, err
}
return results, nil
```

## Transaction Pattern

```go
func (h *dbHandler) CreateWithRelated(ctx context.Context, model *Model) error {
    tx, err := h.db.BeginTx(ctx, nil)
    if err != nil {
        return errors.Wrap(err, "could not begin transaction")
    }
    defer tx.Rollback()

    // Insert main record
    if err := h.insertModel(ctx, tx, model); err != nil {
        return err
    }

    // Insert related records
    if err := h.insertRelated(ctx, tx, model.ID, model.Related); err != nil {
        return err
    }

    return tx.Commit()
}
```

## Debugging Database Issues

### Enable Query Logging

```go
sql, args, _ := query.ToSql()
log.WithFields(log.Fields{
    "sql":  sql,
    "args": args,
}).Debug("Executing query")
```

### Check UUID Conversion

```go
// Verify UUID is being converted to bytes
id := uuid.FromStringOrNil("...")
log.Printf("UUID bytes: %v", id.Bytes())
```

### Verify Soft Delete Filter

```go
// Ensure tm_delete check is present
log.Printf("Query: %s", sql)
// Should contain: tm_delete = '9999-01-01 00:00:00.000000'
```

## See Also

- [Code Quality Standards](code-quality-standards.md#uuid-fields-and-db-tags) - UUID gotcha details
- [Common Workflows](common-workflows.md) - Adding new models
- `bin-common-handler/pkg/databasehandler/` - Utility functions
