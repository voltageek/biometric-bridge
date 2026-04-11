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

// scanRequest is the JSON body for POST /api/scan.
type scanRequest struct {
	DeviceID string `json:"deviceId"`
	Finger   string `json:"finger,omitempty"` // Optional: finger position for LED guidance (e.g., "right_index")
}

// NewScanHandler returns a handler for POST /api/scan (T013).
func NewScanHandler(d driver.Driver, reg *device.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req scanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request: malformed JSON")
			return
		}
		if req.DeviceID == "" {
			writeError(w, http.StatusBadRequest, "invalid request: deviceId is required")
			return
		}

		// Validate optional finger position
		finger := driver.FingerPosition(req.Finger)
		if req.Finger != "" && !driver.ValidFingerPositions[finger] {
			writeError(w, http.StatusBadRequest, "invalid request: unrecognized finger position")
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

		// Call driver with 10s timeout
		slog.Debug("scan started", "device", req.DeviceID)
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		result, err := d.Scan(ctx, req.DeviceID, finger)
		if err != nil {
			slog.Warn("scan failed", "device", req.DeviceID, "error", err)
			if ctx.Err() == context.DeadlineExceeded {
				writeError(w, http.StatusGatewayTimeout, "scan timeout")
				return
			}
			writeError(w, http.StatusBadGateway, "driver error: "+err.Error())
			return
		}

		slog.Info("scan completed", "device", req.DeviceID, "quality", result.Quality)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"template": base64.StdEncoding.EncodeToString(result.Template),
			"quality":  result.Quality,
		})
	}
}
