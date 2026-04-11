package auth

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenValidator validates ES256 JWTs against a pre-shared ECDSA public key,
// enforcing issuer, audience, and expiry with configurable clock skew.
type TokenValidator struct {
	publicKey *ecdsa.PublicKey
	issuer    string
	audience  string
	clockSkew time.Duration
}

// NewTokenValidator creates a validator with the given parameters.
func NewTokenValidator(pub *ecdsa.PublicKey, issuer, audience string, clockSkew time.Duration) *TokenValidator {
	return &TokenValidator{
		publicKey: pub,
		issuer:    issuer,
		audience:  audience,
		clockSkew: clockSkew,
	}
}

// Claims returns the standard JWT claims after successful validation.
type Claims = jwt.RegisteredClaims

// Validate parses and validates a JWT string. It checks the ES256 signature,
// issuer, audience, and expiry. Returns the parsed claims or a structured error
// describing the failure mode.
func (v *TokenValidator) Validate(tokenStr string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"ES256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithLeeway(v.clockSkew),
	)

	token, err := parser.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
