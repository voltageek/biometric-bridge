Implementation Plan: Slap Enrollment Feature
Overview
Add a new /api/slap-enroll endpoint to capture 2 impressions of 4 fingers simultaneously, with quality validation and retry logic.
Goals
- Speed: Enroll 4 fingers in ~40 seconds (2 slaps × 20s) vs ~80 seconds for sequential enrollment (4 fingers × 2 impressions × 10s)
- Reliability: Maintain 2-impression quality standard with configurable quality thresholds
- Flexibility: Support configurable quality thresholds and strict/lenient finger detection modes
---
Architecture Decisions
Based on research and your preferences:
Aspect
Impressions
Endpoint
BS2 Support
Template Storage
Quality Threshold
Retries
Partial Detection
Scope
---
Implementation Tasks
1. Configuration Updates
File: internal/config/config.go
Add new quality control configuration:
type Config struct {
    // ... existing fields ...
    
    Quality QualityConfig `yaml:"quality,omitempty"`
}
type QualityConfig struct {
    MinEnrollmentQuality int `yaml:"min_enrollment_quality"` // Default: 60
    MaxEnrollmentRetries int `yaml:"max_enrollment_retries"` // Default: 3
}
File: config.yaml
quality:
  min_enrollment_quality: 60  # NIST quality score 0-100
  max_enrollment_retries: 3   # Max retry attempts
Validation:
- Quality must be 0-100
- Retries must be 0-10
---
2. Driver Interface Extension
File: internal/driver/driver.go
Add new method to Driver interface:
// SlapEnrollRequest contains parameters for slap enrollment
type SlapEnrollRequest struct {
    DeviceName string
    UserID     string
    UserName   string
    Mode       CaptureMode // left_four, right_four, two_thumbs
    Strict     bool        // If true, reject if any finger missing
    MinQuality int         // Minimum NIST quality per finger (0 = no validation)
    MaxRetries int         // Maximum retry attempts
}
// SlapEnrollResult contains the outcome of slap enrollment
type SlapEnrollResult struct {
    Impressions []SlapEnrollImpression // 2 impressions
    TotalRetries int                   // Retry count used
}
type SlapEnrollImpression struct {
    ImpressionNumber int          // 1 or 2
    Mode            CaptureMode   // Capture mode used
    SlapImage       []byte        // Full slap image
    SlapWidth       int
    SlapHeight      int
    Fingers         []SlapEnrolledFinger
}
type SlapEnrolledFinger struct {
    Finger   FingerPosition
    Template []byte // Segmented finger image
    Width    int
    Height   int
    Quality  int    // NIST quality score
}
// Driver interface addition:
type Driver interface {
    // ... existing methods ...
    
    // SlapEnroll performs multi-impression slap enrollment.
    // Captures multiple impressions of multiple fingers simultaneously,
    // validates quality, and retries if needed.
    // Returns ErrSlapNotSupported if driver doesn't support slap capture.
    SlapEnroll(ctx context.Context, req SlapEnrollRequest) (*SlapEnrollResult, error)
}
Add new error types:
var (
    // ... existing errors ...
    
    ErrQualityBelowThreshold = errors.New("finger quality below threshold")
    ErrMissingFingers        = errors.New("expected fingers not detected")
)
---
3. RealScan Driver Implementation
File: internal/driver/realscan/driver.go
Implement SlapEnroll:
// SlapEnroll performs multi-impression slap enrollment with quality validation.
func (d *RSDriver) SlapEnroll(ctx context.Context, req driver.SlapEnrollRequest) (*driver.SlapEnrollResult, error) {
    dev, ok := d.devices[req.DeviceName]
    if !ok {
        return nil, fmt.Errorf("device %q not found", req.DeviceName)
    }
    result := &driver.SlapEnrollResult{
        Impressions: make([]driver.SlapEnrollImpression, 0, 2),
    }
    // Capture 2 impressions with retry logic
    for impressionNum := 1; impressionNum <= 2; impressionNum++ {
        impression, retries, err := d.captureSlapImpressionWithRetry(
            ctx, dev, req, impressionNum)
        
        if err != nil {
            return nil, err
        }
        
        result.Impressions = append(result.Impressions, *impression)
        result.TotalRetries += retries
    }
    return result, nil
}
// captureSlapImpressionWithRetry captures one slap impression with quality retry logic
func (d *RSDriver) captureSlapImpressionWithRetry(
    ctx context.Context,
    dev *rsDevice,
    req driver.SlapEnrollRequest,
    impressionNum int,
) (*driver.SlapEnrollImpression, int, error) {
    
    maxRetries := req.MaxRetries
    if maxRetries < 0 {
        maxRetries = 0
    }
    for attempt := 0; attempt <= maxRetries; attempt++ {
        select {
        case <-ctx.Done():
            return nil, attempt, ctx.Err()
        default:
        }
        // Emit retry event if not first attempt
        if attempt > 0 {
            d.emitEvent(driver.Event{
                Type:       "enrollment_retry",
                DeviceName: req.DeviceName,
                Data: map[string]interface{}{
                    "impression": impressionNum,
                    "attempt":    attempt,
                    "max_retries": maxRetries,
                    "reason":     "quality_validation_failed",
                },
            })
            
            // Brief delay between retries
            time.Sleep(500 * time.Millisecond)
        }
        // Capture slap
        impression, err := d.captureSlapImpression(ctx, dev, req, impressionNum)
        if err != nil {
            return nil, attempt, err
        }
        // Validate quality if threshold set
        if req.MinQuality > 0 {
            qualityErr := d.validateSlapQuality(impression, req)
            if qualityErr != nil {
                if attempt < maxRetries {
                    slog.Warn("slap quality validation failed, retrying",
                        "device", req.DeviceName,
                        "impression", impressionNum,
                        "attempt", attempt+1,
                        "error", qualityErr,
                    )
                    continue // Retry
                }
                // Max retries exhausted
                return nil, attempt, fmt.Errorf("quality validation failed after %d retries: %w",
                    maxRetries, qualityErr)
            }
        }
        // Success!
        return impression, attempt, nil
    }
    return nil, maxRetries, fmt.Errorf("unreachable")
}
// captureSlapImpression performs a single slap capture
func (d *RSDriver) captureSlapImpression(
    ctx context.Context,
    dev *rsDevice,
    req driver.SlapEnrollRequest,
    impressionNum int,
) (*driver.SlapEnrollImpression, error) {
    
    handle := dev.handle
    mode := req.Mode
    
    // Set LED based on mode
    ledMask := getSlapLEDMask(mode)
    C.sdk_set_capture_led(handle, C.int(ledMask))
    // Capture with timeout
    var imageData *C.uchar
    var width, height C.int
    var slapInfo *C.RSSlapInfo
    var numOfFinger C.int
    timeoutMS := C.int(defaultSlapCaptureTimeoutMS)
    rc := C.sdk_take_image_data_segment(
        handle,
        &imageData,
        &width,
        &height,
        &slapInfo,
        &numOfFinger,
        timeoutMS,
    )
    if rc != C.RS_SUCCESS {
        if rc == C.RS_ERR_CAPTURE_TIMEOUT {
            return nil, fmt.Errorf("%w (impression %d)", driver.ErrScanTimeout, impressionNum)
        }
        if rc == C.RS_ERR_CAPTURE_ABORTED {
            return nil, fmt.Errorf("enrollment cancelled (impression %d)", impressionNum)
        }
        return nil, fmt.Errorf("slap capture failed (impression %d): %s",
            impressionNum, rsErrString(int(rc)))
    }
    defer C.sdk_free_image_data(imageData)
    // Copy slap image
    slapSize := int(width) * int(height)
    slapImageData := C.GoBytes(unsafe.Pointer(imageData), C.int(slapSize))
    // Process each detected finger
    nFingers := int(numOfFinger)
    fingers := make([]driver.SlapEnrolledFinger, 0, nFingers)
    for i := 0; i < nFingers; i++ {
        slapInfoPtr := (*C.RSSlapInfo)(unsafe.Pointer(
            uintptr(unsafe.Pointer(slapInfo)) + uintptr(i)*unsafe.Sizeof(*slapInfo)))
        // Extract finger image
        fImg, fW, fH := extractFingerFromSlap(imageData, width, height, slapInfoPtr)
        if fImg == nil {
            continue
        }
        defer C.sdk_free_image_data(fImg)
        // Get quality score
        var nistQuality C.int
        quality := 0
        qrc := C.sdk_get_quality_score(fImg, C.int(fW), C.int(fH), &nistQuality)
        if qrc == C.RS_SUCCESS {
            quality = int(nistQuality)
        } else if slapInfoPtr.imageQuality > 0 {
            quality = int(slapInfoPtr.imageQuality)
        }
        // Map finger type
        fingerType := int(slapInfoPtr.fingerType)
        fingerPos := driver.FingerPosition("")
        if pos, ok := slapFingerTypeToPosition[fingerType]; ok {
            fingerPos = pos
        }
        // Copy template
        templateSize := int(fW) * int(fH)
        templateData := C.GoBytes(unsafe.Pointer(fImg), C.int(templateSize))
        fingers = append(fingers, driver.SlapEnrolledFinger{
            Finger:   fingerPos,
            Template: templateData,
            Width:    int(fW),
            Height:   int(fH),
            Quality:  quality,
        })
    }
    // Success beep
    C.sdk_beep(handle, C.int(rsBeepPattern1))
    impression := &driver.SlapEnrollImpression{
        ImpressionNumber: impressionNum,
        Mode:            mode,
        SlapImage:       slapImageData,
        SlapWidth:       int(width),
        SlapHeight:      int(height),
        Fingers:         fingers,
    }
    return impression, nil
}
// validateSlapQuality validates finger quality and detection
func (d *RSDriver) validateSlapQuality(
    impression *driver.SlapEnrollImpression,
    req driver.SlapEnrollRequest,
) error {
    
    expectedFingers := getExpectedFingersForMode(req.Mode)
    detectedCount := len(impression.Fingers)
    
    // Check finger count
    if req.Strict && detectedCount < len(expectedFingers) {
        return fmt.Errorf("%w: expected %d, detected %d",
            driver.ErrMissingFingers,
            len(expectedFingers),
            detectedCount,
        )
    }
    // Check quality of each detected finger
    var lowQualityFingers []string
    for _, finger := range impression.Fingers {
        if finger.Quality < req.MinQuality {
            lowQualityFingers = append(lowQualityFingers,
                fmt.Sprintf("%s (%d/%d)", finger.Finger, finger.Quality, req.MinQuality))
        }
    }
    if len(lowQualityFingers) > 0 {
        return fmt.Errorf("%w: %s",
            driver.ErrQualityBelowThreshold,
            strings.Join(lowQualityFingers, ", "))
    }
    return nil
}
// Helper: Get expected fingers for capture mode
func getExpectedFingersForMode(mode driver.CaptureMode) []driver.FingerPosition {
    switch mode {
    case driver.CaptureLeftFour:
        return []driver.FingerPosition{
            driver.FingerLeftIndex,
            driver.FingerLeftMiddle,
            driver.FingerLeftRing,
            driver.FingerLeftLittle,
        }
    case driver.CaptureRightFour:
        return []driver.FingerPosition{
            driver.FingerRightIndex,
            driver.FingerRightMiddle,
            driver.FingerRightRing,
            driver.FingerRightLittle,
        }
    case driver.CaptureTwoThumbs:
        return []driver.FingerPosition{
            driver.FingerLeftThumb,
            driver.FingerRightThumb,
        }
    default:
        return nil
    }
}
// Helper: Get LED mask for capture mode
func getSlapLEDMask(mode driver.CaptureMode) int {
    // LED constants from SDK
    const (
        rsLEDLeftFourFingers  = 0x01
        rsLEDRightFourFingers = 0x02
        rsLEDTwoThumbs        = 0x03
    )
    
    switch mode {
    case driver.CaptureLeftFour:
        return rsLEDLeftFourFingers
    case driver.CaptureRightFour:
        return rsLEDRightFourFingers
    case driver.CaptureTwoThumbs:
        return rsLEDTwoThumbs
    default:
        return 0
    }
}
Helper function (may already exist, verify):
// extractFingerFromSlap extracts individual finger image from slap
func extractFingerFromSlap(
    slapImage *C.uchar,
    slapWidth, slapHeight C.int,
    slapInfo *C.RSSlapInfo,
) (*C.uchar, int, int) {
    
    var fImg *C.uchar
    var fW, fH C.int
    
    rc := C.sdk_get_image_from_segment(
        slapImage,
        slapWidth,
        slapHeight,
        slapInfo,
        &fImg,
        &fW,
        &fH,
    )
    
    if rc != C.RS_SUCCESS {
        return nil, 0, 0
    }
    
    return fImg, int(fW), int(fH)
}
---
4. BS2 Driver Implementation
File: internal/driver/bs2/driver.go
// SlapEnroll returns error - BS2 doesn't support slap capture
func (d *BS2Driver) SlapEnroll(ctx context.Context, req driver.SlapEnrollRequest) (*driver.SlapEnrollResult, error) {
    return nil, driver.ErrSlapNotSupported
}
---
5. Demo Driver Implementation
File: internal/driver/demo/driver.go
// SlapEnroll simulates slap enrollment with configurable delays and quality
func (d *Driver) SlapEnroll(ctx context.Context, req driver.SlapEnrollRequest) (*driver.SlapEnrollResult, error) {
    // Simulate 2 impressions
    result := &driver.SlapEnrollResult{
        Impressions: make([]driver.SlapEnrollImpression, 0, 2),
    }
    expectedFingers := getExpectedFingersForMode(req.Mode)
    
    for impressionNum := 1; impressionNum <= 2; impressionNum++ {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        // Simulate capture delay
        time.Sleep(d.cfg.EnrollDelay)
        // Generate mock fingers
        fingers := make([]driver.SlapEnrolledFinger, len(expectedFingers))
        for i, pos := range expectedFingers {
            quality := d.cfg.QualityMin + rand.Intn(d.cfg.QualityMax-d.cfg.QualityMin)
            
            fingers[i] = driver.SlapEnrolledFinger{
                Finger:   pos,
                Template: stubTemplate, // Mock template
                Width:    300,
                Height:   400,
                Quality:  quality,
            }
        }
        impression := driver.SlapEnrollImpression{
            ImpressionNumber: impressionNum,
            Mode:            req.Mode,
            SlapImage:       bytes.Repeat([]byte{0xCD}, 1600*1500),
            SlapWidth:       1600,
            SlapHeight:      1500,
            Fingers:         fingers,
        }
        result.Impressions = append(result.Impressions, impression)
    }
    return result, nil
}
---
6. HTTP API Handler
File: internal/api/slap_enroll.go (NEW)
package api
import (
    "context"
    "encoding/base64"
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"
    "time"
    "github.com/yourusername/biometric-bridge/internal/device"
    "github.com/yourusername/biometric-bridge/internal/driver"
)
type slapEnrollRequest struct {
    DeviceID   string `json:"deviceId"`
    UserID     string `json:"userId"`
    UserName   string `json:"userName"`
    Mode       string `json:"mode"`        // left_four, right_four, two_thumbs
    Strict     *bool  `json:"strict"`      // Optional, defaults to config or true
    MinQuality *int   `json:"minQuality"`  // Optional, defaults to config
    MaxRetries *int   `json:"maxRetries"`  // Optional, defaults to config
}
type slapEnrollResponse struct {
    OK           bool                         `json:"ok"`
    UserID       string                       `json:"userId"`
    Impressions  []slapEnrollImpressionJSON   `json:"impressions"`
    TotalRetries int                          `json:"totalRetries"`
}
type slapEnrollImpressionJSON struct {
    ImpressionNumber int                      `json:"impressionNumber"`
    Mode            string                    `json:"mode"`
    SlapImage       string                    `json:"slapImage"` // base64
    SlapWidth       int                       `json:"slapWidth"`
    SlapHeight      int                       `json:"slapHeight"`
    Fingers         []slapEnrolledFingerJSON  `json:"fingers"`
}
type slapEnrolledFingerJSON struct {
    Finger   string `json:"finger"`
    Template string `json:"template"` // base64
    Width    int    `json:"width"`
    Height   int    `json:"height"`
    Quality  int    `json:"quality"`
}
// NewSlapEnrollHandler returns handler for POST /api/slap-enroll
func NewSlapEnrollHandler(
    d driver.Driver,
    reg *device.Registry,
    defaultMinQuality int,
    defaultMaxRetries int,
) http.HandlerFunc {
    
    return func(w http.ResponseWriter, r *http.Request) {
        var req slapEnrollRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeError(w, http.StatusBadRequest, "invalid request: malformed JSON")
            return
        }
        // Validate required fields
        if req.DeviceID == "" {
            writeError(w, http.StatusBadRequest, "invalid request: deviceId is required")
            return
        }
        if req.UserID == "" {
            writeError(w, http.StatusBadRequest, "invalid request: userId is required")
            return
        }
        if req.UserName == "" {
            writeError(w, http.StatusBadRequest, "invalid request: userName is required")
            return
        }
        if req.Mode == "" {
            writeError(w, http.StatusBadRequest, "invalid request: mode is required")
            return
        }
        // Validate mode
        mode := driver.CaptureMode(req.Mode)
        if !driver.ValidCaptureModes[mode] {
            writeError(w, http.StatusBadRequest,
                "invalid request: mode must be left_four, right_four, or two_thumbs")
            return
        }
        // Apply defaults
        strict := true
        if req.Strict != nil {
            strict = *req.Strict
        }
        
        minQuality := defaultMinQuality
        if req.MinQuality != nil {
            minQuality = *req.MinQuality
            if minQuality < 0 || minQuality > 100 {
                writeError(w, http.StatusBadRequest,
                    "invalid request: minQuality must be 0-100")
                return
            }
        }
        
        maxRetries := defaultMaxRetries
        if req.MaxRetries != nil {
            maxRetries = *req.MaxRetries
            if maxRetries < 0 || maxRetries > 10 {
                writeError(w, http.StatusBadRequest,
                    "invalid request: maxRetries must be 0-10")
                return
            }
        }
        // Acquire device lock
        if err := reg.Acquire(req.DeviceID); err != nil {
            var devErr *device.DeviceError
            if errors.As(err, &devErr) {
                writeError(w, devErr.Code, devErr.Message)
                return
            }
            writeError(w, http.StatusInternalServerError, err.Error())
            return
        }
        defer reg.Release(req.DeviceID)
        // Call driver with timeout (40s for 2 slaps)
        slog.Info("slap enrollment started",
            "device", req.DeviceID,
            "userId", req.UserID,
            "mode", req.Mode,
            "minQuality", minQuality,
            "maxRetries", maxRetries,
        )
        
        ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
        defer cancel()
        driverReq := driver.SlapEnrollRequest{
            DeviceName: req.DeviceID,
            UserID:     req.UserID,
            UserName:   req.UserName,
            Mode:       mode,
            Strict:     strict,
            MinQuality: minQuality,
            MaxRetries: maxRetries,
        }
        result, err := d.SlapEnroll(ctx, driverReq)
        if err != nil {
            slog.Warn("slap enrollment failed",
                "device", req.DeviceID,
                "userId", req.UserID,
                "error", err,
            )
            
            if errors.Is(err, driver.ErrSlapNotSupported) {
                writeError(w, http.StatusNotImplemented,
                    "slap enrollment not supported by this driver")
                return
            }
            if errors.Is(err, driver.ErrQualityBelowThreshold) {
                writeError(w, http.StatusUnprocessableEntity,
                    "enrollment failed: "+err.Error())
                return
            }
            if errors.Is(err, driver.ErrMissingFingers) {
                writeError(w, http.StatusUnprocessableEntity,
                    "enrollment failed: "+err.Error())
                return
            }
            if ctx.Err() == context.DeadlineExceeded || errors.Is(err, driver.ErrScanTimeout) {
                writeError(w, http.StatusGatewayTimeout, "slap enrollment timeout")
                return
            }
            writeError(w, http.StatusBadGateway, "driver error: "+err.Error())
            return
        }
        // Build response
        impressions := make([]slapEnrollImpressionJSON, 0, len(result.Impressions))
        for _, imp := range result.Impressions {
            fingers := make([]slapEnrolledFingerJSON, 0, len(imp.Fingers))
            for _, f := range imp.Fingers {
                fingers = append(fingers, slapEnrolledFingerJSON{
                    Finger:   string(f.Finger),
                    Template: base64.StdEncoding.EncodeToString(f.Template),
                    Width:    f.Width,
                    Height:   f.Height,
                    Quality:  f.Quality,
                })
            }
            impressions = append(impressions, slapEnrollImpressionJSON{
                ImpressionNumber: imp.ImpressionNumber,
                Mode:            string(imp.Mode),
                SlapImage:       base64.StdEncoding.EncodeToString(imp.SlapImage),
                SlapWidth:       imp.SlapWidth,
                SlapHeight:      imp.SlapHeight,
                Fingers:         fingers,
            })
        }
        slog.Info("slap enrollment completed",
            "device", req.DeviceID,
            "userId", req.UserID,
            "impressions", len(impressions),
            "totalRetries", result.TotalRetries,
        )
        writeJSON(w, http.StatusOK, slapEnrollResponse{
            OK:           true,
            UserID:       req.UserID,
            Impressions:  impressions,
            TotalRetries: result.TotalRetries,
        })
    }
}
---
7. Router Registration
File: internal/api/router.go
func NewRouter(deps Dependencies) http.Handler {
    // ... existing code ...
    
    // Load quality config with defaults
    minQuality := 60
    if deps.Config.Quality.MinEnrollmentQuality > 0 {
        minQuality = deps.Config.Quality.MinEnrollmentQuality
    }
    
    maxRetries := 3
    if deps.Config.Quality.MaxEnrollmentRetries >= 0 {
        maxRetries = deps.Config.Quality.MaxEnrollmentRetries
    }
    
    // ... existing route registrations ...
    
    mux.HandleFunc("POST /api/slap-enroll",
        requireAuth(NewSlapEnrollHandler(
            deps.Driver,
            deps.Registry,
            minQuality,
            maxRetries,
        )),
    )
    
    // ... rest of routes ...
}
---
### 8. Events Integration
**File:** `internal/events/handler.go`
The templates will be emitted via existing WebSocket event mechanism. No changes needed if we use the current event emission pattern in the driver.
Verify events are emitted for:
- `enrollment_retry` - when quality validation fails and retrying
- Individual impression captures (if using existing event emission)
---
9. Documentation Updates
File: README.md
Add slap enrollment example:
### Slap Enrollment (Multi-Finger)
Enroll multiple fingers simultaneously in 2 impressions (RealScan G10 only):
```bash
# Enroll right hand (4 fingers, 2 impressions)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "deviceId": "reception",
       "userId": "user-001",
       "userName": "Jane Smith",
       "mode": "right_four",
       "minQuality": 60,
       "strict": true
     }' \
     http://127.0.0.1:7070/api/slap-enroll
# Enroll with custom quality threshold
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "deviceId": "reception",
       "userId": "user-002",
       "userName": "John Doe",
       "mode": "left_four",
       "minQuality": 50,
       "maxRetries": 5,
       "strict": false
     }' \
     http://127.0.0.1:7070/api/slap-enroll
Parameters:
- deviceId (required): Device name from config
- userId (required): User identifier
- userName (required): User display name
- mode (required): left_four, right_four, or two_thumbs
- minQuality (optional): Minimum NIST quality score 0-100 (default: 60)
- maxRetries (optional): Maximum retry attempts 0-10 (default: 3)
- strict (optional): Reject if any expected finger missing (default: true)
Response includes:
- 2 impressions with full slap images
- Per-finger segmented images with quality scores
- Total retry count
**File:** `specs/001-biometric-bridge/contracts/http-api.md`
Add API contract:
```markdown
### `POST /api/slap-enroll`
**Auth**: JWT required  
**Purpose**: Enroll multiple fingers simultaneously using slap capture (2 impressions). Validates quality and retries if needed. RealScan G10 only.
**Request**:
```json
{
  "deviceId": "reception",
  "userId": "user-001",
  "userName": "Jane Smith",
  "mode": "right_four",
  "minQuality": 60,
  "maxRetries": 3,
  "strict": true
}
Parameters:
- deviceId (string, required): Device name
- userId (string, required): User identifier
- userName (string, required): User display name
- mode (string, required): Capture mode - left_four, right_four, or two_thumbs
- minQuality (int, optional): Minimum NIST quality score per finger (0-100, default: config value or 60)
- maxRetries (int, optional): Maximum retry attempts (0-10, default: config value or 3)
- strict (bool, optional): Reject if any expected finger missing (default: true)
Success Response (200):
{
  "ok": true,
  "userId": "user-001",
  "impressions": [
    {
      "impressionNumber": 1,
      "mode": "right_four",
      "slapImage": "base64-encoded-image...",
      "slapWidth": 1600,
      "slapHeight": 1500,
      "fingers": [
        {
          "finger": "right_index",
          "template": "base64-encoded-template...",
          "width": 300,
          "height": 400,
          "quality": 85
        },
        // ... 3 more fingers
      ]
    },
    {
      "impressionNumber": 2,
      // ... second impression
    }
  ],
  "totalRetries": 1
}
Error Responses:
- 400 Bad Request: Invalid parameters
- 401 Unauthorized: Invalid/missing JWT
- 404 Not Found: Device not found
- 409 Conflict: Device busy
- 422 Unprocessable Entity: Quality validation failed after max retries, or missing fingers in strict mode
- 501 Not Implemented: Driver doesn't support slap capture (BS2)
- 504 Gateway Timeout: Enrollment timeout
- 502 Bad Gateway: Driver error
Timing: ~40 seconds for 2 impressions (2 × 20s slap capture timeout)
---
### 10. Testing Plan
#### Unit Tests
**File:** `internal/driver/realscan/driver_test.go`
Test cases:
- ✅ Successful 2-impression enrollment
- ✅ Quality validation triggers retry
- ✅ Max retries exhausted returns error
- ✅ Missing fingers in strict mode returns error
- ✅ Missing fingers in lenient mode succeeds with partial
- ✅ Timeout handling
- ✅ Context cancellation
**File:** `internal/api/slap_enroll_test.go`
Test cases:
- ✅ Valid request succeeds
- ✅ Missing required fields returns 400
- ✅ Invalid mode returns 400
- ✅ Device not found returns 404
- ✅ Quality validation failure returns 422
- ✅ BS2 driver returns 501
- ✅ Config defaults applied correctly
- ✅ Request parameters override defaults
#### Integration Tests
Manual testing scenarios:
1. **Happy path**: Right hand enrollment with good quality
2. **Quality retry**: Intentionally poor placement triggers retry
3. **Max retries**: Consistently poor quality exhausts retries
4. **Missing finger**: Place only 3 of 4 fingers in strict mode
5. **Partial enrollment**: Missing finger in lenient mode
6. **Timeout**: Don't place fingers, verify timeout
7. **Both hands**: Enroll left_four, then right_four, then two_thumbs
8. **BS2 device**: Verify ErrSlapNotSupported returned
---
## File Summary
### New Files
1. `internal/api/slap_enroll.go` - HTTP handler (~250 lines)
### Modified Files
1. `internal/config/config.go` - Add QualityConfig struct
2. `config.yaml` - Add quality section
3. `internal/driver/driver.go` - Add SlapEnroll interface method, types, errors
4. `internal/driver/realscan/driver.go` - Implement SlapEnroll (~300 lines)
5. `internal/driver/bs2/driver.go` - Return ErrSlapNotSupported (~5 lines)
6. `internal/driver/demo/driver.go` - Simulate slap enrollment (~50 lines)
7. `internal/api/router.go` - Register new route (~10 lines)
8. `README.md` - Add usage examples
9. `specs/001-biometric-bridge/contracts/http-api.md` - Add API contract
### Test Files (New)
1. `internal/api/slap_enroll_test.go` - HTTP handler tests
2. Add test cases to `internal/driver/realscan/driver_test.go`
---
## Implementation Order
1. **Config updates** (config.go, config.yaml) - Foundation
2. **Driver interface** (driver.go) - Define contracts
3. **BS2 stub** (bs2/driver.go) - Quick win
4. **Demo implementation** (demo/driver.go) - For testing without hardware
5. **RealScan implementation** (realscan/driver.go) - Core logic
6. **HTTP handler** (api/slap_enroll.go) - API layer
7. **Router registration** (api/router.go) - Wire it up
8. **Documentation** (README, specs) - User-facing
9. **Tests** (unit + integration) - Validation
---
## Open Questions / Considerations
1. **WebSocket events**: Should we emit templates via WebSocket (like current RealScan Enroll) OR return them in HTTP response? 
   - **Decision**: Return in HTTP response for simplicity, emit enrollment_retry events via WebSocket
2. **Template storage**: Where should the widget store the returned templates?
   - Out of scope for bridge - widget integration will handle this
3. **Backwards compatibility**: Does this change any existing behavior?
   - **No** - this is a new endpoint, existing `/api/enroll` unchanged
4. **Performance**: 60s timeout sufficient for 2 slaps + retries?
   - 2 slaps × 20s = 40s baseline
   - 3 retries × 20s = 60s worst case
   - **Recommendation**: 60s timeout adequate
5. **Quality threshold tuning**: Should we provide quality statistics/monitoring?
   - Future enhancement - log quality distributions for ops monitoring
---
## Success Criteria
- ✅ Can enroll 4 fingers in ~40 seconds (vs 80s sequential)
- ✅ Quality validation works with configurable thresholds
- ✅ Retry logic handles poor quality gracefully
- ✅ Strict/lenient modes handle missing fingers correctly
- ✅ BS2 returns clear "not supported" error
- ✅ Demo mode works for testing without hardware
- ✅ API documented and tested
- ✅ Templates returned via HTTP response (or WebSocket events)
- ✅ Zero breaking changes to existing functionality
