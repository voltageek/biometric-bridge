// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims captures relevant JWT claims for display in the tray UI.
type Claims struct {
	Subject   string    // sub claim
	Name      string    // name claim
	Issuer    string    // iss claim
	Audience  string    // aud claim
	ExpiresAt time.Time // exp claim
	RawToken  string    // Full JWT string for COPY JWT feature
}

// JWTStore holds the most recent JWT information for the tray application.
type JWTStore struct {
	mu      sync.RWMutex
	claims  Claims
	hasData bool
}

// NewJWTStore creates a new JWT store.
func NewJWTStore() *JWTStore {
	return &JWTStore{}
}

// SetClaims stores the claims from an authenticated request.
func (s *JWTStore) SetClaims(claims Claims) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claims = claims
	s.hasData = true
}

// GetClaims returns the stored claims and a boolean indicating if any exist.
func (s *JWTStore) GetClaims() (Claims, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.claims, s.hasData
}

// GetToken returns the raw JWT token string for the COPY JWT feature.
func (s *JWTStore) GetToken() (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasData || s.claims.RawToken == "" {
		return "", false
	}
	return s.claims.RawToken, true
}

// IsValid checks if the stored token is still valid (not expired).
func (s *JWTStore) IsValid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasData {
		return false
	}
	return time.Now().Before(s.claims.ExpiresAt)
}

// Clear removes all stored claims.
func (s *JWTStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claims = Claims{}
	s.hasData = false
}

// ExtractClaimsFromJWT extracts relevant claims from a jwt.Claims interface.
func ExtractClaimsFromJWT(rawClaims jwt.Claims, rawToken string) Claims {
	claims := Claims{
		RawToken: rawToken,
	}

	if registered, ok := rawClaims.(jwt.MapClaims); ok {
		if sub, ok := registered["sub"].(string); ok {
			claims.Subject = sub
		}
		if name, ok := registered["name"].(string); ok {
			claims.Name = name
		}
		if iss, ok := registered["iss"].(string); ok {
			claims.Issuer = iss
		}
		if aud, ok := registered["aud"].(string); ok {
			claims.Audience = aud
		}
		if exp, ok := registered["exp"].(float64); ok {
			claims.ExpiresAt = time.Unix(int64(exp), 0)
		}
	}

	// If name is empty, use subject as fallback
	if claims.Name == "" {
		claims.Name = claims.Subject
	}

	return claims
}
