# HTTP API Contract — Biometric Bridge

This document describes the canonical HTTP API surface for the Biometric Bridge. It focuses on the new slap enrollment endpoint added in recent work.

All authenticated API endpoints require a short-lived ES256 JWT in the `Authorization: Bearer <token>` header unless stated otherwise. `/healthz` is public.

## POST /api/slap-enroll

Authenticate: JWT required

Purpose: Perform multi-finger "slap" enrollment (two impressions) on a device that supports slap capture (RealScan G10). The endpoint captures two impressions, validates per-finger quality against `minQuality`, retries per-impression up to `maxRetries`, and returns base64-encoded templates and slap images to the caller. The bridge does not store templates.

Request JSON

- Content-Type: application/json
- Body schema:

```json
{
  "deviceId": "reception",      // string, required — device name from config
  "userId": "user-001",        // string, required — caller's user id
  "userName": "Jane Smith",    // string, required — display name
  "mode": "right_four",        // string enum: left_four | right_four | two_thumbs
  "minQuality": 60,              // integer 0-100, optional, default: configured default (60)
  "maxRetries": 3,               // integer 0-10, optional, default: configured default (3)
  "strict": true                 // boolean, optional, default: true
}
```

Behavior and defaults

- `minQuality` is interpreted on a 0–100 scale (higher is better). Drivers normalize their native quality scales to 0–100 before returning them to the API.
- `maxRetries` controls the number of retry attempts per impression. The handler will attempt `maxRetries + 1` total attempts (initial capture + retries).
- `strict` controls finger-count enforcement: when `strict=true`, the handler will return 422 if the captured impression detects fewer fingers than expected for the requested `mode`.
- Overall timeout: default 60s for the whole operation (two impressions + retries). Per-impression capture timeout is typically 20s.

Success response (200)

Content-Type: application/json

Body schema (abridged):

```json
{
  "ok": true,
  "userId": "user-001",
  "impressions": [
    {
      "impressionNumber": 1,
      "mode": "right_four",
      "slapImage": "<base64>",
      "slapWidth": 1600,
      "slapHeight": 1500,
      "fingers": [
        {
          "finger": "right_index",
          "template": "<base64>",
          "width": 300,
          "height": 400,
          "quality": 85
        }
      ]
    },
    { "impressionNumber": 2, ... }
  ],
  "totalRetries": 1
}
```

Notes:
- `template` values are base64-encoded raw template/image bytes returned by the driver. The bridge returns templates to the caller but does not persist them.
- `quality` is the normalized 0–100 score (higher = better). Drivers that expose different native scales (for example, RealScan NFIQ 1–5) are responsible for normalizing to 0–100.

Error responses

- 400 Bad Request — invalid JSON or validation error (missing required fields, invalid `mode`, `minQuality` out of range, etc.)
- 401 Unauthorized — missing or invalid JWT
- 404 Not Found — device name not registered in the registry
- 409 Conflict — device is busy (registry Acquire returned busy)
- 422 Unprocessable Entity — quality validation failed after retries or strict finger-count mismatch. Response body contains details in `failedFingers` when applicable.
- 501 Not Implemented — driver does not support slap capture (e.g., BS2)
- 504 Gateway Timeout — overall enrollment timeout
- 502 Bad Gateway — driver error

Timing expectations

- Typical baseline: ~40s (2 × 20s per-impression capture timeout) when no retries are needed.
- Worst-case with retries: depends on `maxRetries` (e.g., 60s default accommodates a small number of retries).

Examples

CURL (happy path):

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"deviceId":"reception","userId":"user-001","userName":"Jane","mode":"right_four","minQuality":60}' \
  https://bridge.example.local/api/slap-enroll
```

Security / privacy note

- The bridge returns raw biometric templates to the caller but does not store them. Integrators are responsible for secure handling and storage of templates.
