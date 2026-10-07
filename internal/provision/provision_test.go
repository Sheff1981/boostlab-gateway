package provision

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func makeTicket(t *testing.T, secret []byte, claims Claims) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyTicket(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	now := time.Unix(1_800_000_000, 0)
	ticket := makeTicket(t, secret, Claims{
		NodeID:        "fra-1",
		DeviceID:      "BLDEV-ABC",
		WireGuardKey:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		ExpiresAtUnix: now.Add(time.Minute).Unix(),
		Nonce:         "nonce",
	})

	claims, err := VerifyTicket(ticket, secret, "fra-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if claims.DeviceID != "BLDEV-ABC" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestVerifyTicketRejectsWrongGatewayAndSignature(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	now := time.Unix(1_800_000_000, 0)
	ticket := makeTicket(t, secret, Claims{
		NodeID:        "fra-1",
		DeviceID:      "BLDEV-ABC",
		WireGuardKey:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		ExpiresAtUnix: now.Add(time.Minute).Unix(),
		Nonce:         "nonce",
	})

	if _, err := VerifyTicket(ticket, secret, "ams-1", now); err == nil {
		t.Fatal("expected wrong gateway to fail")
	}
	if _, err := VerifyTicket(ticket+"x", secret, "fra-1", now); err == nil {
		t.Fatal("expected bad signature to fail")
	}
}

type fakeRunner struct {
	showOutput string
	calls      []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.calls = append(f.calls, call)
	if name == "wg" && len(args) >= 3 && args[0] == "show" {
		return []byte(f.showOutput), nil
	}
	if name == "wg" && len(args) >= 1 && args[0] == "set" {
		return nil, nil
	}
	if name == "wg-quick" && len(args) >= 1 && args[0] == "save" {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected command: %s", call)
}

func TestPeerManagerAllocatesNextFreeAddress(t *testing.T) {
	runner := &fakeRunner{
		showOutput: "old-key\t10.77.0.2/32\nother-key\t10.77.0.3/32\n",
	}
	manager := PeerManager{
		Interface:  "wg0",
		TunnelCIDR: "10.77.0.0/24",
		Persist:    true,
		Runner:     runner,
	}

	address, err := manager.Register(
		context.Background(),
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	)
	if err != nil {
		t.Fatal(err)
	}
	if address != "10.77.0.4/32" {
		t.Fatalf("expected 10.77.0.4/32, got %s", address)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("expected show, set, save; got %#v", runner.calls)
	}
}

func TestPeerManagerReusesExistingAddress(t *testing.T) {
	key := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	runner := &fakeRunner{
		showOutput: key + "\t10.77.0.22/32\n",
	}
	manager := PeerManager{
		Interface:  "wg0",
		TunnelCIDR: "10.77.0.0/24",
		Persist:    false,
		Runner:     runner,
	}

	address, err := manager.Register(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if address != "10.77.0.22/32" {
		t.Fatalf("unexpected address %s", address)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("existing peer should not be changed: %#v", runner.calls)
	}
}
