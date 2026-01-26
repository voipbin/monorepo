# billing-manager

Billing-manager for billing related process and webhook event notification.

## CLI Tool: billing-control

A command-line tool for managing billing accounts and viewing billing records. All output is JSON format for easy parsing and scripting.

### Build
```bash
go build -o ./bin/billing-control ./cmd/billing-control/
```

### Commands

**Account Operations:**
```bash
# Create account - returns created account JSON
billing-control account create --customer-id <uuid> [--name N] [--detail D] [--payment-type T] [--payment-method M]

# Get account by ID - returns account JSON
billing-control account get --id <uuid>

# List accounts - returns JSON array
billing-control account list [--limit 100] [--token T] [--customer-id <uuid>]

# Delete account - returns deleted account JSON
billing-control account delete --id <uuid>

# Add balance - returns updated account JSON
billing-control account add-balance --id <uuid> --amount <float>

# Subtract balance - returns updated account JSON
billing-control account subtract-balance --id <uuid> --amount <float>
```

**Billing Operations:**
```bash
# Get billing record by ID - returns billing JSON
billing-control billing get --id <uuid>

# List billing records - returns JSON array
billing-control billing list [--limit 100] [--token T] [--customer-id <uuid>] [--account-id <uuid>]
```

### Output

- **stdout**: JSON formatted results only
- **stderr**: Log messages (DEBUG, ERROR, etc.)

Example:
```bash
# Redirect JSON to file, logs visible in terminal
./billing-control account list --customer-id <uuid> > accounts.json
```

### Configuration

Uses the same environment variables as billing-manager:
- `DATABASE_DSN` - Database connection string
- `RABBITMQ_ADDRESS` - RabbitMQ server address
- `REDIS_ADDRESS` - Redis server address
- `REDIS_PASSWORD` - Redis password (optional)
- `REDIS_DATABASE` - Redis database index (default: 1)

# RUN
```
Usage of ./billing-manager:
  -dbDSN string
        database dsn for webhook-manager. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_exchange_subscribes string
        rabbitmq exchange name for subscribe (default "bin-manager.call-manager.event")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.webhook-manager.request")
  -rabbit_queue_notify string
        rabbitmq queue name for event notify (default "bin-manager.webhook-manager.event")
  -rabbit_queue_susbscribe string
        rabbitmq queue name for message subscribe (default "bin-manager.webhook-manager.subscribe")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
```

# EXAMPLE
```
./billing-manager \
        -dbDSN 'bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager' \
        -prom_endpoint /metrics \
        -prom_listen_addr :2112 \
        -rabbit_addr amqp://guest:guest@rabbitmq.voipbin.net:5672 \
        -rabbit_exchange_delay bin-manager.delay \
        -rabbit_exchange_subscribes "bin-manager.call-manager.event" \
        -rabbit_queue_listen bin-manager.webhook-manager.request \
        -rabbit_queue_notify bin-manager.webhook-manager.event \
        -rabbit_queue_susbscribe bin-manager.webhook-manager.subscribe \
        -redis_addr 10.164.15.220:6379 \
        -redis_db 1
```
