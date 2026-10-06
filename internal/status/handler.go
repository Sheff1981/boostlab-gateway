package status

import (
	"encoding/json"
	"net/http"
	"time"
)

type Snapshot struct {
	Service             string `json:"service"`
	NodeID              string `json:"node_id"`
	Region              string `json:"region"`
	UDPAddr             string `json:"udp_addr"`
	WireGuardPublicKey  string `json:"wireguard_public_key,omitempty"`
	WireGuardPort       int    `json:"wireguard_port"`
	StartedAt           string `json:"started_at"`
}

type Handler struct {
	Snapshot Snapshot
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
		Snapshot: Snapshot{
			Service:            "boostlab-gateway",
			NodeID:             nodeID,
			Region:             region,
			UDPAddr:            udpAddr,
			WireGuardPublicKey: wireGuardPublicKey,
			WireGuardPort:      wireGuardPort,
			StartedAt:          startedAt.UTC().Format(time.RFC3339),
		},
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
		_ = json.NewEncoder(w).Encode(h.Snapshot)
	})
	return mux
}
