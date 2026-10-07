package routequality

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Target struct {
	ID      string `json:"id"`
	Host    string `json:"host"`
	TCPPort int    `json:"tcp_port"`
}

func ParseTargets(raw string) ([]Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var targets []Target
	if err := json.Unmarshal([]byte(raw), &targets); err != nil {
		return nil, fmt.Errorf("parse BOOSTLAB_GAME_ROUTES_JSON: %w", err)
	}

	seen := make(map[string]struct{}, len(targets))
	out := make([]Target, 0, len(targets))
	for i, target := range targets {
		target.ID = strings.TrimSpace(target.ID)
		target.Host = strings.TrimSpace(target.Host)
		if target.ID == "" || len(target.ID) > 80 {
			return nil, fmt.Errorf("target %d: invalid id", i)
		}
		for _, r := range target.ID {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
				(r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
				return nil, fmt.Errorf("target %q: invalid id", target.ID)
			}
		}
		if target.Host == "" ||
			len(target.Host) > 253 ||
			strings.ContainsAny(target.Host, " /\\") {
			return nil, fmt.Errorf("target %q: invalid host", target.ID)
		}
		if target.TCPPort < 1 || target.TCPPort > 65535 {
			return nil, fmt.Errorf("target %q: invalid TCP port", target.ID)
		}
		key := strings.ToLower(target.ID)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate target %q", target.ID)
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	return out, nil
}

func (t Target) Address(ip net.IP) string {
	return net.JoinHostPort(ip.String(), strconv.Itoa(t.TCPPort))
}
