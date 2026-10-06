package status

import (
	"encoding/json"
	"net/http"
	"time"
)

type Snapshot struct {
	Service   string
	NodeID    string
	Region    string
	UDPAddr   string
	StartedAt string
}

type Handler struct {
	Snapshot Snapshot
}

func New(nodeID, region, udpAddr string, startedAt time.Time) Handler {
	return Handler{
		Snapshot: Snapshot{
			Service:   "boostlab-gateway",
			NodeID:    nodeID,
			Region:    region,
			UDPAddr:   udpAddr,
			StartedAt: startedAt.UTC().Format(time.RFC3339),
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
		_ = json.NewEncoder(w).Encode(h.Snapshot)
	})
	return mux
}
