package probe

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"
)

const Magic = "BOOSTLAB/PROBE/1"

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

	buf := make([]byte, 512)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		if string(buf[:n]) != Magic {
			continue
		}

		_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if _, err := conn.WriteTo([]byte(Magic), peer); err != nil && s.Log != nil {
			s.Log.Warn("probe reply failed", "peer", peer.String(), "error", err)
		}
	}
}
