# BOOSTLAB WireGuard data plane

BOOSTLAB uses the official WireGuard protocol for the encrypted tunnel. The custom gateway service remains responsible for lightweight route-quality probes; it does not implement custom cryptography.

## Ports

- UDP 51820 — WireGuard tunnel.
- UDP 51821 — BOOSTLAB route-quality probe.
- TCP 8080 — temporary gateway health/status endpoint.

## Server setup outline

1. Install WireGuard and nftables using the Linux distribution packages.
2. Generate the server private key **on the server**:
   `umask 077; wg genkey | tee /etc/wireguard/server.key | wg pubkey`
3. Put the private key into `/etc/wireguard/wg0.conf` with mode `0600`.
4. Enable IPv4 forwarding and, when IPv6 is enabled for the product, configure IPv6 forwarding deliberately rather than leaking around the tunnel.
5. Replace `<WAN_IFACE>` in the nftables example with the real public interface.
6. Add the client's public key as a WireGuard peer with a unique tunnel address.
7. Open inbound UDP 51820 and 51821 in the provider firewall.
8. Start `wg-quick@wg0` and the BOOSTLAB gateway probe service.

## Client identity

The Android app generates its WireGuard identity locally. Its private key is encrypted at rest with an Android Keystore AES-GCM wrapping key. Only the public key should be copied to or registered with the server.

Do not put server or client private keys in Git, environment examples, screenshots, logs, or the control-plane node list.

## Per-app routing

The Android WireGuard config uses `IncludedApplications` so only the user-selected Android package is routed through the tunnel. Other applications continue to use the phone's normal network path.

## Current status

The Android WireGuard backend and secure client identity storage are present as Stage 4 groundwork. They are not connected automatically until a real gateway has a server key, client peer, tunnel address, and validated route.
