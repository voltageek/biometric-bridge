package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
)

// enrollRequest is the JSON body for POST /api/enroll.
type enrollRequest struct {
	DeviceID string   `json:"deviceId"`
	UserID   string   `json:"userId"`
	UserName string   `json:"userName"`
	Fingers  []string `json:"fingers,omitempty"` // Optional: finger position per impression for LED guidance
}

// NewEnrollHandler returns a handler for POST /api/enroll (T014).
func NewEnrollHandler(d driver.Driver, reg *device.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req enrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request: malformed JSON")
			return
		}
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

		// Validate optional finger positions
		var fingers []driver.FingerPosition
		for i, f := range req.Fingers {
			fp := driver.FingerPosition(f)
			if f != "" && !driver.ValidFingerPositions[fp] {
				writeError(w, http.StatusBadRequest,
					fmt.Sprintf("invalid request: unrecognized finger position at index %d", i))
				return
			}
			fingers = append(fingers, fp)
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

		// Call driver with 10s timeout per impression
		slog.Debug("enroll started", "device", req.DeviceID, "userId", req.UserID)
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := d.Enroll(ctx, req.DeviceID, req.UserID, req.UserName, fingers); err != nil {
			slog.Warn("enroll failed", "device", req.DeviceID, "userId", req.UserID, "error", err)
			if ctx.Err() == context.DeadlineExceeded {
				writeError(w, http.StatusGatewayTimeout, "enroll timeout")
				return
			}
			writeError(w, http.StatusBadGateway, "driver error: "+err.Error())
			return
		}

		slog.Info("enroll completed", "device", req.DeviceID, "userId", req.UserID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":     true,
			"userId": req.UserID,
		})
	}
}
