# WebSocket Keep-Alive with Server-Side Ping/Pong

**Date:** 2026-01-20
**Status:** Approved
**Author:** Design Session

## Problem Statement

WebSocket connections in bin-api-manager drop during idle periods. When no messages flow for a certain period, the load balancer assumes the connection died and closes it.

## Root Cause

The WebSocket implementation lacks a keep-alive mechanism:
- Missing ping/pong frames
- Missing read/write deadlines
- Passive goroutines transmit only during business traffic

When no client commands arrive and no ZMQ events need forwarding, the load balancer sees a dead connection and closes it.

## Solution Overview

Implement server-side ping/pong using WebSocket protocol (RFC 6455). The server sends periodic ping frames. Clients automatically respond with pong frames. This bidirectional traffic prevents load balancer timeouts and detects dead connections.

## Architecture

### Modified Connection Flow

```
subscriptionRun() creates WebSocket connection
├─ subscriptionRunWebsock()    // Existing: handles client messages
├─ subscriptionRunZMQSub()      // Existing: forwards ZMQ events
└─ subscriptionRunPinger()      // NEW: sends periodic pings
```

### Configuration

- **Ping Interval**: 30 seconds (works for most load balancers with 60s+ timeouts)
- **Pong Wait**: 60 seconds (time to wait for pong response)
- **Write Wait**: 10 seconds (timeout for writing frames)

### How It Works

1. Every 30 seconds, the pinger goroutine sends a ping frame
2. The gorilla/websocket library handles pong responses automatically
3. A pong handler updates the read deadline on each pong
4. If no pong arrives within 60 seconds, the read deadline expires
5. The pinger sets write deadlines to detect write failures
6. Any error (write failure, deadline expiration) cancels the context and closes all goroutines

## Implementation Details

### 1. Add Configuration Constants

In `pkg/websockhandler/subscription.go`:

```go
const (
    // Time allowed to write a message to the peer
    writeWait = 10 * time.Second

    // Time allowed to read the next pong message from the peer
    pongWait = 60 * time.Second

    // Send pings to peer with this period (must be less than pongWait)
    pingPeriod = 30 * time.Second
)
```

### 2. Modify subscriptionRun()

- Add pong handler to WebSocket connection
- Set initial read deadline
- Launch subscriptionRunPinger() goroutine
- Add mutex for write protection

### 3. Create subscriptionRunPinger()

New function that:
- Uses time.Ticker to send pings every 30 seconds
- Sets write deadline before each ping
- Exits on write errors or context cancellation
- Cleans up ticker on exit

### 4. Protect Concurrent Writes

Add `sync.Mutex` to protect all `ws.WriteMessage()` calls:
- ZMQ subscriber writes data frames
- Pinger writes ping frames
- Both need mutex protection to avoid race conditions

## Error Handling

### Scenarios Covered

**Client disconnects abruptly:**
- Pinger fails to write ping → Returns error → Cancels context → Clean shutdown
- Read deadline expires → Other goroutines error → Clean shutdown

**Network partition:**
- Ping write succeeds but pong never arrives → Read deadline expires → Closes connection
- Prevents zombie connections

**Slow client:**
- 60s pong wait allows for network latency
- Increase if needed

**Server shutdown:**
- Context cancellation propagates to all goroutines
- Defer stops ticker
- Closes WebSocket gracefully

**Race conditions:**
- Mutex protects concurrent writes
- One write operation at a time

## Scope

This fix applies to both WebSocket endpoints:
- `/ws` - General agent WebSocket
- `/service_agents/ws` - Service agent WebSocket

Both use `RunSubscription` and receive the keep-alive mechanism.

## Testing Considerations

**Manual Testing:**
1. Connect WebSocket client
2. Stop sending messages (idle for 60+ seconds)
3. Verify connection stays alive
4. Monitor server logs for ping/pong activity

**What to Monitor:**
- Connection duration during idle periods
- Ping frame transmission (every 30s)
- Pong response reception
- No disconnections during normal idle periods

**Edge Cases to Test:**
- Client that ignores pings (should disconnect after 60s)
- Network interruption (should detect and close connection)
- High-traffic scenario (pings should not interfere with data)

## Rollout Plan

1. Implement changes in bin-api-manager
2. Run verification workflow (go mod tidy, vendor, test, lint)
3. Deploy to staging environment
4. Monitor WebSocket connection stability
5. Deploy to production

## Future Enhancements

- Make ping interval and pong wait configurable via CLI flags
- Add Prometheus metrics for WebSocket connection health
- Implement graceful shutdown with proper close handshake
- Consider client-side ping validation for bidirectional monitoring
