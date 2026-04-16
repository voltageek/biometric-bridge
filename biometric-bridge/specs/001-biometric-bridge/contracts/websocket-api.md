# WebSocket API Contract — Biometric Bridge (/events)

The Biometric Bridge provides an authenticated WebSocket event stream at `GET /events`. Clients MUST supply a short-lived ES256 JWT in the `token` query parameter (e.g., `/events?token=<jwt>`). The server will validate the token and close the connection with close code `4001` when the token expires.

General event format

Events are JSON objects. Common fields:

- `type` (string) — event type
- `deviceId` (string) — device name
- `userId` (string, optional) — user ID when relevant
- `message` (string, optional) — textual or JSON-encoded message

Example event envelope:

```json
{
  "type": "scan",
  "deviceId": "reception",
  "userId": "demo-user",
  "eventCode": 0
}
```

New event: `enrollment_retry`

When the server's slap-enroll handler performs retries for an impression because one or more fingers failed the quality threshold, it emits an `enrollment_retry` event to let subscribed clients observe progress.

Event schema:

```json
{
  "type": "enrollment_retry",
  "deviceId": "reception",
  "userId": "user-001",
  "attempt": 2,
  "message": "{\"impressionNum\":1,\"failedFingers\":[{\"finger\":\"right_index\",\"quality\":40,\"threshold\":60}]}"
}
```

Notes and recommendations:

- `attempt` is the one-based retry attempt number (e.g., `2` means this event precedes the second attempt).
- Currently, the `message` field contains a JSON-encoded details object. Clients SHOULD parse `message` as JSON when `type == "enrollment_retry"` to obtain `impressionNum` and `failedFingers`.
- Recommendation: If you control both bridge and client, treat `message` as structured JSON. Future versions may expose a typed `details` field in the envelope for easier consumption.

Other event types (existing)

- `scan` — emitted after a single-finger scan
- `connected` — device connected
- `error` — device error; `message` contains the human-readable error
- `reconnecting` — device is reconnecting; contains retry/attempt info

Auth/connection lifecycle

- Clients must pass `token` query parameter containing a valid ES256 JWT.
- The server will close the connection with close code `4001` when the token expires.

Example (wscat)

```bash
wscat -c "wss://bridge.example.local/events?token=$TOKEN"
```
