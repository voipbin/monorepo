# flow-manager

The flow-manager manages the flows.

# Usage
```
$ ./flow-manager -h
Usage of ./flow-manager:
  -dbDSN string
        database dsn for flow-manager. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -log_level int
        log level (default 5)
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_queue_event string
        rabbitmq queue name for event notify (default "bin-manager.flow-manager.event")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.flow-manager.request")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
```

## Example
```
$ ./flow-manager \
-prom_endpoint /metrics \
-prom_listen_addr :2112 \
-dbDSN 'bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager' \
-rabbit_addr amqp://guest:guest@rabbitmq.voipbin.net:5672 \
-rabbit_queue_listen bin-manager.flow-manager.request \
-rabbit_queue_event bin-manager.flow-manager.event \
-rabbit_exchange_delay bin-manager.delay \
-redis_addr 10.164.15.220:6379 \
-redis_db 1
```

# RabbitMQ RPC

## Qeueue
Queue name: flow_manager-request

## Request
RPC request
```
{
  "uri": "<string>",
  "method": "<string>",
  "data_type": "<string>"
  "data": {...},
}
```
* uri: The target uri destination.
* method: Capitalized http methods. POST, GET, PUT, DELETE, ...
* data_type: Type of data. Mostly, "application/json".
* data: data

## Response
RPC response
```
{
  "status_code": <number>,
  "data_type": "<string>"
  "data": "{...}"
}
```
* status_code: Status code.
* data_type: Type of data.
* data: data.

# URI string

# Restful APIs


## /flows/<flow-id>
Returns registered flow info.

## /flows/<flow-id>/actions
Returns registered actions.

## /flows/<flow-id>/actions/<action-id>
Returns registered action.

### example
```
request
{
  "uri": "/v1/flows/3271831e-880f-11ea-bc66-4f3de31bc41e/actions",
  "method": "GET",
  "data_type": "application/json"
  "data": {},
}

response
{
  "status_code": 200,
  "data": {
    "id": "7e4ae910-880f-11ea-b08b-dbc017c70055",
    "action": "answer",
    "next_action": "93835308-880f-11ea-97f7-f71252fc3528",
    "flow_id": "3271831e-880f-11ea-bc66-4f3de31bc41e",
    "data": {
      "action_version": "0.1"
    }
  }
}
```
