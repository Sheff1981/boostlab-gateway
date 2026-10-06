# Windows local test — no VPS required

This mode exists so BOOSTLAB can be tested before paying for a VPS.

It validates Android-to-Windows UDP reachability, automatic LAN discovery, RTT/jitter/packet-loss measurement, and the client/gateway probe protocol.

It does not prove that the route is a real gaming boost. A PC on the same home network does not create a useful alternative Internet route.

## What you need

- Windows 10/11 PC on the same Wi-Fi/LAN as the Android phone.
- The GitHub Actions artifact named boostlab-gateway-windows-amd64.
- Private/home network profile in Windows.

## Start

1. Download and extract boostlab-gateway-windows-amd64.exe from the latest green Gateway CI artifact.
2. Put start-local-test.ps1 beside the EXE.
3. Open PowerShell as Administrator once and run:

    .\start-local-test.ps1 -OpenFirewall

The firewall rule opens only UDP 51821 on the Windows Private profile.

4. On Android, open BOOSTLAB and tap:

    Найти локальный сервер без VPS

5. Android broadcasts the normal tiny BOOSTLAB probe packet on the LAN. The Windows gateway replies, and Android automatically measures the discovered gateway.

## Later starts

After the firewall rule exists, Administrator rights are not required:

    .\start-local-test.ps1

## Cleanup

To remove the local firewall rule, run PowerShell as Administrator:

    .\remove-firewall-rule.ps1

## Limits

The local test gateway only proves connectivity and measurement. A real WireGuard tunnel that can change Internet routing still requires a reachable gateway outside the phone's normal path: typically a VPS, server, or another remotely reachable machine.
