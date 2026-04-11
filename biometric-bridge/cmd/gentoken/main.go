// Command gentoken generates a signed JWT for testing Biometric Bridge.
// Usage: go run ./cmd/gentoken -key test-private.pem
package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	keyFile := flag.String("key", "test-private.pem", "ECDSA P-256 private key (PEM)")
	issuer := flag.String("iss", "test", "Token issuer (must match config)")
	audience := flag.String("aud", "biometric-bridge", "Token audience (must match config)")
	ttl := flag.Duration("ttl", 1*time.Hour, "Token lifetime")
	flag.Parse()

	data, err := os.ReadFile(*keyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read key: %v\n", err)
		os.Exit(1)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		fmt.Fprintln(os.Stderr, "no PEM block found in key file")
		os.Exit(1)
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		k, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "parse key: %v (also tried PKCS8: %v)\n", err, err2)
			os.Exit(1)
		}
		var ok bool
		key, ok = k.(*ecdsa.PrivateKey)
		if !ok {
			fmt.Fprintln(os.Stderr, "key is not ECDSA")
			os.Exit(1)
		}
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    *issuer,
		Audience:  jwt.ClaimStrings{*audience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(*ttl)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(signed)
}
