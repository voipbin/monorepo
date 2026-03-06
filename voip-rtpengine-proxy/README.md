# voip-rtpengine-proxy

RTPEngine NG protocol proxy. Bridges call-manager RabbitMQ commands to RTPEngine's
native NG protocol (bencode/UDP).

## Queues

- Permanent: `rtpengine.proxy.request`
- Volatile: `rtpengine.<proxyID>.request` (proxyID = VM internal IP, e.g. `10.164.0.12`)

## Command format

POST `/v1/commands` with any RTPEngine NG command as JSON:

```json
{"command":"query","call-id":"abc123","from-tag":"xyz"}
```

## Environment variables

See `cmd/rtpengine-proxy/init.go` for the full list with defaults.
