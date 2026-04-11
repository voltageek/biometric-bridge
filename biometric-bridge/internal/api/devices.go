package api

import (
	"net/http"

	"biometric-bridge/internal/driver"
)

// NewDevicesHandler returns a handler for GET /api/devices (T016).
func NewDevicesHandler(d driver.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices := d.ListDevices()

		type deviceJSON struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Model           string `json:"model"`
			FirmwareVersion string `json:"firmwareVersion"`
			FingerSupported bool   `json:"fingerSupported"`
		}

		out := make([]deviceJSON, len(devices))
		for i, d := range devices {
			out[i] = deviceJSON{
				ID:              d.Name, // API uses human-readable name as ID (Principle VI)
				Name:            d.Name,
				Model:           d.Model,
				FirmwareVersion: d.FirmwareVersion,
				FingerSupported: d.FingerSupported,
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"devices": out})
	}
}
