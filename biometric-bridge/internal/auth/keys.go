// Package auth provides ECDSA public key loading, JWT validation, and HTTP/WS
// authentication middleware for the Biometric Bridge.
package auth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPublicKey reads an ECDSA P-256 public key from a PEM file.
// It returns a clear error if the file is missing, unreadable, or not a valid
// ECDSA public key (FR-017).
func LoadPublicKey(path string) (*ecdsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("public key file %q: no PEM block found", path)
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA (got %T)", pub)
	}

	return ecdsaPub, nil
}

// LoadPublicKeyBytes reads an ECDSA P-256 public key from PEM-encoded bytes.
func LoadPublicKeyBytes(data []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("public key bytes: no PEM block found")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA (got %T)", pub)
	}

	return ecdsaPub, nil
}

// LoadPrivateKeyBytes reads an ECDSA P-256 private key from PEM-encoded bytes.
// Supports both SEC1 (EC) and PKCS8 formats.
func LoadPrivateKeyBytes(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("private key bytes: no PEM block found")
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	k, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err2 != nil {
		return nil, fmt.Errorf("parse private key (EC: %v, PKCS8: %v)", err, err2)
	}

	ecdsaKey, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA (got %T)", k)
	}

	return ecdsaKey, nil
}
