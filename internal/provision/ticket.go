package provision

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	NodeID        string `json:"node_id"`
	DeviceID      string `json:"device_id"`
	WireGuardKey  string `json:"wireguard_public_key"`
	ExpiresAtUnix int64  `json:"exp"`
	Nonce         string `json:"nonce"`
}

func VerifyTicket(raw string, secret []byte, expectedNode string, now time.Time) (Claims, error) {
	if len(secret) < 32 {
		return Claims{}, fmt.Errorf("provisioning secret is not configured")
	}
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return Claims{}, fmt.Errorf("invalid ticket format")
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)

	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, actual) {
		return Claims{}, fmt.Errorf("invalid ticket signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("invalid ticket payload")
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("invalid ticket claims")
	}

	if strings.TrimSpace(claims.NodeID) != strings.TrimSpace(expectedNode) {
		return Claims{}, fmt.Errorf("ticket is for another gateway")
	}
	if claims.ExpiresAtUnix <= now.Unix() {
		return Claims{}, fmt.Errorf("ticket expired")
	}
	if claims.ExpiresAtUnix > now.Add(5*time.Minute).Unix() {
		return Claims{}, fmt.Errorf("ticket expiry is too far in the future")
	}
	if strings.TrimSpace(claims.DeviceID) == "" ||
		strings.TrimSpace(claims.WireGuardKey) == "" ||
		strings.TrimSpace(claims.Nonce) == "" {
		return Claims{}, fmt.Errorf("ticket claims are incomplete")
	}

	return claims, nil
}
