# HTTP API Contract: Biometric Bridge

**Phase**: 1 — Design  
**Date**: 2026-04-10  
**Base URL**: `http://127.0.0.1:7070`

## Authentication

All endpoints except `/healthz` require a JWT in the `Authorization: Bearer <token>` header. The JWT must:
- Use `ES256` algorithm
- Have `aud` = `"biometric-bridge"`
- Have `iss` matching the configured `token_issuer`
- Have a valid, non-expired `exp` claim
- Be signed with the ECDSA private key corresponding to the configured public key

Failure to authenticate returns `401 Unauthorized`.

## CORS

- `Access-Control-Allow-Origin`: configured `allowed_origin` (single origin, never `*`)
- `Access-Control-Allow-Methods`: `GET, POST, OPTIONS`
- `Access-Control-Allow-Headers`: `Authorization, Content-Type`
- `Access-Control-Max-Age`: `3600`
- Preflight `OPTIONS` requests are handled automatically

---

## Endpoints

### `GET /healthz`

**Auth**: None  
**Purpose**: Confirm the bridge is running. Used by installers and monitoring tools.

**Response**: `200 OK`
```json
{ "status": "ok" }
```

**Errors**: None (if the server is listening, it responds).

---

### `GET /api/devices`

**Auth**: JWT required  
**Purpose**: List all connected devices with metadata and capability flags.

**Response**: `200 OK`
```json
{
  "devices": [
    {
      "id": "reception",
      "name": "reception",
      "model": "BioStation 2",
      "firmwareVersion": "2.8.0",
      "fingerSupported": true
    },
    {
      "id": "server-room",
      "name": "server-room",
      "model": "BioEntry W2",
      "firmwareVersion": "1.5.1",
      "fingerSupported": true
    }
  ]
}
```

**Notes**:
- `id` and `name` both use the human-readable device name from config (SDK-agnostic)
- The list reflects devices that successfully connected at startup
- During reconnection, a device still appears in the list (state is observable via events)

**Errors**:
| Status | Body | Condition |
|--------|------|-----------|
| 401 | `{"error": "unauthorized"}` | Missing/invalid/expired JWT |

---

### `POST /api/enroll`

**Auth**: JWT required  
**Purpose**: Enroll a user's fingerprint on a device (two impressions).

**Request**:
```json
{
  "deviceId": "reception",
  "userId": "user-001",
  "userName": "Jane Smith"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `deviceId` | string | yes | Device name from config |
| `userId` | string | yes | Unique user identifier |
| `userName` | string | yes | User display name |

**Response**: `200 OK`
```json
{ "ok": true, "userId": "user-001" }
```

**Errors**:
| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "invalid request: ..."}` | Missing required fields, malformed JSON |
| 401 | `{"error": "unauthorized"}` | Missing/invalid/expired JWT |
| 409 | `{"error": "device busy"}` | Another scan/enroll in progress on this device |
| 502 | `{"error": "driver error: ..."}` | SDK call failed (e.g., finger quality too low, enrollment rejected) |
| 503 | `{"error": "device unavailable: reception"}` | Device not connected or reconnecting |
| 504 | `{"error": "scan timeout"}` | User did not place finger within 10 seconds |

---

### `POST /api/slap-scan`

**Auth**: JWT required  
**Purpose**: Capture multiple fingers simultaneously on readers that support a flat-bed platen (e.g., RealScan G10). Returns the full slap image and individually segmented finger images with quality scores.

**Request**:
```json
{
  "deviceId": "reception",
  "mode": "right_four"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `deviceId` | string | yes | Device name from config |
| `mode` | string | yes | Capture group: `left_four`, `right_four`, or `two_thumbs` |

**Response**: `200 OK`
```json
{
  "mode": "right_four",
  "slapImage": "SGVsbG8gV29ybGQ=",
  "slapWidth": 1600,
  "slapHeight": 1500,
  "fingers": [
    {
      "finger": "right_index",
      "image": "SGVsbG8gV29ybGQ=",
      "width": 300,
      "height": 400,
      "quality": 78
    },
    {
      "finger": "right_middle",
      "image": "SGVsbG8gV29ybGQ=",
      "width": 300,
      "height": 400,
      "quality": 82
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `mode` | string | Echo of the requested capture mode |
| `slapImage` | string | Base64-encoded full slap image (raw 8-bit grayscale) |
| `slapWidth` | integer | Full slap image width in pixels |
| `slapHeight` | integer | Full slap image height in pixels |
| `fingers` | array | Individually segmented finger captures |
| `fingers[].finger` | string | Finger position (e.g., `"right_index"`); present in every element but may be an empty string (`""`) when the SDK could not identify the finger during segmentation |
| `fingers[].image` | string | Base64-encoded raw 8-bit grayscale image for this finger |
| `fingers[].width` | integer | Image width in pixels |
| `fingers[].height` | integer | Image height in pixels |
| `fingers[].quality` | integer | 0–100 NIST quality score |

**Errors**:
| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "invalid request: ..."}` | Missing fields, malformed JSON, or unrecognized `mode` |
| 401 | `{"error": "unauthorized"}` | Missing/invalid/expired JWT |
| 409 | `{"error": "device busy"}` | Another scan/enroll in progress on this device |
| 501 | `{"error": "slap capture not supported by this driver"}` | Active driver does not support multi-finger capture |
| 502 | `{"error": "driver error: ..."}` | SDK call failed |
| 503 | `{"error": "device unavailable: reception"}` | Device not connected or reconnecting |
| 504 | `{"error": "slap scan timeout"}` | User did not place fingers within 20 seconds |

---

### `POST /api/scan`

**Auth**: JWT required  
**Purpose**: Capture one fingerprint impression and return the raw template.

**Request**:
```json
{
  "deviceId": "reception"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `deviceId` | string | yes | Device name from config |

**Response**: `200 OK`
```json
{
  "template": "SGVsbG8gV29ybGQ=",
  "quality": 82
}
```

| Field | Type | Description |
|-------|------|-------------|
| `template` | string | Base64-encoded raw fingerprint template bytes |
| `quality` | integer | 0–100 quality score from the SDK |

**Errors**:
| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "invalid request: ..."}` | Missing `deviceId`, malformed JSON |
| 401 | `{"error": "unauthorized"}` | Missing/invalid/expired JWT |
| 409 | `{"error": "device busy"}` | Another scan/enroll in progress on this device |
| 502 | `{"error": "driver error: ..."}` | SDK call failed |
| 503 | `{"error": "device unavailable: reception"}` | Device not connected or reconnecting |
| 504 | `{"error": "scan timeout"}` | User did not place finger within 10 seconds |

---

## Error Response Format

All error responses follow this shape:

```json
{
  "error": "<human-readable error message>"
}
```

Content-Type is always `application/json`.

## HTTP Status Code Summary

| Code | Meaning | Used By |
|------|---------|---------|
| 200 | Success | All endpoints on success |
| 400 | Malformed request body | `/api/enroll`, `/api/scan`, `/api/slap-scan` |
| 401 | Missing, invalid, or expired JWT | All authenticated endpoints |
| 409 | Device busy (operation in progress) | `/api/enroll`, `/api/scan`, `/api/slap-scan` |
| 501 | Feature not supported by driver | `/api/slap-scan` |
| 502 | Driver-level failure (SDK error) | `/api/enroll`, `/api/scan`, `/api/slap-scan` |
| 503 | Device not connected | `/api/enroll`, `/api/scan`, `/api/slap-scan` |
| 504 | Scan timeout (finger not placed) | `/api/enroll`, `/api/scan`, `/api/slap-scan` |
