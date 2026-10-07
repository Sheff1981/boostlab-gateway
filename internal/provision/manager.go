package provision

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/Sheff1981/boostlab-gateway/internal/peer"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type PeerManager struct {
	Interface  string
	TunnelCIDR string
	Persist    bool
	Runner     CommandRunner
}

func (m PeerManager) Register(ctx context.Context, publicKey string) (string, error) {
	if m.Runner == nil {
		m.Runner = ExecRunner{}
	}
	iface := strings.TrimSpace(m.Interface)
	if iface == "" {
		return "", fmt.Errorf("WireGuard interface is required")
	}

	baseIP, network, err := net.ParseCIDR(strings.TrimSpace(m.TunnelCIDR))
	if err != nil {
		return "", fmt.Errorf("invalid tunnel CIDR: %w", err)
	}
	base := baseIP.To4()
	ones, bits := network.Mask.Size()
	if base == nil || bits != 32 || ones != 24 {
		return "", fmt.Errorf("automatic provisioning currently requires an IPv4 /24 tunnel")
	}

	output, err := m.Runner.Run(ctx, "wg", "show", iface, "allowed-ips")
	if err != nil {
		return "", fmt.Errorf("read WireGuard peers: %w", err)
	}

	existingByKey, used := parseAllowedIPs(string(output))
	publicKey = strings.TrimSpace(publicKey)
	if existing := existingByKey[publicKey]; existing != "" {
		if m.Persist {
			if _, err := m.Runner.Run(ctx, "wg-quick", "save", iface); err != nil {
				return "", fmt.Errorf("peer exists but persistence failed: %w", err)
			}
		}
		return existing, nil
	}

	var address string
	for host := 2; host <= 254; host++ {
		candidateIP := net.IPv4(base[0], base[1], base[2], byte(host)).String()
		candidate := candidateIP + "/32"
		if _, exists := used[candidateIP]; !exists {
			address = candidate
			break
		}
	}
	if address == "" {
		return "", fmt.Errorf("WireGuard address pool exhausted")
	}

	if err := peer.Validate(peer.Spec{
		Interface: iface,
		PublicKey: publicKey,
		Address:   address,
	}); err != nil {
		return "", err
	}

	if output, err := m.Runner.Run(
		ctx,
		"wg", "set", iface,
		"peer", publicKey,
		"allowed-ips", address,
	); err != nil {
		return "", fmt.Errorf("apply WireGuard peer: %w: %s", err, strings.TrimSpace(string(output)))
	}

	if m.Persist {
		if output, err := m.Runner.Run(ctx, "wg-quick", "save", iface); err != nil {
			return "", fmt.Errorf("peer applied but persistence failed: %w: %s", err, strings.TrimSpace(string(output)))
		}
	}

	return address, nil
}

func parseAllowedIPs(raw string) (map[string]string, map[string]struct{}) {
	byKey := make(map[string]string)
	used := make(map[string]struct{})
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		for _, cidr := range strings.Split(fields[1], ",") {
			cidr = strings.TrimSpace(cidr)
			ip, _, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			used[ip.String()] = struct{}{}
			if key != "" && byKey[key] == "" {
				byKey[key] = cidr
			}
		}
	}
	return byKey, used
}
