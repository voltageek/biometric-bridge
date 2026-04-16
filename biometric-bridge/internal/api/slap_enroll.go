package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/events"
)

// Defaults for slap enrollment
const (
	defaultSlapMinQuality = 60
	defaultSlapMaxRetries = 3
	defaultSlapTimeout    = 60 * time.Second
	perSlapTimeout        = 20 * time.Second
)

// Request/Response types
type slapEnrollRequest struct {
	DeviceID   string `json:"deviceId"`
	UserID     string `json:"userId"`
	UserName   string `json:"userName"`
	Mode       string `json:"mode"`
	MinQuality *int   `json:"minQuality,omitempty"`
	MaxRetries *int   `json:"maxRetries,omitempty"`
	Strict     *bool  `json:"strict,omitempty"`
}

type slapEnrollResult struct {
	OK           bool                   `json:"ok"`
	UserID       string                 `json:"userId"`
	Impressions  []slapEnrollImpression `json:"impressions"`
	TotalRetries int                    `json:"totalRetries"`
}

type slapEnrollImpression struct {
	ImpressionNumber int                  `json:"impressionNumber"`
	Mode             string               `json:"mode"`
	SlapImage        string               `json:"slapImage"`
	SlapWidth        int                  `json:"slapWidth"`
	SlapHeight       int                  `json:"slapHeight"`
	Fingers          []slapEnrolledFinger `json:"fingers"`
}

type slapEnrolledFinger struct {
	Finger   string `json:"finger"`
	Template string `json:"template"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Quality  int    `json:"quality"`
}

type qualityFailure struct {
	Finger    string `json:"finger"`
	Quality   int    `json:"quality"`
	Threshold int    `json:"threshold"`
}

type slapEnrollError struct {
	Error         string           `json:"error"`
	FailedFingers []qualityFailure `json:"failedFingers,omitempty"`
	Detected      int              `json:"detected,omitempty"`
	Expected      int              `json:"expected,omitempty"`
}

// Validation helpers
func validateSlapEnrollRequest(r *slapEnrollRequest) error {
	if r.DeviceID == "" {
		return errors.New("invalid request: deviceId is required")
	}
	if r.UserID == "" {
		return errors.New("invalid request: userId is required")
	}
	if r.UserName == "" {
		return errors.New("invalid request: userName is required")
	}
	if r.Mode == "" {
		return errors.New("invalid request: mode is required (left_four, right_four, two_thumbs)")
	}
	if !driver.ValidCaptureModes[driver.CaptureMode(r.Mode)] {
		return errors.New("invalid request: unrecognized mode; use left_four, right_four, or two_thumbs")
	}
	if r.MinQuality != nil {
		if *r.MinQuality < 0 || *r.MinQuality > 100 {
			return errors.New("invalid request: minQuality must be 0-100")
		}
	}
	if r.MaxRetries != nil {
		if *r.MaxRetries < 0 || *r.MaxRetries > 10 {
			return errors.New("invalid request: maxRetries must be 0-10")
		}
	}
	return nil
}

func expectedFingerCount(mode driver.CaptureMode) int {
	switch mode {
	case driver.CaptureLeftFour, driver.CaptureRightFour:
		return 4
	case driver.CaptureTwoThumbs:
		return 2
	default:
		return 0
	}
}

// convert driver.SlapScanResult to slapEnrollImpression
func convertSlapResult(impressionNum int, mode driver.CaptureMode, r *driver.SlapScanResult) slapEnrollImpression {
	imgBase64 := base64.StdEncoding.EncodeToString(r.SlapImage)
	imp := slapEnrollImpression{
		ImpressionNumber: impressionNum,
		Mode:             string(mode),
		SlapImage:        imgBase64,
		SlapWidth:        r.SlapWidth,
		SlapHeight:       r.SlapHeight,
		Fingers:          []slapEnrolledFinger{},
	}
	for _, f := range r.Fingers {
		imp.Fingers = append(imp.Fingers, slapEnrolledFinger{
			Finger:   string(f.Finger),
			Template: base64.StdEncoding.EncodeToString(f.Template),
			Width:    f.Width,
			Height:   f.Height,
			Quality:  f.Quality,
		})
	}
	return imp
}

// validateQuality checks each ScanResult and returns failures when quality < minQuality.
// If minQuality == 0 it bypasses validation and returns empty slice.
func validateQuality(fingers []driver.ScanResult, minQuality int) []qualityFailure {
	if minQuality <= 0 {
		return nil
	}
	var failures []qualityFailure
	for _, f := range fingers {
		if f.Quality < minQuality {
			failures = append(failures, qualityFailure{
				Finger:    string(f.Finger),
				Quality:   f.Quality,
				Threshold: minQuality,
			})
		}
	}
	return failures
}

// NewSlapEnrollHandler returns an http.HandlerFunc that performs slap enrollment.
// It takes driver, device registry, and events broker dependencies.
func NewSlapEnrollHandler(d driver.Driver, reg *device.Registry, broker interface{}) http.HandlerFunc {
	b, _ := broker.(*events.Broker)
	// broker parameter is accepted to match router deps; if provided it will be used for emitting events.
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		var body slapEnrollRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request: json decode failed"})
			return
		}

		if err := validateSlapEnrollRequest(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		mode := driver.CaptureMode(body.Mode)

		// Set defaults
		minQuality := defaultSlapMinQuality
		if body.MinQuality != nil {
			minQuality = *body.MinQuality
		}
		maxRetries := defaultSlapMaxRetries
		if body.MaxRetries != nil {
			maxRetries = *body.MaxRetries
		}
		strict := true
		if body.Strict != nil {
			strict = *body.Strict
		}
		// Use variables to avoid "declared and not used" during incremental implementation
		_ = minQuality
		_ = maxRetries
		_ = strict

		// Log enrollment start
		start := time.Now()
		slog.Info("slap enroll start",
			"device", body.DeviceID,
			"user", body.UserID,
			"mode", body.Mode,
			"minQuality", minQuality,
			"maxRetries", maxRetries,
			"strict", strict,
		)

		// Acquire device lock
		if err := reg.Acquire(body.DeviceID); err != nil {
			// DeviceError provides HTTP mapping
			if de, ok := err.(*device.DeviceError); ok {
				slog.Warn("device acquire failed", "device", body.DeviceID, "error", de.Message)
				writeJSON(w, de.Code, map[string]string{"error": de.Message})
				return
			}
			slog.Error("device registry error", "device", body.DeviceID, "err", err.Error())
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "device registry error"})
			return
		}
		defer reg.Release(body.DeviceID)

		// Overall timeout
		overallCtx, cancel := context.WithTimeout(ctx, defaultSlapTimeout)
		defer cancel()

		var impressions []slapEnrollImpression
		totalRetries := 0

		for i := 1; i <= 2; i++ {
			var chosen *driver.SlapScanResult
			var lastFailures []qualityFailure

			// Per-impression retry loop
			maxAttempts := maxRetries + 1
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				snapCtx, snapCancel := context.WithTimeout(overallCtx, perSlapTimeout)
				res, err := d.SlapScan(snapCtx, body.DeviceID, mode)
				snapCancel()

				if err != nil {
					if errors.Is(err, driver.ErrSlapNotSupported) {
						writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "slap enrollment not supported by this driver"})
						return
					}
					// Treat other errors as timeouts/service issues
					writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "slap enrollment timeout"})
					return
				}

				// Finger count check (strict)
				expected := expectedFingerCount(mode)
				detected := len(res.Fingers)
				if strict && expected > 0 && detected < expected {
					// Strict mode failure -> return 422
					errBody := slapEnrollError{
						Error:    fmt.Sprintf("enrollment failed: expected %d fingers, detected %d (strict mode)", expected, detected),
						Detected: detected,
						Expected: expected,
					}
					slog.Warn("strict finger count failure", "device", body.DeviceID, "expected", expected, "detected", detected)
					writeJSON(w, http.StatusUnprocessableEntity, errBody)
					return
				}

				// Quality validation (if minQuality > 0)
				failures := validateQuality(res.Fingers, minQuality)
				if len(failures) == 0 {
					chosen = res
					break
				}

				// Record last failures
				lastFailures = failures

				// If we have attempts remaining, emit retry event and continue
				if attempt < maxAttempts {
					totalRetries++
					// Emit enrollment_retry event if broker available
					if b != nil {
						// Prepare details
						details := map[string]interface{}{
							"impressionNum": i,
							"failedFingers": failures,
						}
						msgBytes, _ := json.Marshal(details)
						evt := driver.Event{
							Type:       "enrollment_retry",
							DeviceName: body.DeviceID,
							UserID:     body.UserID,
							Attempt:    attempt + 1, // 2 = first retry
							Message:    string(msgBytes),
						}
						slog.Info("enrollment retry", "device", body.DeviceID, "user", body.UserID, "attempt", attempt+1, "impression", i)
						b.Emit(evt)
					} else {
						slog.Info("enrollment retry (no broker)", "device", body.DeviceID, "user", body.UserID, "attempt", attempt+1, "impression", i)
					}
					// Continue to next attempt
					continue
				}

				// Attempts exhausted -> return 422 with failures
				errBody := slapEnrollError{
					Error:         fmt.Sprintf("enrollment failed: quality threshold not met after %d retries", maxRetries),
					FailedFingers: lastFailures,
				}
				writeJSON(w, http.StatusUnprocessableEntity, errBody)
				return
			}

			if chosen == nil {
				// Defensive: if no chosen result after loop, fail
				slog.Error("no chosen impression after retries", "device", body.DeviceID, "user", body.UserID)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal enrollment error"})
				return
			}

			imp := convertSlapResult(i, mode, chosen)
			impressions = append(impressions, imp)
		}

		res := slapEnrollResult{
			OK:           true,
			UserID:       body.UserID,
			Impressions:  impressions,
			TotalRetries: totalRetries,
		}
		slog.Info("slap enroll complete", "device", body.DeviceID, "user", body.UserID, "impressions", len(impressions), "totalRetries", totalRetries, "duration_ms", time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, res)
	}
}
