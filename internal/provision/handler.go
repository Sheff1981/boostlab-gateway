package provision

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	NodeID string
	Secret []byte
	Peers  *PeerManager
	Replay *TicketReplayGuard
}

func (h Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/peers/register", func(w http.ResponseWriter, r *http.Request) {
		if len(h.Secret) < 32 {
			http.Error(w, "peer provisioning is not configured", http.StatusServiceUnavailable)
			return
		}

		const prefix = "Bearer "
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(auth, prefix) {
			http.Error(w, "provisioning ticket required", http.StatusUnauthorized)
			return
		}

		claims, err := VerifyTicket(
			strings.TrimSpace(strings.TrimPrefix(auth, prefix)),
			h.Secret,
			h.NodeID,
			time.Now(),
		)
		if err != nil {
			http.Error(w, "invalid provisioning ticket", http.StatusUnauthorized)
			return
		}

		if h.Replay == nil || !h.Replay.Use(claims.Nonce, claims.ExpiresAtUnix, time.Now()) {
			http.Error(w, "provisioning ticket already used", http.StatusUnauthorized)
			return
		}

		if h.Peers == nil {
			http.Error(w, "peer manager unavailable", http.StatusServiceUnavailable)
			return
		}
		address, err := h.Peers.Register(r.Context(), claims.WireGuardKey)
		if err != nil {
			http.Error(w, "failed to register WireGuard peer", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			DeviceID      string `json:"device_id"`
			TunnelAddress string `json:"tunnel_address"`
		}{
			DeviceID:      claims.DeviceID,
			TunnelAddress: address,
		})
	})

	return mux
}
