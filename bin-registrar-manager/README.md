# registrar-manager

Registrat manager for the voipbin project

# Usage
```
$ ./registrar-manager -h
Usage of ./registrar-manager:
  -dbDSNAst string
        database dsn for asterisk. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -dbDSNBin string
        database dsn for bin-manager. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.registrar-manager.request")
  -rabbit_queue_notify string
        rabbitmq queue name for event notify (default "bin-manager.registrar-manager.event")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
```

## Example
```
$ ./registrar-manager \
    -prom_endpoint /metrics \
    -prom_listen_addr :2112 \
    -dbDSNAstasterisk:b62160b0-ea4a-11ea-9d60-8b6c204cab46@tcp(10.126.80.5:3306)/asterisk \
    -dbDSNBin bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager \
    -rabbit_addr amqp://guest:guest@rabbitmq.voipbin.net:5672 \
    -rabbit_queue_listen bin-manager.registrar-manager.request \
    -rabbit_queue_notify bin-manager.registrar-manager.event \
    -rabbit_exchange_delay bin-manager.delay \
    -redis_addr 10.164.15.220:6379 \
    -redis_db 1
```

# RabbitMQ queues
## Request Listen Queue
bin-manager.registrar-manager.request

## Event Notify Queue
bin-manager.registrar-manager.event


<!-- Updated dependencies: 2026-02-20 -->
