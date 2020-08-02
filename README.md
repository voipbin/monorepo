# api-manager

API manager for Restful APIs to access from the public internet.

# Usage
```
Usage of ./api-manager:
  -dsn string
        database dsn (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -jwt_key string
        key string for jwt hashing (default "voipbin")
```

# SSL
* See detial at `./etc/ssl`.

# Example
```
./api-manager \
      -dsn "bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager" \
      -ssl_private "./etc/ssl/privkey.pem" \
      -ssl_cert "./etc/ssl/cert.pem" \
      -jwt_key "voipbin" \
      -rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" \
      -rabbit_exchange_delay bin-manager.delay \
      -rabbit_queue_call bin-manager.call-manager.request \
      -rabbit_queue_flow bin-manager.flow-manager.request
```

# Test
```
$ curl -k https://api.voipbin.net/ping
```
