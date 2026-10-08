package provision

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type StoredPeer struct {
	PublicKey            string `json:"public_key"`
	Address              string `json:"address"`
	LastRegisteredAtUnix int64  `json:"last_registered_at_unix,omitempty"`
}

type PeerStore struct {
	Path string
}

func (s PeerStore) Load() ([]StoredPeer, error) {
	path := strings.TrimSpace(s.Path)
	if path == "" {
		return nil, nil
	}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read peer store: %w", err)
	}

	var peers []StoredPeer
	if err := json.Unmarshal(raw, &peers); err != nil {
		return nil, fmt.Errorf("parse peer store: %w", err)
	}

	seenKeys := make(map[string]struct{}, len(peers))
	seenAddresses := make(map[string]struct{}, len(peers))
	out := make([]StoredPeer, 0, len(peers))
	for _, item := range peers {
		item.PublicKey = strings.TrimSpace(item.PublicKey)
		item.Address = strings.TrimSpace(item.Address)
		if item.PublicKey == "" || item.Address == "" {
			continue
		}
		if _, exists := seenKeys[item.PublicKey]; exists {
			continue
		}
		if _, exists := seenAddresses[item.Address]; exists {
			continue
		}
		seenKeys[item.PublicKey] = struct{}{}
		seenAddresses[item.Address] = struct{}{}
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Address < out[j].Address
	})
	return out, nil
}

func (s PeerStore) Save(peers []StoredPeer) error {
	path := strings.TrimSpace(s.Path)
	if path == "" {
		return nil
	}

	clean := make([]StoredPeer, 0, len(peers))
	for _, item := range peers {
		item.PublicKey = strings.TrimSpace(item.PublicKey)
		item.Address = strings.TrimSpace(item.Address)
		if item.PublicKey == "" || item.Address == "" {
			continue
		}
		clean = append(clean, item)
	}
	sort.Slice(clean, func(i, j int) bool {
		return clean[i].Address < clean[j].Address
	})

	raw, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return fmt.Errorf("encode peer store: %w", err)
	}
	raw = append(raw, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create peer store directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write peer store: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace peer store: %w", err)
	}
	return nil
}
