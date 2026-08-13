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

`bin-hook-manager-build` pushes the image, and a single `build-approval` gate covers
the whole pipeline (test -> build -> deploy) through to production.
`bin-hook-manager-deploy` runs after `bin-hook-manager-build`, bumping this service's pin on
bm-nyc-01 and recreating the container. See
`.circleci/scripts/ssh-deploy.sh` (this pattern was piloted with
bin-call-manager). The previous GKE deploy path for this service has been
removed.
