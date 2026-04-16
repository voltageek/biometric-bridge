# Data Model: Slap Enrollment

**Feature**: 004-slap-enroll | **Date**: 2026-04-16 | **Plan**: [plan.md](./plan.md)

## Overview

This document defines the data structures for the slap enrollment feature. All structures are API-layer types defined in `internal/api/slap_enroll.go`. The driver layer uses existing types (`driver.SlapScanResult`, `driver.ScanResult`, `driver.CaptureMode`).

## Request Types

### SlapEnrollRequest

JSON body for `POST /api/slap-enroll`.

```go
// slapEnrollRequest is the JSON body for POST /api/slap-enroll.
type slapEnrollRequest struct {
    DeviceID   string `json:"deviceId"`            // Required: device name from config
    UserID     string `json:"userId"`              // Required: user identifier for logging/events
    UserName   string `json:"userName"`            // Required: user display name for logging
    Mode       string `json:"mode"`                // Required: "left_four", "right_four", or "two_thumbs"
    MinQuality int    `json:"minQuality,omitempty"` // Optional: 0-100, default from config (60)
    MaxRetries int    `json:"maxRetries,omitempty"` // Optional: 0-10, default from config (3)
    Strict     *bool  `json:"strict,omitempty"`     // Optional: reject if fewer fingers detected (default true)
}
```

**Validation Rules**:
- `deviceId`: Required, non-empty, must exist in device registry
- `userId`: Required, non-empty
- `userName`: Required, non-empty
- `mode`: Required, must be one of `left_four`, `right_four`, `two_thumbs`
- `minQuality`: Optional, 0-100 range; 0 means no quality validation
- `maxRetries`: Optional, 0-10 range; 0 means no retries (single attempt)
- `strict`: Optional pointer to distinguish "not provided" from "false"; defaults to true

## Response Types

### SlapEnrollResult (Success - 200 OK)

```go
// slapEnrollResult is the JSON response for successful slap enrollment.
type slapEnrollResult struct {
    OK           bool                    `json:"ok"`           // Always true for success
    UserID       string                  `json:"userId"`       // Echo back the userId
    Impressions  []slapEnrollImpression  `json:"impressions"`  // Array of 2 impressions
    TotalRetries int                     `json:"totalRetries"` // Total retry attempts across all impressions
}
```

### SlapEnrollImpression

```go
// slapEnrollImpression represents a single slap capture (one of the 2 impressions).
type slapEnrollImpression struct {
    ImpressionNumber int                   `json:"impressionNumber"` // 1 or 2
    Mode             string                `json:"mode"`             // Capture mode used
    SlapImage        string                `json:"slapImage"`        // Base64-encoded full slap image (raw grayscale)
    SlapWidth        int                   `json:"slapWidth"`        // Full slap image width in pixels
    SlapHeight       int                   `json:"slapHeight"`       // Full slap image height in pixels
    Fingers          []slapEnrolledFinger  `json:"fingers"`          // Segmented finger templates
}
```

### SlapEnrolledFinger

```go
// slapEnrolledFinger represents a single segmented finger from a slap capture.
type slapEnrolledFinger struct {
    Finger   string `json:"finger"`   // Finger position: "right_index", "left_thumb", etc.
    Template string `json:"template"` // Base64-encoded finger template (raw grayscale)
    Width    int    `json:"width"`    // Finger image width in pixels
    Height   int    `json:"height"`   // Finger image height in pixels
    Quality  int    `json:"quality"`  // NIST quality score (0-100)
}
```

### Error Responses

#### 400 Bad Request - Validation Error
```json
{"error": "invalid request: deviceId is required"}
{"error": "invalid request: mode is required (left_four, right_four, two_thumbs)"}
{"error": "invalid request: unrecognized mode; use left_four, right_four, or two_thumbs"}
{"error": "invalid request: minQuality must be 0-100"}
{"error": "invalid request: maxRetries must be 0-10"}
```

#### 409 Conflict - Device Busy
```json
{"error": "device busy"}
```

#### 422 Unprocessable Entity - Quality/Finger Count Failure
```json
{
  "error": "enrollment failed: quality threshold not met after 3 retries",
  "failedFingers": [
    {"finger": "right_index", "quality": 45, "threshold": 60},
    {"finger": "right_ring", "quality": 52, "threshold": 60}
  ]
}
```

```json
{
  "error": "enrollment failed: expected 4 fingers, detected 3 (strict mode)",
  "detected": 3,
  "expected": 4
}
```

#### 501 Not Implemented - Unsupported Driver
```json
{"error": "slap enrollment not supported by this driver"}
```

#### 503 Service Unavailable - Device Disconnected
```json
{"error": "device unavailable: device-name"}
```

#### 504 Gateway Timeout
```json
{"error": "slap enrollment timeout"}
```

## WebSocket Event Types

### enrollment_retry Event

Emitted when a slap capture fails quality validation and is being retried.

```go
// enrollmentRetryEvent is emitted via WebSocket when a retry occurs.
type enrollmentRetryEvent struct {
    Type          string               `json:"type"`          // Always "enrollment_retry"
    DeviceName    string               `json:"deviceId"`      // Device name
    UserID        string               `json:"userId"`        // User being enrolled
    Attempt       int                  `json:"attempt"`       // Current attempt number (2 = first retry)
    ImpressionNum int                  `json:"impressionNum"` // Which impression failed (1 or 2)
    FailedFingers []failedFingerInfo   `json:"failedFingers"` // Fingers that failed quality
}

type failedFingerInfo struct {
    Finger    string `json:"finger"`    // Finger position
    Quality   int    `json:"quality"`   // Actual quality score
    Threshold int    `json:"threshold"` // Required threshold
}
```

**Example**:
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

## Expected Finger Counts by Mode

| Mode | Expected Fingers | Finger Positions |
|------|-----------------|------------------|
| `left_four` | 4 | left_index, left_middle, left_ring, left_little |
| `right_four` | 4 | right_index, right_middle, right_ring, right_little |
| `two_thumbs` | 2 | left_thumb, right_thumb |

## Configuration Defaults

Added to `config.BridgeConfig` (or as constants in the handler):

```go
const (
    defaultSlapMinQuality = 60  // NIST quality threshold
    defaultSlapMaxRetries = 3   // Maximum retry attempts per impression
    defaultSlapTimeout    = 60 * time.Second  // Overall enrollment timeout
    perSlapTimeout        = 20 * time.Second  // Per-slap capture timeout
)
```

## Relationship to Existing Types

The API handler converts between API types and driver types:

```
slapEnrollRequest.Mode -> driver.CaptureMode (validated)
driver.SlapScanResult  -> slapEnrollImpression (converted)
driver.ScanResult      -> slapEnrolledFinger (converted, base64 encoded)
```

No new driver-layer types are required. The existing `driver.SlapScanResult` contains:
- `SlapImage []byte` - Full slap image
- `SlapWidth`, `SlapHeight int` - Dimensions
- `Fingers []ScanResult` - Segmented finger data with quality scores

## State Transitions

### Enrollment Flow State Machine

```
[Request Received]
        │
        ▼
[Validate Request] ──400──▶ [Error Response]
        │
        │ valid
        ▼
[Acquire Device Lock] ──409──▶ [Device Busy]
        │
        │ acquired
        ▼
┌─────────────────────────────────────┐
│  For impression = 1, 2:             │
│       │                             │
│       ▼                             │
│  [Capture Slap] ──504──▶ [Timeout]  │
│       │                             │
│       │ success                     │
│       ▼                             │
│  [Check Finger Count] ──422──▶ [Strict Failure]
│       │                             │
│       │ ok (or lenient)             │
│       ▼                             │
│  [Check Quality] ─┬──▶ [Retry?]     │
│       │           │     │           │
│       │           │  yes│  ≤maxRetries
│       │           │     │           │
│       │           │     ▼           │
│       │           │  [Emit Event]   │
│       │           │     │           │
│       │           │     └──────┐    │
│       │           │            │    │
│       │           │ no (exhausted)  │
│       │           │            │    │
│       │           │            ▼    │
│       │           └────422────▶ [Quality Failure]
│       │                             │
│       │ all pass                    │
│       ▼                             │
│  [Store Impression]                 │
│       │                             │
└───────┼─────────────────────────────┘
        │
        │ both complete
        ▼
[Release Device Lock]
        │
        ▼
[Return 200 OK]
```
