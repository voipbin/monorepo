# Burstable QoS for All Services

**Goal:** Switch all 30 services from Guaranteed to Burstable QoS class by adding explicit `requests` at 10% of `limits` for both CPU and memory.

**Problem:** All deployments only specify `limits`, which makes Kubernetes set `requests = limits` (Guaranteed QoS). This reserves the full limit on each node even though actual usage is 12-17% CPU. Result: nodes appear full at 80-92% allocated CPU while being mostly idle.

**Solution:** Add explicit `requests` at 10% of `limits`. This tells the scheduler "reserve 10% but allow bursting up to the limit." Changes QoS class from Guaranteed to Burstable.

**Before:**
```yaml
resources:
  limits:
    cpu: "200m"
    memory: "100Mi"
```

**After:**
```yaml
resources:
  requests:
    cpu: "20m"
    memory: "10Mi"
  limits:
    cpu: "200m"
    memory: "100Mi"
```

**Minimum floor:** 1m CPU, 2M memory (for services with very small limits).

**Scope:** All 30 `bin-*/k8s/deployment.yml` files, including multi-container pods (pipecat-manager, tts-manager).

**Risk:** Under extreme memory pressure, Kubernetes may evict Burstable pods before Guaranteed ones. Acceptable given current low utilization.
