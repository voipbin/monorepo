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
        The asterisk-proxy uses this asterisk ari application name. (default "asterisk-proxy")
  -ari_subscribe_all string
        The asterisk-proxy uses this asterisk subscribe all option. (default "true")
  -rabbit_addr string
        The asterisk-proxy connect to rabbitmq address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_queue_arievent string
        The asterisk-proxy sends the ARI event to this rabbitmq queue name. (default "asterisk_ari")
  -rabbit_queue_arirequest string
        The asterisk-proxy gets the ARI request from this rabbitmq queue name. (default "asterisk_ari_request_ip")
```

example
```
$ ./asterisk-proxy \
  -ari_account asterisk:asterisk \
  -ari_addr localhost:8088 \
  -ari_application voipbin \
  -ari_subscribe_all true \
  -rabbit_addr amqp://guest:guest@10.164.15.243:5672 \
  -rabbit_queue_arievent asterisk_ari_event \
  -rabbit_queue_arirequest asterisk_ari_10.164.0.3
```

# RabbitMQ RPC

RPC request
```
{
  "url": "/channels\?api_key=asterisk:asterisk\&endpoint=pjsip/test@sippuas\&app=test",
  "method": "POST",
  "date": "data",
  "data_type": "text/plain"
}
```

RPC response
```
{
  "status_code": 200,
  "data": "{...}"
}
```
