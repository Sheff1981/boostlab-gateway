package provision

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

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
	DataFile   string
	Runner     CommandRunner

	mu sync.Mutex
}

func (m *PeerManager) Register(ctx context.Context, publicKey string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runner := m.Runner
	if runner == nil {
		runner = ExecRunner{}
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

	output, err := runner.Run(ctx, "wg", "show", iface, "allowed-ips")
	if err != nil {
		return "", fmt.Errorf("read WireGuard peers: %w", err)
	}

	existingByKey, used := parseAllowedIPs(string(output))
	publicKey = strings.TrimSpace(publicKey)
	now := time.Now().UTC()
	if existing := existingByKey[publicKey]; existing != "" {
		if m.Persist {
			_ = m.persistCurrent(existingByKey, publicKey, now)
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

	if commandOutput, err := runner.Run(
		ctx,
		"wg", "set", iface,
		"peer", publicKey,
		"allowed-ips", address,
	); err != nil {
		return "", fmt.Errorf(
			"apply WireGuard peer: %w: %s",
			err,
			strings.TrimSpace(string(commandOutput)),
		)
	}

	if m.Persist {
		existingByKey[publicKey] = address
		if err := m.persistCurrent(existingByKey, publicKey, now); err != nil {
			return "", fmt.Errorf("peer applied but registry persistence failed: %w", err)
		}
	}

	return address, nil
}

func (m *PeerManager) Restore(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.Persist {
		return 0, nil
	}
	runner := m.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	store := PeerStore{Path: m.DataFile}
	peers, err := store.Load()
	if err != nil {
		return 0, err
	}

	restored := 0
	for _, item := range peers {
		if err := peer.Validate(peer.Spec{
			Interface: m.Interface,
			PublicKey: item.PublicKey,
			Address:   item.Address,
		}); err != nil {
			return restored, fmt.Errorf("invalid stored peer: %w", err)
		}
		if commandOutput, err := runner.Run(
			ctx,
			"wg", "set", m.Interface,
			"peer", item.PublicKey,
			"allowed-ips", item.Address,
		); err != nil {
			return restored, fmt.Errorf(
				"restore WireGuard peer: %w: %s",
				err,
				strings.TrimSpace(string(commandOutput)),
			)
		}
		restored++
	}
	return restored, nil
}

func (m *PeerManager) persistCurrent(
	existingByKey map[string]string,
	touchedKey string,
	now time.Time,
) error {
	store := PeerStore{Path: m.DataFile}
	stored, err := store.Load()
	if err != nil {
		return err
	}

	registeredAt := make(map[string]int64, len(stored))
	for _, item := range stored {
		registeredAt[item.PublicKey] = item.LastRegisteredAtUnix
	}
	if strings.TrimSpace(touchedKey) != "" {
		registeredAt[touchedKey] = now.Unix()
	}

	peers := make([]StoredPeer, 0, len(existingByKey))
	for publicKey, address := range existingByKey {
		if strings.TrimSpace(publicKey) == "" || strings.TrimSpace(address) == "" {
			continue
		}
		peers = append(peers, StoredPeer{
			PublicKey:            publicKey,
			Address:              address,
			LastRegisteredAtUnix: registeredAt[publicKey],
		})
	}
	return store.Save(peers)
}

func (m *PeerManager) PruneStale(
	ctx context.Context,
	maxAge time.Duration,
	now time.Time,
) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.Persist || maxAge <= 0 {
		return 0, nil
	}
	runner := m.Runner
	if runner == nil {
		runner = ExecRunner{}
	}

	store := PeerStore{Path: m.DataFile}
	stored, err := store.Load()
	if err != nil {
		return 0, err
	}
	if len(stored) == 0 {
		return 0, nil
	}

	output, err := runner.Run(ctx, "wg", "show", m.Interface, "latest-handshakes")
	if err != nil {
		return 0, fmt.Errorf("read WireGuard handshakes: %w", err)
	}
	handshakes := parseLatestHandshakes(string(output))

	kept := make([]StoredPeer, 0, len(stored))
	removed := 0
	for _, item := range stored {
		referenceUnix := handshakes[item.PublicKey]
		if referenceUnix <= 0 {
			referenceUnix = item.LastRegisteredAtUnix
		}
		if referenceUnix <= 0 || now.Sub(time.Unix(referenceUnix, 0)) < maxAge {
			kept = append(kept, item)
			continue
		}

		commandOutput, err := runner.Run(
			ctx,
			"wg", "set", m.Interface,
			"peer", item.PublicKey,
			"remove",
		)
		if err != nil {
			return removed, fmt.Errorf(
				"remove stale WireGuard peer: %w: %s",
				err,
				strings.TrimSpace(string(commandOutput)),
			)
		}
		removed++
	}

	if removed > 0 {
		if err := store.Save(kept); err != nil {
			return removed, fmt.Errorf("persist pruned peer registry: %w", err)
		}
	}
	return removed, nil
}

func parseLatestHandshakes(raw string) map[string]int64 {
	out := make(map[string]int64)
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		out[strings.TrimSpace(fields[0])] = value
	}
	return out
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
