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
