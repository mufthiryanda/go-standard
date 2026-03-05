package snapbi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"go-standard/internal/apperror"
)

// SignSymmetric creates the HMAC-SHA512 signature for SNAP BI transaction endpoints.
//
// Flow:
//  1. SHA256(accessToken) → lowercase hex
//  2. SHA256(minifiedBody) → lowercase hex
//  3. stringToSign = method + ":" + path + ":" + hashedToken + ":" + hashedBody + ":" + timestamp
//  4. HMAC-SHA512(stringToSign, rawAccessToken) → base64
func SignSymmetric(accessToken, method, path string, body []byte, timestamp string) (string, error) {
	minified, err := minifyJSON(body)
	if err != nil {
		return "", apperror.Internal("snapbi: minify request body failed", err)
	}

	// Step 1 — hash access token
	tokenHash := sha256.Sum256([]byte(accessToken))
	hashedToken := strings.ToLower(hex.EncodeToString(tokenHash[:]))

	// Step 2 — hash body
	bodyHash := sha256.Sum256(minified)
	hashedBody := strings.ToLower(hex.EncodeToString(bodyHash[:]))

	// Step 3 — compose string to sign
	stringToSign := fmt.Sprintf("%s:%s:%s:%s:%s",
		strings.ToUpper(method), path, hashedToken, hashedBody, timestamp,
	)

	// Step 4 — HMAC-SHA512
	mac := hmac.New(sha512.New, []byte(accessToken))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// minifyJSON compacts JSON bytes, removing insignificant whitespace.
func minifyJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
