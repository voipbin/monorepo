FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o timeline-manager ./cmd/timeline-manager
RUN go build -o timeline-control ./cmd/timeline-control

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/timeline-manager .
COPY --from=builder /app/timeline-control .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./timeline-manager"]
