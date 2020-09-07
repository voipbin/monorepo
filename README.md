# api-manager

API manager for Restful APIs to access from the public internet.

# Usage
```
Usage of ./api-manager:
  -dsn string
        database dsn (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -jwt_key string
        key string for jwt hashing (default "voipbin")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_queue_call string
        rabbitmq queue name for request listen (default "bin-manager.call-manager.request")
  -rabbit_queue_flow string
        rabbitmq queue name for flow request (default "bin-manager.flow-manager.request")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
  -ssl_cert string
        Cert key file for ssl connection. (default "./etc/ssl/cert.pem")
  -ssl_private string
        Private key file for ssl connection. (default "./etc/ssl/prikey.pem")```

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

$ curl -k -X POST https://api.voipbin.net/auth/login -d '{"username":"test","password":"test"}' -v

curl -k -X POST https://api.voipbin.net/v1.0/conferences\?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDAwNDQ5MjQsInVzZXIiOnsiaWQiOjEsInVzZXJuYW1lIjoidGVzdCJ9fQ.UJR04FE7b00PRnjEt9kNy4f6DYyrZvZ_jpAVomqzNso -d '{"type":"conference", "name":"test conference"}' -v
```
