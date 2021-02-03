# asterisk-proxy

Asterisk proxy

# Run

```
$ ./asterisk-proxy -h
Usage of ./asterisk-proxy:
  -ami_event_filter string
        The list of messages for listen.
  -ami_host string
        The host address for AMI connection. (default "127.0.0.1")
  -ami_password string
        The password for AMI login. (default "asterisk")
  -ami_port string
        The port number for AMI connection. (default "5038")
  -ami_username string
        The username for AMI login. (default "asterisk")
  -ari_account string
        The asterisk-proxy uses this asterisk ari account info. id:password (default "asterisk:asterisk")
  -ari_addr string
        The asterisk-proxy connects to this asterisk ari service address (default "localhost:8088")
  -ari_application string
        The asterisk-proxy uses this asterisk ari application name. (default "voipbin")
  -ari_subscribe_all string
        The asterisk-proxy uses this asterisk subscribe all option. (default "true")
  -interface_name string
        The main interface device name. (default "eth0")
  -rabbit_addr string
        The asterisk-proxy connect to rabbitmq address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_queue_listen string
        Additional comma separated asterisk-proxy's listen request queue name. (default "asterisk.call.request")
  -rabbit_queue_publish string
        The asterisk-proxy sends the ARI event to this rabbitmq queue name. The queue must be created before. (default "asterisk.all.event")
  -redis_addr string
        The redis address for data caching (default "localhost:6379")
  -redis_db int
        The redis database for caching
```

example
```bash
$ ./asterisk-proxy \
  -ari_account asterisk:asterisk \
  -ari_addr localhost:8088 \
  -ari_application voipbin \
  -ari_subscribe_all true \
  -ami_host 127.0.0.1 \
  -ami_port 5038 \
  -ami_username asterisk \
  -ami_password asterisk \
  -interface_name docker0 \
  -rabbit_addr amqp://guest:guest@10.164.15.243:5672 \
  -rabbit_queue_publish asterisk.all.event \
  -rabbit_queue_listen asterisk.call.request \
  -redis_addr 10.164.15.220:6379 \
  -redis_db 1
```

# RabbitMQ RPC

Event message
```
	Type     string `json:"type"`
	DataType string `json:"data_type"`
	Data     string `json:"data"`
```

ARI event
```
{
  "type": "ari_event",
  "data_type": "application/json",
  "data: "{...}"
}
```

AMI event
```
{
  "type": "ami_event",
  "data_type": "application/json",
  "data: "{...}"
}
```


RPC requests

ARI request
```
{
  "uri": "/ari/channels\?api_key=asterisk:asterisk\&endpoint=pjsip/test@sippuas\&app=test",
  "method": "POST",
  "data": "data",
  "data_type": "text/plain"
}
```

AMI request
```
{
  "uri": "/ami",
  "method": "",
  "data": "{\"Action\": \"Ping\"}",
  "data_type": "text/plain"
}
```

RPC response
```
{
  "status_code": 200,
  "data_type": "application/json",
  "data": "{...}"
}
```

# Test

```
$ ssh -L 8088:127.0.0.1:8088 10.164.0.3
$ ./asterisk-proxy \
  -ari_account asterisk:asterisk \
  -ari_addr localhost:8088 \
  -ari_application voipbin \
  -ari_subscribe_all true \
  -ami_host 127.0.0.1 \
  -ami_port 5038 \
  -ami_username asterisk \
  -ami_password asterisk \
  -rabbit_addr amqp://guest:guest@10.164.15.243:5672 \
  -rabbit_queue_publish asterisk.all.event \
  -rabbit_queue_listen asterisk.42:01:0a:a4:0f:d0.request,asterisk.call.request \
  -asterisk_id 42:01:0a:a4:0f:d0 \
  -asterisk_address_internal 10.164.15.208 \
  -redis_addr 10.164.15.220:6379 \
  -redis_db 1
```
