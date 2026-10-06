package peer

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
)

type Spec struct {
	Interface string
	PublicKey string
	Address   string
}

func Validate(spec Spec) error {
	spec.Interface = strings.TrimSpace(spec.Interface)
	spec.PublicKey = strings.TrimSpace(spec.PublicKey)
	spec.Address = strings.TrimSpace(spec.Address)

	if spec.Interface == "" {
		return fmt.Errorf("interface is required")
	}
	if len(spec.Interface) > 15 {
		return fmt.Errorf("interface name is too long")
	}

	key, err := base64.StdEncoding.DecodeString(spec.PublicKey)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("invalid WireGuard public key")
	}

	ip, network, err := net.ParseCIDR(spec.Address)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}
	if ip.To4() != nil {
		ones, bits := network.Mask.Size()
		if bits != 32 || ones != 32 {
			return fmt.Errorf("IPv4 peer address must use /32")
		}
	} else {
		ones, bits := network.Mask.Size()
		if bits != 128 || ones != 128 {
			return fmt.Errorf("IPv6 peer address must use /128")
		}
	}

	return nil
}
