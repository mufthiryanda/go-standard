package snapbi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	"go-standard/internal/apperror"
)

// SignAsymmetric creates an RSA-SHA256 PKCS1v15 signature for the SNAP BI access token flow.
// The string to sign is: clientKey + "|" + timestamp
func SignAsymmetric(privateKey *rsa.PrivateKey, clientKey, timestamp string) (string, error) {
	message := clientKey + "|" + timestamp
	hash := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", apperror.Internal("snapbi: asymmetric signature failed", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// loadPrivateKey reads a PEM-encoded RSA private key from path.
// Supports PKCS8 and PKCS1 formats.
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("snapbi: read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("snapbi: invalid PEM block in %s", path)
	}

	// Try PKCS8 first
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("snapbi: private key is not RSA")
		}
		return rsaKey, nil
	}

	// Fallback to PKCS1
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("snapbi: parse private key: %w", err)
	}
	return rsaKey, nil
}
