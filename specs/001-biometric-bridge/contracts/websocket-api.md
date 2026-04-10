# WebSocket API Contract: Biometric Bridge

**Phase**: 1 — Design  
**Date**: 2026-04-10  
**Endpoint**: `ws://127.0.0.1:7070/events?token=<JWT>`

## Connection

### Authentication

The JWT is passed via the `token` query parameter (browsers cannot set custom headers on WebSocket upgrade requests). The same validation rules apply as for HTTP endpoints:
- `ES256` algorithm
- `aud` = `"biometric-bridge"`
- Valid `iss` and `exp`

If the token is invalid, the server rejects the upgrade with HTTP `401`.

### Token Expiry Enforcement

The bridge reads the `exp` claim at connection time and schedules a timer. When the token expires, the bridge sends a WebSocket close frame (code `4001`, reason `"token expired"`) and closes the connection. The client must reconnect with a fresh token.

### Connection Limits

Maximum 10 concurrent WebSocket connections (FR-004). The 11th connection attempt receives HTTP `503` with body `{"error": "max subscribers reached"}`.

### Connection Lifecycle

```
Client                                          Bridge
  │                                               │
  ├── GET /events?token=<JWT> ──────────────────► │
  │   Upgrade: websocket                          │
  │                                               ├── Validate JWT
  │                                               ├── Check subscriber count < 10
  │ ◄──────────────────── 101 Switching Protocols ─┤
  │                                               ├── Register subscriber
  │                                               │
  │ ◄──── {"type":"scan","deviceId":"reception",  │
  │        "userId":"user-001","eventCode":4354}   │
  │                                               │
  │ ◄──── {"type":"reconnecting",                 │
  │        "deviceId":"reception",                │
  │        "attempt":1,"waitSeconds":1}           │
  │                                               │
  │ ◄──── {"type":"connected",                    │
  │        "deviceId":"reception"}                │
  │                                               │
  │ ◄──── close frame (4001, "token expired") ────┤  [when JWT exp reached]
  │                                               ├── Unregister subscriber
```

## Event Message Types

All messages are JSON objects sent from server to client. The client does not send data messages (the connection is server-push only). The client may send WebSocket ping/pong frames for keep-alive.

### Scan Event

Emitted when a fingerprint is detected on any connected device.

```json
{
  "type": "scan",
  "deviceId": "reception",
  "userId": "user-001",
  "eventCode": 4354
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"scan"` |
| `deviceId` | string | Human-readable device name from config |
| `userId` | string | User ID associated with the scan event (from device) |
| `eventCode` | integer | SDK-specific event code (for debugging/logging) |

### Reconnecting Event

Emitted when the driver begins a reconnection attempt for a device.

```json
{
  "type": "reconnecting",
  "deviceId": "reception",
  "attempt": 3,
  "waitSeconds": 4
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"reconnecting"` |
| `deviceId` | string | Human-readable device name from config |
| `attempt` | integer | Reconnection attempt number (1-based) |
| `waitSeconds` | integer | Seconds the driver will wait before this attempt |

### Connected Event

Emitted when a device successfully reconnects after a disconnection.

```json
{
  "type": "connected",
  "deviceId": "reception"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"connected"` |
| `deviceId` | string | Human-readable device name from config |

### Error Event

Emitted when a device encounters an error (e.g., stream closed, SDK error).

```json
{
  "type": "error",
  "deviceId": "reception",
  "message": "stream closed"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"error"` |
| `deviceId` | string | Human-readable device name from config |
| `message` | string | Human-readable error description |

## WebSocket Close Codes

| Code | Reason | Condition |
|------|--------|-----------|
| 1000 | `"normal closure"` | Bridge is shutting down gracefully |
| 4001 | `"token expired"` | JWT `exp` reached during active connection |

## Client Behavior Expectations

1. The client should reconnect with exponential backoff when the connection closes
2. On close code `4001`, the client must obtain a fresh JWT before reconnecting
3. The client should handle all four event types gracefully
4. The client should not send data messages (server-push only)
5. Events may arrive from any connected device in any order
6. During a burst of events, all events are delivered in arrival order without dropping
