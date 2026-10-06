package probe

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"time"
)

const (
	MagicV1       = "BOOSTLAB/PROBE/1"
	MagicV2Prefix = "BOOSTLAB/PROBE/2/"
	maxProbeBytes = 96
)

type Server struct {
	Addr string
	Log  *slog.Logger
}

func (s Server) Run(ctx context.Context) error {
	conn, err := net.ListenPacket("udp", s.Addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, maxProbeBytes+1)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		if !validPayload(buf[:n]) {
			continue
		}

		_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if _, err := conn.WriteTo(buf[:n], peer); err != nil && s.Log != nil {
			s.Log.Warn("probe reply failed", "peer", peer.String(), "error", err)
		}
	}
}

func validPayload(payload []byte) bool {
	if len(payload) == 0 || len(payload) > maxProbeBytes {
		return false
	}

	text := string(payload)
	if text == MagicV1 {
		return true
	}

	if !strings.HasPrefix(text, MagicV2Prefix) {
		return false
	}

	rest := strings.TrimPrefix(text, MagicV2Prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return false
	}

	nonce := parts[0]
	sequence := parts[1]

	if len(nonce) < 8 || len(nonce) > 32 {
		return false
	}
	if len(sequence) < 1 || len(sequence) > 3 {
		return false
	}

	for _, r := range nonce {
		if !((r >= 'a' && r <= 'f') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	for _, r := range sequence {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
