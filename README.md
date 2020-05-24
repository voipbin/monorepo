# asterisk-proxy

Asterisk proxy

# Run

```
./asterisk-proxy -h
Usage of ./asterisk-proxy:
  -ari_account string
        The asterisk-proxy uses this asterisk ari account info. id:password (default "asterisk:asterisk")
  -ari_addr string
        The asterisk-proxy connects to this asterisk ari service address (default "localhost:8088")
  -ari_application string
        The asterisk-proxy uses this asterisk ari application name. (default "voipbin")
  -ari_subscribe_all string
        The asterisk-proxy uses this asterisk subscribe all option. (default "true")
  -asterisk_address_internal string
        The asterisk internal ip address (default "127.0.0.1:5060")
  -asterisk_id string
        The asterisk id (default "00:11:22:33:44:55")
  -rabbit_addr string
        The asterisk-proxy connect to rabbitmq address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_queue_listen string
        Comma separated asterisk-proxy's listen request queue name. (default "asterisk.<asterisk_id>.request,asterisk.call.request")
  -rabbit_queue_publish string
        The asterisk-proxy sends the ARI event to this rabbitmq queue name. The queue must be created before. (default "asterisk.all.event")
  -redis_addr string
        The redis address for data caching (default "localhost:6379")
  -redis_db int

```

example
```
$ ./asterisk-proxy \
  -ari_account asterisk:asterisk \
  -ari_addr localhost:8088 \
  -ari_application voipbin \
  -ari_subscribe_all true \
  -asterisk_address_internal 10.164.0.3:5060 \
  -asterisk_id 42:01:0a:a4:00:03 \
  -rabbit_addr amqp://guest:guest@10.164.15.243:5672 \
  -rabbit_queue_listen asterisk.42:01:0a:a4:00:03.request,asterisk.call.request \
  -rabbit_queue_publish asterisk.all.event \
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

```
{
  "type": "ari_event",
  "data_type": "application/json",
  "data: "{...}"
}
```


RPC request
```
{
  "uri": "/channels\?api_key=asterisk:asterisk\&endpoint=pjsip/test@sippuas\&app=test",
  "method": "POST",
  "date": "data",
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
  -rabbit_addr amqp://guest:guest@10.164.15.243:5672 \
  -rabbit_queue_arievent asterisk_ari_event \
  -rabbit_queue_arirequest asterisk_ari_request-42:01:0a:a4:00:03
```
