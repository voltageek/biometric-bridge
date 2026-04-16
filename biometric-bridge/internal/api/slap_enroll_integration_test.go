package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"biometric-bridge/internal/auth"
	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/driver/demo"
	"biometric-bridge/internal/events"
)

// TestSlapEnroll_SuccessAfterRetry verifies the slap-enroll flow succeeds when
// the demo driver returns low quality on the first attempt and improved
// quality on the second attempt.
func TestSlapEnroll_SuccessAfterRetry(t *testing.T) {
	// Configure demo to return InitialQuality low on first attempt then ImprovedQuality
	cfg := demo.DefaultDemoConfig()
	cfg.AttemptsToImprove = 1
	cfg.InitialQuality = 40
	cfg.ImprovedQuality = 90

	d, err := demo.New(cfg)
	if err != nil {
		t.Fatalf("failed to create demo driver: %v", err)
	}

	// Set up registry and register the demo device
	reg := device.NewRegistry()
	// Register expects driver.DeviceInfo
	reg.Register(driver.DeviceInfo{
		Name:            cfg.Devices[0].Name,
		ID:              cfg.Devices[0].ID,
		Model:           cfg.Devices[0].Model,
		FirmwareVersion: cfg.Devices[0].FirmwareVersion,
		FingerSupported: cfg.Devices[0].FingerSupported,
	})

	// Set up broker
	// Create broker from driver's Subscribe channel
	broker := events.NewBroker(d.Subscribe())
	broker.Start()
	defer broker.Stop()

	// Create a TokenValidator backed by a generated ECDSA key so we can
	// produce a valid JWT for the test request.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	issuer := "test-issuer"
	audience := "test-audience"
	validator := auth.NewTokenValidator(&priv.PublicKey, issuer, audience, time.Minute)

	router := NewRouter(RouterDeps{TokenValidator: validator, Driver: d, Registry: reg, Broker: broker, Demo: true})

	// Prepare request: minQuality set to 60 so first attempt fails and second succeeds
	reqBody := map[string]any{
		"deviceId":   cfg.Devices[0].Name,
		"userId":     "test-user-1",
		"userName":   "Test User",
		"mode":       "left_four",
		"minQuality": 60,
		"maxRetries": 2,
	}
	b, _ := json.Marshal(reqBody)

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create a signed JWT
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/slap-enroll", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+signed)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// We don't need to decode full body here; the 200 status indicates success
}
