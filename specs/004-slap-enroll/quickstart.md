# Quickstart: Slap Enrollment

**Feature**: 004-slap-enroll | **Date**: 2026-04-16 | **Plan**: [plan.md](./plan.md)

## Prerequisites

1. **Bridge running** with RealScan G10 device connected
2. **Valid JWT token** from your backend server
3. **WebSocket connection** to `/events` (optional, for retry notifications)

## Basic Usage

### Enroll Right Hand (4 Fingers)

```bash
# Get a JWT token from your backend
JWT_TOKEN="eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9..."

# Enroll right hand (index, middle, ring, little fingers)
curl -X POST http://127.0.0.1:7070/api/slap-enroll \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": "RealScan-G10",
    "userId": "user-12345",
    "userName": "John Doe",
    "mode": "right_four"
  }'
```

### Enroll Left Hand

```bash
curl -X POST http://127.0.0.1:7070/api/slap-enroll \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": "RealScan-G10",
    "userId": "user-12345",
    "userName": "John Doe",
    "mode": "left_four"
  }'
```

### Enroll Both Thumbs

```bash
curl -X POST http://127.0.0.1:7070/api/slap-enroll \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": "RealScan-G10",
    "userId": "user-12345",
    "userName": "John Doe",
    "mode": "two_thumbs"
  }'
```

## Configuration Options

### Quality Threshold

Set minimum quality score (0-100). Default is 60.

```json
{
  "deviceId": "RealScan-G10",
  "userId": "user-12345",
  "userName": "John Doe",
  "mode": "right_four",
  "minQuality": 70
}
```

- `minQuality: 0` disables quality validation (accept any capture)
- `minQuality: 60` (default) is a good balance of quality and usability
- `minQuality: 80+` may require multiple attempts for some users

### Retry Attempts

Set maximum retries per impression (0-10). Default is 3.

```json
{
  "deviceId": "RealScan-G10",
  "userId": "user-12345",
  "userName": "John Doe",
  "mode": "right_four",
  "maxRetries": 5
}
```

- `maxRetries: 0` means single attempt only (no retries)
- `maxRetries: 3` (default) allows up to 4 total attempts per impression

### Strict Mode

Control behavior when fewer fingers detected than expected.

```json
{
  "deviceId": "RealScan-G10",
  "userId": "user-12345",
  "userName": "John Doe",
  "mode": "right_four",
  "strict": false
}
```

- `strict: true` (default) - Fails if fewer fingers detected
- `strict: false` - Accepts partial enrollment

## Handling Responses

### Success (200 OK)

```javascript
const response = await fetch('/api/slap-enroll', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    deviceId: 'RealScan-G10',
    userId: 'user-12345',
    userName: 'John Doe',
    mode: 'right_four'
  })
});

const result = await response.json();
// result.ok === true
// result.impressions.length === 2
// Each impression has 4 fingers (for right_four mode)

// Store templates on your server
for (const impression of result.impressions) {
  for (const finger of impression.fingers) {
    await saveTemplate({
      userId: result.userId,
      finger: finger.finger,
      template: finger.template,
      quality: finger.quality,
      impressionNumber: impression.impressionNumber
    });
  }
}
```

### Quality Failure (422)

```javascript
if (response.status === 422) {
  const error = await response.json();
  if (error.failedFingers) {
    // Quality validation failed
    console.log('Fingers with low quality:', error.failedFingers);
    // Show user which fingers need better placement
  } else if (error.detected !== undefined) {
    // Missing fingers (strict mode)
    console.log(`Detected ${error.detected} of ${error.expected} expected`);
  }
}
```

### Device Errors

```javascript
switch (response.status) {
  case 409:
    // Device busy - another operation in progress
    alert('Scanner is busy. Please wait.');
    break;
  case 501:
    // Slap not supported (e.g., BS2 device)
    alert('This scanner does not support slap capture.');
    break;
  case 503:
    // Device disconnected
    alert('Scanner is not connected.');
    break;
  case 504:
    // Timeout
    alert('Capture timed out. Please try again.');
    break;
}
```

## WebSocket Events

Listen for retry events to show progress to the user:

```javascript
const ws = new WebSocket(`ws://127.0.0.1:7070/events?token=${token}`);

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  if (data.type === 'enrollment_retry') {
    console.log(`Retry attempt ${data.attempt} for impression ${data.impressionNum}`);
    console.log('Low quality fingers:', data.failedFingers);
    
    // Show user feedback
    showRetryNotification({
      attempt: data.attempt,
      failedFingers: data.failedFingers.map(f => f.finger)
    });
  }
};
```

## Demo Mode Testing

When running with `--demo` flag, the bridge simulates slap enrollment:

```bash
# Start bridge in demo mode
./bridge --demo

# Test enrollment (no hardware required)
curl -X POST http://127.0.0.1:7070/api/slap-enroll \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": "Demo-Device",
    "userId": "test-user",
    "userName": "Test User",
    "mode": "right_four"
  }'
```

## Full Enrollment Workflow

Complete 10-finger enrollment using slap capture:

```javascript
async function enrollAllFingers(userId, userName, deviceId, token) {
  const results = [];
  
  // Right hand: index, middle, ring, little
  const rightHand = await slapEnroll(deviceId, userId, userName, 'right_four', token);
  results.push(...rightHand.impressions);
  
  // Left hand: index, middle, ring, little
  const leftHand = await slapEnroll(deviceId, userId, userName, 'left_four', token);
  results.push(...leftHand.impressions);
  
  // Both thumbs
  const thumbs = await slapEnroll(deviceId, userId, userName, 'two_thumbs', token);
  results.push(...thumbs.impressions);
  
  return results;
}

async function slapEnroll(deviceId, userId, userName, mode, token) {
  const response = await fetch('http://127.0.0.1:7070/api/slap-enroll', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ deviceId, userId, userName, mode })
  });
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error);
  }
  
  return response.json();
}
```

## Timing Expectations

| Mode | Expected Time (no retries) | With 1 retry per impression |
|------|---------------------------|----------------------------|
| right_four | ~15-20s | ~25-35s |
| left_four | ~15-20s | ~25-35s |
| two_thumbs | ~10-15s | ~20-25s |
| **Full 10-finger** | **~40-55s** | **~70-95s** |

Compare to sequential single-finger enrollment: ~80-120s for 10 fingers.
