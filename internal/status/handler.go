package status

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Snapshot struct {
	Service             string   `json:"service"`
	NodeID              string   `json:"node_id"`
	Region              string   `json:"region"`
	UDPAddr             string   `json:"udp_addr"`
	WireGuardPublicKey  string   `json:"wireguard_public_key,omitempty"`
	WireGuardPort       int      `json:"wireguard_port"`
	StartedAt           string   `json:"started_at"`
	UptimeSeconds       int64    `json:"uptime_seconds"`
	CPUCount            int      `json:"cpu_count"`
	Goroutines          int      `json:"goroutines"`
	HeapAllocMB         float64  `json:"heap_alloc_mb"`
	Load1               *float64 `json:"load1,omitempty"`
	Load5               *float64 `json:"load5,omitempty"`
	Load15              *float64 `json:"load15,omitempty"`
}

type Handler struct {
	Service            string
	NodeID             string
	Region             string
	UDPAddr            string
	WireGuardPublicKey string
	WireGuardPort      int
	StartedAt          time.Time
	LoadAvgPath        string
}

func New(
	nodeID string,
	region string,
	udpAddr string,
	wireGuardPublicKey string,
	wireGuardPort int,
	startedAt time.Time,
) Handler {
	return Handler{
		Service:            "boostlab-gateway",
		NodeID:             nodeID,
		Region:             region,
		UDPAddr:            udpAddr,
		WireGuardPublicKey: wireGuardPublicKey,
		WireGuardPort:      wireGuardPort,
		StartedAt:          startedAt,
		LoadAvgPath:        "/proc/loadavg",
	}
}

func (h Handler) snapshot(now time.Time) Snapshot {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	load1, load5, load15 := readLoadAverage(h.LoadAvgPath)

	uptime := int64(now.Sub(h.StartedAt).Seconds())
	if uptime < 0 {
		uptime = 0
	}

	return Snapshot{
		Service:            h.Service,
		NodeID:             h.NodeID,
		Region:             h.Region,
		UDPAddr:            h.UDPAddr,
		WireGuardPublicKey: h.WireGuardPublicKey,
		WireGuardPort:      h.WireGuardPort,
		StartedAt:          h.StartedAt.UTC().Format(time.RFC3339),
		UptimeSeconds:      uptime,
		CPUCount:           runtime.NumCPU(),
		Goroutines:         runtime.NumGoroutine(),
		HeapAllocMB:        float64(memory.HeapAlloc) / (1024.0 * 1024.0),
		Load1:              load1,
		Load5:              load5,
		Load15:             load15,
	}
}

func (h Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /v1/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(h.snapshot(time.Now()))
	})
	return mux
}

func readLoadAverage(path string) (*float64, *float64, *float64) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 3 {
		return nil, nil, nil
	}

	values := make([]*float64, 3)
	for index := 0; index < 3; index++ {
		value, err := strconv.ParseFloat(fields[index], 64)
		if err != nil {
			return nil, nil, nil
		}
		copyValue := value
		values[index] = &copyValue
	}
	return values[0], values[1], values[2]
}
