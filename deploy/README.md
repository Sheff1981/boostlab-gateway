# Gateway deployment baseline

The current gateway exposes:

- TCP `8080` for `/healthz`, `/readyz`, and `/v1/status`;
- UDP `51821` for the lightweight route-quality probe.

## Minimum VPS requirements

A small Linux VPS with:

- a public IPv4 or IPv6 address;
- inbound UDP `51821`;
- inbound TCP `8080` only while testing, or restricted by firewall later;
- systemd.

## Install outline

1. Build `boostlab-gateway` for Linux amd64.
2. Create a dedicated system user named `boostlab`.
3. Place the binary at `/usr/local/bin/boostlab-gateway`.
4. Copy `gateway.env.example` to `/etc/boostlab/gateway.env` and set a unique node ID/region.
5. Install `boostlab-gateway.service` under `/etc/systemd/system/`.
6. Open UDP `51821` in the VPS/provider firewall.
7. Start the service and verify `/healthz`.
8. Put the public IP into the Android Stage 2 screen and run the probe.

Do not enable application traffic tunnelling yet. Stage 2 validates that the phone can reach the gateway and that the measured route quality is stable.
