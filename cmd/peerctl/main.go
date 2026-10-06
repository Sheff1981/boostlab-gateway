package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/Sheff1981/boostlab-gateway/internal/peer"
)

func main() {
	var (
		iface     = flag.String("interface", "wg0", "WireGuard interface")
		publicKey = flag.String("public-key", "", "client WireGuard public key")
		address   = flag.String("address", "", "client tunnel address, e.g. 10.77.0.2/32")
		apply     = flag.Bool("apply", false, "apply the peer to the live interface")
	)
	flag.Parse()

	spec := peer.Spec{
		Interface: strings.TrimSpace(*iface),
		PublicKey: strings.TrimSpace(*publicKey),
		Address:   strings.TrimSpace(*address),
	}
	if err := peer.Validate(spec); err != nil {
		log.Fatal(err)
	}

	args := []string{
		"set",
		spec.Interface,
		"peer",
		spec.PublicKey,
		"allowed-ips",
		spec.Address,
	}

	if !*apply {
		fmt.Printf("dry-run: wg %s\n", strings.Join(args, " "))
		fmt.Println("rerun with --apply to modify the live WireGuard interface")
		return
	}

	cmd := exec.Command("wg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("wg command failed: %v", err)
	}

	fmt.Printf("peer added to %s with address %s\n", spec.Interface, spec.Address)
}
