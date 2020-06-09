FROM golang:latest

ENV GOPROXY https://goproxy.cn,direct
WORKDIR $GOPATH/src/gitlab.com/voipbin/bin-manager/api-manager
COPY . $GOPATH/src/gitlab.com/voipbin/bin-manager/api-manager
RUN go build .

EXPOSE 8000
ENTRYPOINT ["./go-gin-example"]
