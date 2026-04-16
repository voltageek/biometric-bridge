# API Contract: POST /api/slap-enroll

**Feature**: 004-slap-enroll | **Date**: 2026-04-16 | **Plan**: [../plan.md](../plan.md)

## Endpoint Summary

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/slap-enroll | JWT Bearer | Capture 2 slap impressions with quality validation and retry logic |

## Request

### Headers

| Header | Required | Value |
|--------|----------|-------|
| Authorization | Yes | `Bearer <jwt-token>` |
| Content-Type | Yes | `application/json` |

### Body

```json
{
  "deviceId": "RealScan-G10",
  "userId": "user-12345",
  "userName": "John Doe",
  "mode": "right_four",
  "minQuality": 60,
  "maxRetries": 3,
  "strict": true
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| deviceId | string | Yes | — | Device name from configuration |
| userId | string | Yes | — | User identifier for logging and events |
| userName | string | Yes | — | User display name for logging |
| mode | string | Yes | — | Capture mode: `left_four`, `right_four`, or `two_thumbs` |
| minQuality | int | No | 60 | Minimum NIST quality score (0-100). 0 disables validation. |
| maxRetries | int | No | 3 | Maximum retry attempts per impression (0-10). 0 = no retries. |
| strict | bool | No | true | If true, reject when fewer fingers detected than expected |

## Response

### 200 OK - Enrollment Successful

```json
{
  "ok": true,
  "userId": "user-12345",
  "impressions": [
    {
      "impressionNumber": 1,
      "mode": "right_four",
      "slapImage": "base64-encoded-slap-image...",
      "slapWidth": 1600,
      "slapHeight": 1500,
      "fingers": [
        {
          "finger": "right_index",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 78
        },
        {
          "finger": "right_middle",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 82
        },
        {
          "finger": "right_ring",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 75
        },
        {
          "finger": "right_little",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 71
        }
      ]
    },
    {
      "impressionNumber": 2,
      "mode": "right_four",
      "slapImage": "base64-encoded-slap-image...",
      "slapWidth": 1600,
      "slapHeight": 1500,
      "fingers": [
        {
          "finger": "right_index",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 80
        },
        {
          "finger": "right_middle",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 85
        },
        {
          "finger": "right_ring",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 77
        },
        {
          "finger": "right_little",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 73
        }
      ]
    }
  ],
  "totalRetries": 0
}
```

### 400 Bad Request - Validation Error

```json
{"error": "invalid request: deviceId is required"}
```

```json
{"error": "invalid request: mode is required (left_four, right_four, two_thumbs)"}
```

```json
{"error": "invalid request: unrecognized mode; use left_four, right_four, or two_thumbs"}
```

### 401 Unauthorized - Invalid/Missing JWT

```json
{"error": "unauthorized"}
```

### 409 Conflict - Device Busy

```json
{"error": "device busy"}
```

### 422 Unprocessable Entity - Quality Validation Failed

Returned when all retry attempts exhausted and quality still below threshold.

```json
{
  "error": "enrollment failed: quality threshold not met after 3 retries",
  "failedFingers": [
    {"finger": "right_index", "quality": 45, "threshold": 60},
    {"finger": "right_ring", "quality": 52, "threshold": 60}
  ]
}
```

### 422 Unprocessable Entity - Insufficient Fingers (Strict Mode)

Returned when strict=true and fewer fingers detected than expected.

```json
{
  "error": "enrollment failed: expected 4 fingers, detected 3 (strict mode)",
  "detected": 3,
  "expected": 4
}
```

### 501 Not Implemented - Unsupported Driver

Returned when the device/driver does not support slap capture (e.g., BS2 devices).

```json
{"error": "slap enrollment not supported by this driver"}
```

### 503 Service Unavailable - Device Disconnected

```json
{"error": "device unavailable: RealScan-G10"}
```

### 504 Gateway Timeout

Returned when enrollment exceeds the overall timeout (60s) or per-slap timeout (20s).

```json
{"error": "slap enrollment timeout"}
```

## WebSocket Events

During enrollment, the following events may be emitted on the `/events` WebSocket connection:

### enrollment_retry

Emitted when a slap capture fails quality validation and is being retried.

```json
{
  "type": "enrollment_retry",
  "deviceId": "RealScan-G10",
  "userId": "user-12345",
  "attempt": 2,
  "impressionNum": 1,
  "failedFingers": [
    {"finger": "right_index", "quality": 45, "threshold": 60}
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| type | string | Always `"enrollment_retry"` |
| deviceId | string | Device name |
| userId | string | User being enrolled |
| attempt | int | Current attempt number (2 = first retry, 3 = second retry, etc.) |
| impressionNum | int | Which impression failed (1 or 2) |
| failedFingers | array | Fingers that failed quality validation |

## Behavioral Notes

### Capture Flow

1. Acquire device lock from registry
2. For impressionNumber = 1, 2:
   a. Attempt slap capture (20s timeout)
   b. Validate finger count (if strict=true)
   c. Validate quality for each finger (if minQuality > 0)
   d. If validation fails and retries remaining:
      - Emit `enrollment_retry` event
      - Retry capture (goto step a)
   e. If validation fails and no retries:
      - Release device lock
      - Return 422 with failure details
   f. Store impression result
3. Release device lock
4. Return 200 with both impressions

### Timing

- Overall enrollment timeout: 60 seconds
- Per-slap capture timeout: 20 seconds
- Allows for 2 captures + up to 3 retries within overall timeout

### Quality Validation

- Quality is validated per-finger after each capture
- If ANY finger fails quality threshold, the entire impression is retried
- Quality scores are NIST scores (0-100) from the SDK
- Setting `minQuality: 0` disables quality validation

### Strict Mode

- `strict: true` (default): Enrollment fails if detected fingers < expected count
- `strict: false`: Accepts partial enrollment with available fingers
- Expected counts: `left_four`=4, `right_four`=4, `two_thumbs`=2

### Device Locking

- Device is acquired before capture begins
- Device is released after enrollment completes (success or failure)
- Concurrent enrollment requests to the same device return 409

## Example cURL

```bash
curl -X POST http://127.0.0.1:7070/api/slap-enroll \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": "RealScan-G10",
    "userId": "user-12345",
    "userName": "John Doe",
    "mode": "right_four",
    "minQuality": 60,
    "maxRetries": 3,
    "strict": true
  }'
```

## Related Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/slap-scan` | Single slap capture without enrollment logic |
| `POST /api/enroll` | Single-finger enrollment (2 impressions) |
| `POST /api/scan` | Single-finger scan |
| `GET /api/devices` | List connected devices |
