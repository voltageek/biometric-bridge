package auth

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

//go:embed testkeys/test_ec256.pub
var testPublicKeyPEM []byte

//go:embed testkeys/test_ec256.priv
var testPrivateKeyPEM []byte

// TestPublicKeyPEM returns the embedded ECDSA P-256 public key PEM bytes.
func TestPublicKeyPEM() []byte {
	return testPublicKeyPEM
}

// TestPrivateKeyPEM returns the embedded ECDSA P-256 private key PEM bytes.
// This key MUST only be used for demo JWT generation — never in production.
func TestPrivateKeyPEM() []byte {
	return testPrivateKeyPEM
}

// DemoJWT generates a signed ES256 JWT using the embedded test key pair.
// The token includes the provided issuer, audience, subject, and name claims,
// and expires after ttl.
func DemoJWT(issuer, audience, subject, name string, ttl time.Duration) (string, error) {
	privKey, err := LoadPrivateKeyBytes(testPrivateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("load demo private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	if name != "" {
		claims["name"] = name
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := token.SignedString(privKey)
	if err != nil {
		return "", fmt.Errorf("sign demo JWT: %w", err)
	}

	return signed, nil
}
