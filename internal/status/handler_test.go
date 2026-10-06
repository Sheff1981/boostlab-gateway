package status

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	h := New("node-1", "test", ":51821", "", 51820, time.Unix(0, 0)).Routes()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}

func TestStatusContainsWireGuardMetadata(t *testing.T) {
	h := New(
		"node-1",
		"test",
		":51821",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		51820,
		time.Unix(0, 0),
	).Routes()

	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload Snapshot
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}

	if payload.WireGuardPort != 51820 {
		t.Fatalf("expected wireguard port 51820, got %d", payload.WireGuardPort)
	}
	if payload.WireGuardPublicKey == "" {
		t.Fatal("expected WireGuard public key")
	}
}
