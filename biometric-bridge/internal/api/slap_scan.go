package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
)

// slapScanRequest is the JSON body for POST /api/slap-scan.
type slapScanRequest struct {
	DeviceID string `json:"deviceId"`
	Mode     string `json:"mode"` // Required: "left_four", "right_four", or "two_thumbs"
}

// slapScanFingerResult is a single segmented finger in the response.
type slapScanFingerResult struct {
	Finger  string `json:"finger"`  // e.g., "right_index" — may be empty if SDK couldn't identify
	Image   string `json:"image"`   // Base64-encoded raw 8-bit grayscale
	Width   int    `json:"width"`   // Image width in pixels
	Height  int    `json:"height"`  // Image height in pixels
	Quality int    `json:"quality"` // NIST quality score (0-100)
}

// NewSlapScanHandler returns a handler for POST /api/slap-scan.
// Captures multiple fingers simultaneously and returns individually segmented
// finger images with quality scores.
func NewSlapScanHandler(d driver.Driver, reg *device.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req slapScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request: malformed JSON")
			return
		}
		if req.DeviceID == "" {
			writeError(w, http.StatusBadRequest, "invalid request: deviceId is required")
			return
		}
		if req.Mode == "" {
			writeError(w, http.StatusBadRequest, "invalid request: mode is required (left_four, right_four, two_thumbs)")
			return
		}

		// Validate capture mode
		mode := driver.CaptureMode(req.Mode)
		if !driver.ValidCaptureModes[mode] {
			writeError(w, http.StatusBadRequest, "invalid request: unrecognized mode; use left_four, right_four, or two_thumbs")
			return
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

		// Call driver with 20s timeout (slap captures take longer)
		slog.Debug("slap scan started", "device", req.DeviceID, "mode", req.Mode)
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		result, err := d.SlapScan(ctx, req.DeviceID, mode)
		if err != nil {
			slog.Warn("slap scan failed", "device", req.DeviceID, "mode", req.Mode, "error", err)
			if errors.Is(err, driver.ErrSlapNotSupported) {
				writeError(w, http.StatusNotImplemented, "slap capture not supported by this driver")
				return
			}
			if ctx.Err() == context.DeadlineExceeded {
				writeError(w, http.StatusGatewayTimeout, "slap scan timeout")
				return
			}
			writeError(w, http.StatusBadGateway, "driver error: "+err.Error())
			return
		}

		// Build response
		fingerResults := make([]slapScanFingerResult, 0, len(result.Fingers))
		for _, f := range result.Fingers {
			fingerResults = append(fingerResults, slapScanFingerResult{
				Finger:  string(f.Finger),
				Image:   base64.StdEncoding.EncodeToString(f.Template),
				Width:   f.Width,
				Height:  f.Height,
				Quality: f.Quality,
			})
		}

		slog.Info("slap scan completed",
			"device", req.DeviceID,
			"mode", req.Mode,
			"fingers", len(fingerResults),
		)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"mode":       req.Mode,
			"slapImage":  base64.StdEncoding.EncodeToString(result.SlapImage),
			"slapWidth":  result.SlapWidth,
			"slapHeight": result.SlapHeight,
			"fingers":    fingerResults,
		})
	}
}
