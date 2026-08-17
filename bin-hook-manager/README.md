# hook-manager
The hook-manager handles webhook message from the outside of the voipbin.

# Usage
```
Usage of ./hook-manager:
  -dsn string
        database dsn (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -ssl_cert_base64 string
        Base64 encoded cert key for ssl connection.
  -ssl_private_base64 string
        Base64 encoded private key for ssl connection.
```

# SSL
The app needs the base64 encrypted ssl certificate to enable the SSL connection.
To generate base64 encoded certificates, you need to run this command.
```
$ cat <your cert file> | base64 -w 0
```

# List of endpoints

* https://hook.voipbin.net/v1.0/emails : email-manager
* https://hook.voipbin.net/v1.0/messages : message-manager
* https://hook.voipbin.net/v1.0/conversation : conversation-manager

<!-- Updated dependencies: 2026-03-30 -->

# Deploy

Komodo-managed (VOIP-1353), same mechanism as the other `bin-*-manager` services
(see bin-call-manager for the original pattern). `bin-hook-manager-build` pushes
the image, then `bin-hook-manager-deploy` renders the tag into
`komodo/docker-compose.yml` and deploys via `.circleci/scripts/komodo-api-deploy.sh`,
gated behind a single `build-approval`.

Unlike every other `bin-*-manager` service, this one has no RabbitMQ
competing-consumer safety net — `hook.voipbin.net`'s public traffic is routed by
`monorepo-etc/infra-caddy`'s Caddyfile, which reverse-proxies to this container by
its fixed name (`voipbin-hook-manager`). See
[docs/operations.md](docs/operations.md#deployment-komodo) for the full
cutover-sequencing note and what happens if it's done out of order.
