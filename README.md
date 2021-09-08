# webhook-manager

Webhook-manager for webhook event notification.

# RUN
```
Usage of ./webhook-manager:
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
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
```

# EXAMPLE
```
./webhook-manager \
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
