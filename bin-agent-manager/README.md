# agent-manager
The agnet-manager manages the agents.

# Usage
```
$ ./agent-manager -h
Usage of ./agent-manager:
  -dbDSN string
        database dsn for agent-manager. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_exchange_notify string
        rabbitmq exchange name for event notify (default "bin-manager.agent-manager.event")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.agent-manager.request")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
```

# RabbitMQ RPC

## Queues
Request queue
```
bin-manager.agent-manager.request
```

Event queue
```
bin-manager.agent-manager.event
```

## Events

### agent_created(to-be)
Event for agent's create.

### agent_updated(to-be)
Event for agent's update.

### agent_deleted(to-be)
Event for agent's delete.

### tag_created(to-be)
Event for tag's create.

### tag_updated(to-be)
Event for tag's update.

### tag_deleted(to-be)
Event for tag's delete.

# Agent
agent is a person who handles incoming or outgoing calls. Normally, they will answer the calls where in the queue.

## Status
Agent has status.

```
	StatusAvailable Status = "available" // available
	StatusAway      Status = "away"      // away
	StatusBusy      Status = "busy"      // busy
	StatusOffline   Status = "offline"   // offline
	StatusRinging   Status = "ringing"   // voipbin is making a call to the agent
```

# Tag
Represents agent's skills and groups.

