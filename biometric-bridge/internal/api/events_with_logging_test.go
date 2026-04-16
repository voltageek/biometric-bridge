package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"biometric-bridge/internal/auth"
	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/driver/demo"
	"biometric-bridge/internal/events"
	"net/http/httptest"
)

// TestEventsWebSocketWithLogging ensures that the /events WebSocket can be
// upgraded even when the requestLogging middleware is enabled (wrapper
// implements http.Hijacker).
func TestEventsWebSocketWithLogging(t *testing.T) {
	cfg := demo.DefaultDemoConfig()
	d, err := demo.New(cfg)
	if err != nil {
		t.Fatalf("demo.New failed: %v", err)
	}

	// Registry with demo device registered
	reg := device.NewRegistry()
	reg.Register(driver.DeviceInfo{
		Name:            cfg.Devices[0].Name,
		ID:              cfg.Devices[0].ID,
		Model:           cfg.Devices[0].Model,
		FirmwareVersion: cfg.Devices[0].FirmwareVersion,
		FingerSupported: cfg.Devices[0].FingerSupported,
	})

	// Broker from driver subscribe
	broker := events.NewBroker(d.Subscribe())
	broker.Start()
	defer broker.Stop()

	// Token validator
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("key gen failed: %v", err)
	}
	issuer := "test-issuer"
	audience := "test-audience"
	validator := auth.NewTokenValidator(&priv.PublicKey, issuer, audience, time.Minute)

	router := NewRouter(RouterDeps{TokenValidator: validator, Driver: d, Registry: reg, Broker: broker, Demo: true, RequestLogging: true})
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Sign JWT for query param
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("token sign failed: %v", err)
	}

	// Build ws URL
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/events?token=" + signed

	dialer := websocket.Dialer{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v (http status %v)", err, resp)
	}
	defer conn.Close()

	// Allow broker to register subscriber
	time.Sleep(50 * time.Millisecond)
	if broker.SubscriberCount() == 0 {
		t.Fatalf("expected at least 1 subscriber, got 0")
	}
}
