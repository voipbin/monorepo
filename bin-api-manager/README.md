# api-manager
API manager for Restful APIs to access from the public internet.

# Usage
```
Usage of ./api-manager:
  -dsn string
        database dsn (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -gcp_bucket_name string
        the gcp bucket name for tmp storage (default "bucket")
  -gcp_credential string
        the GCP credential file path (default "./credential.json")
  -gcp_project_id string
        the gcp project id (default "project")
  -jwt_key string
        key string for jwt hashing (default "voipbin")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
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

# Example
```
./api-manager \
      -dsn "bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager" \
      -ssl_private "./etc/ssl/privkey.pem" \
      -ssl_cert "./etc/ssl/cert.pem" \
      -jwt_key "voipbin" \
      -rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" \
      -rabbit_exchange_delay bin-manager.delay \
      -rabbit_queue_request_call bin-manager.call-manager.request \
      -rabbit_queue_request_flow bin-manager.flow-manager.request \
      -rabbit_queue_request_registrar bin-manager.registrar-manager.request \
      -rabbit_queue_request_number bin-manager.number-manager.request \
      -rabbit_queue_request_storage bin-manager.storage-manager.request \
      -rabbit_queue_request_transcode bin-manager.transcode-manager.request \
      -redis_addr 10.164.15.220:6379 \
      -redis_db 1
```

# Test

## API test
```
$ curl -k https://api.voipbin.net/ping

$ curl -k -X POST https://api.voipbin.net/auth/login -d '{"username":"test","password":"test"}' -v

$ curl -k -X POST https://api.voipbin.net/v1.0/conferences\?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDAwNDQ5MjQsInVzZXIiOnsiaWQiOjEsInVzZXJuYW1lIjoidGVzdCJ9fQ.UJR04FE7b00PRnjEt9kNy4f6DYyrZvZ_jpAVomqzNso -d '{"type":"conference", "name":"test conference"}' -v

$ curl -k -X POST https://localhost:443/v1.0/calls\?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDAyMDIyMDcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.YnECmUr2chV-cpBbwedJ905ztcVUq0xVv5Tec_nibaU -v -d '{"source": {"type": "sip","target": "source@test.voipbin.net"},"destination": {"type": "sip","target": "destination@test.voipbin.net"},"actions": []}'
```

## SWAGGER test
Access to
```
https://api.voipbin.net/swagger/index.html
```

# Build
Update git config
```
$ git config --global url.git@gitlab.com:.insteadOf https://gitlab.com/
or
$ git config --global url."https://<$GL_DEPLOY_USER>:<$GL_DEPLOY_TOKEN@gitlab.com>".insteadOf "https://gitlab.com"
```

Set golang
```
$ export GOPRIVATE="gitlab.com/voipbin"
```

```
$ go mod vendor
$ go build ./cmd/...
```

## swag
Install the swaggo
```
$ go install github.com/swaggo/swag/cmd/swag@latest

$ go get -u -v github.com/go-swagger/go-swagger/cmd/swagger
$ go get -u github.com/swaggo/swag/cmd/swag
$ go get -u github.com/swaggo/gin-swagger
$ go get -u github.com/swaggo/files
```

swag
```
$ swag fmt
$ swag init --parseDependency --parseInternal -g cmd/api-manager/main.go -o docsapi
```

# API documents
```
https://api.voipbin.net/docs/
```

## Sphinx install
```
$ sudo apt install python3-sphinx
$ pip3 install sphinx-rtd-theme
```

## Install sphinx-wagtail-theme
```
$ python3 -m venv ~/.venv/sphinx-wagtail-theme
$ ~/.venv/sphinx-wagtail-theme/bin/pip install sphinx sphinx_rtd_theme sphinx-wagtail-theme
$ cd docsdev
$ ~/.venv/sphinx-wagtail-theme/bin/sphinx-build ./source ./build
```

## Sphinx with docker
```
$ cd docsdev
$ docker run --rm -v /Users/sungtaekim/gitlab/voipbin/bin-manager/api-manager/docsdev:/documents suttang/sphinx-rtd-theme make html
```

## Create html
```
$ cd docsdev
$ make html
$ ~/.venv/sphinx-wagtail-theme/bin/sphinx-build ./source ./build

```
