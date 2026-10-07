package status

import (
	"encoding/json"
	"net/http"
	"os"
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


func TestSnapshotContainsRuntimeCapacity(t *testing.T) {
	started := time.Now().Add(-90 * time.Second)
	handler := New(
		"fra-1",
		"europe",
		":51821",
		"wg-key",
		51820,
		started,
	)

	snapshot := handler.snapshot(time.Now())
	if snapshot.NodeID != "fra-1" {
		t.Fatalf("unexpected node id: %q", snapshot.NodeID)
	}
	if snapshot.CPUCount < 1 {
		t.Fatalf("invalid cpu count: %d", snapshot.CPUCount)
	}
	if snapshot.Goroutines < 1 {
		t.Fatalf("invalid goroutine count: %d", snapshot.Goroutines)
	}
	if snapshot.UptimeSeconds < 80 {
		t.Fatalf("unexpected uptime: %d", snapshot.UptimeSeconds)
	}
	if snapshot.HeapAllocMB < 0 {
		t.Fatalf("unexpected heap: %f", snapshot.HeapAllocMB)
	}
}

func TestReadLoadAverage(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "loadavg")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if _, err := file.WriteString("0.25 0.50 0.75 1/100 123\n"); err != nil {
		t.Fatal(err)
	}

	load1, load5, load15 := readLoadAverage(file.Name())
	if load1 == nil || load5 == nil || load15 == nil {
		t.Fatal("expected load average values")
	}
	if *load1 != 0.25 || *load5 != 0.50 || *load15 != 0.75 {
		t.Fatalf("unexpected loads: %v %v %v", *load1, *load5, *load15)
	}
}

func TestReadLoadAverageMissingFileIsSafe(t *testing.T) {
	load1, load5, load15 := readLoadAverage("/definitely/missing/loadavg")
	if load1 != nil || load5 != nil || load15 != nil {
		t.Fatal("expected no load average on missing file")
	}
}


func TestReadyRejectsConfiguredBrokenDataPlane(t *testing.T) {
	handler := New(
		"node-1",
		"test",
		":51821",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		51820,
		time.Now(),
	)
	handler.WireGuardInterface = "definitely-missing-wg-interface"
	handler.IPv4ForwardingPath = "/definitely/missing/ip_forward"

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", res.Code)
	}
}

func TestReadyAcceptsConfiguredWorkingDataPlane(t *testing.T) {
	forwardingFile := t.TempDir() + "/ip_forward"
	if err := os.WriteFile(forwardingFile, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler := New(
		"node-1",
		"test",
		":51821",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		51820,
		time.Now(),
	)
	handler.WireGuardInterface = "lo"
	handler.IPv4ForwardingPath = forwardingFile

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}
