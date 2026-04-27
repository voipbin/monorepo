# Per-pod RabbitMQ Queues

Some services need to route RPCs to a *specific* pod that owns in-memory state (e.g., a streaming session, an audio bridge). VoIPbin handles this with per-pod queues alongside the standard shared per-service queue.

## Naming convention

```
<service>.request                       — shared queue, any consumer wins
<service>.request.<host_id>             — per-pod queue, owned by exactly one pod
```

Example from `bin-pipecat-manager`:
- `bin-manager.pipecat-manager.request` — call creation, lookup, etc.
- `bin-manager.pipecat-manager.request.<POD_IP>` — message-send, terminate, ping

## Declaration

Per-pod queues MUST be declared as **volatile** (auto-delete when the last consumer disconnects). The volatility is what guarantees the queue disappears when its owning pod dies, so a publisher can detect death by RPC timeout.

```go
sockHandler.QueueCreate(queue, "volatile")
```

Shared queues use `"normal"` (durable) declaration.

## Identity source

`HostID` is set at pod startup from the K8s Downward API — typically `POD_IP`:

```yaml
env:
- name: POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
```

Then in `cmd/<service>/main.go`:

```go
listenIP := os.Getenv("POD_IP")
if listenIP == "" {
    return fmt.Errorf("could not get the listen ip address")
}
listenQueue := fmt.Sprintf("%s.%s", commonoutline.QueueName<Service>Request, listenIP)
```

Persist this `HostID` on the resource (e.g., `pipecatcall.HostID`) so the consumer service can route follow-up RPCs to the right pod.

## Limitations

- **Calico POD_IP recycle.** Calico CNI reassigns released pod IPs within minutes. A stored `HostID` may resolve to a different pod after a restart. Pair this pattern with [per-pod-liveness-preflight.md](per-pod-liveness-preflight.md) to detect dead pods quickly, but be aware that recycle gives a matching IP and bypasses the simple echo check. v2 candidate: store `POD_UID` (immutable across IP recycle) instead of or alongside `POD_IP`.
- **No load balancing.** A per-pod queue routes to one pod by definition; if that pod is busy or backlogged, the request waits. Use the shared queue for stateless calls and reserve per-pod for genuine session affinity.
- **No persistence.** Volatile queues lose messages if the pod dies before consuming them. Acceptable for the use case (the session is also gone), but don't use this pattern for important non-session work.

## When to use

Use per-pod queues when **all** of these are true:
1. Some operation must reach a specific pod that holds in-memory state.
2. That state cannot be reconstructed cheaply on another pod.
3. You have a way to handle pod death (typically the liveness preflight pattern + caller error surfacing).

If the state is reconstructable from DB or shared cache, use the shared queue and let any pod handle the request.

## Reference implementations

- `bin-pipecat-manager/cmd/pipecat-manager/main.go:115-118` (POD_IP read into `listenIP`) and `:167-168` (per-pod queue construction `<QueueNamePipecatRequest>.<host_id>`)
- `bin-pipecat-manager/pkg/listenhandler/main.go` (volatile per-pod queue declaration)
- `bin-common-handler/pkg/requesthandler/pipecat_message.go` (per-pod RPC routing)
