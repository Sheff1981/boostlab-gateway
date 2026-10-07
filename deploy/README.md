# Gateway deployment baseline

BOOSTLAB Gateway is the low-latency data-plane service used by Android Network Boost.

## Ports

- UDP `51820` — WireGuard game tunnel.
- UDP `51821` — BOOSTLAB latency / route-quality probe.
- TCP `8080` — local HTTP service for health, route quality, runtime status and signed peer provisioning. Put it behind HTTPS for Android access.

## Minimum VPS requirements

A small Linux VPS is enough for the control/probe workload. Network location and routing quality matter more than CPU size for game latency.

Required:

- public IPv4; IPv6 can be added later;
- WireGuard installed and `wg0` active;
- systemd;
- inbound UDP 51820 and 51821;
- HTTPS reverse proxy to the gateway HTTP service.

## Automatic phone provisioning

Android no longer needs a manual `peerctl` command for every phone.

The flow is:

1. Android proves possession of its non-exportable ECDSA device key to BOOSTLAB Control.
2. Control issues a 90-second provisioning ticket bound to one Gateway and one WireGuard public key.
3. Gateway validates the HMAC ticket.
4. Gateway allocates a free `10.77.0.x/32` address and applies the peer with `wg set`.
5. The peer registry is atomically stored in `/var/lib/boostlab/peers.json` and restored after reboot.

The shared `BOOSTLAB_PROVISIONING_SECRET` exists only on Control and Gateway. It must never be placed in Android, Git, logs or screenshots.

Generate it once:

    openssl rand -hex 32

Use the same value in `/etc/boostlab/control.env` and `/etc/boostlab/gateway.env`.

## systemd permissions

The gateway service runs as the unprivileged `boostlab` user. It receives only `CAP_NET_ADMIN` and `AF_NETLINK` access so it can call `wg set`. It does not need root access or write access to `/etc/wireguard`.

The persistent peer registry is stored in the systemd-managed state directory.

## Install outline

1. Install WireGuard and configure `wg0` with server address `10.77.0.1/24` and UDP port 51820.
2. Generate the WireGuard server private key on the VPS; never put it in Git.
3. Install `boostlab-gateway` at `/usr/local/bin/boostlab-gateway`.
4. Copy `gateway.env.example` to `/etc/boostlab/gateway.env`.
5. Set node ID, region, public WireGuard key and the shared provisioning secret.
6. Install `boostlab-gateway.service`.
7. Run `systemctl daemon-reload && systemctl enable --now wg-quick@wg0 boostlab-gateway`.
8. Put the HTTP service behind HTTPS and publish that URL as `route_api_url` in BOOSTLAB Control.
9. Verify `/healthz`, `/v1/status`, UDP probe 51821 and an Android peer enrollment.

## Security boundaries

- Client WireGuard private keys stay on each Android device.
- The WireGuard server private key stays only on the VPS.
- Public device enrollment is not open: a new phone needs the one-time Control enrollment code.
- Provisioning tickets expire after 90 seconds and are bound to a specific Gateway and WireGuard public key.
