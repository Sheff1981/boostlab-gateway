param(
    [string]$GatewayExe = ".\boostlab-gateway-windows-amd64.exe",
    [switch]$OpenFirewall
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $GatewayExe)) {
    throw "Gateway executable not found: $GatewayExe"
}

if ($OpenFirewall) {
    $principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $isAdmin = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

    if (-not $isAdmin) {
        throw "Run PowerShell as Administrator when using -OpenFirewall."
    }

    $ruleName = "BOOSTLAB Local Probe UDP 51821"
    $existing = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if (-not $existing) {
        New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Action Allow -Protocol UDP -LocalPort 51821 -Profile Private | Out-Null
    }
}

$env:BOOSTLAB_NODE_ID = "windows-local"
$env:BOOSTLAB_REGION = "LAN"
$env:BOOSTLAB_HTTP_ADDR = "127.0.0.1:8080"
$env:BOOSTLAB_UDP_ADDR = ":51821"

Write-Host "BOOSTLAB local gateway starting."
Write-Host "Probe: UDP 51821"
Write-Host "Health: http://127.0.0.1:8080/healthz"
Write-Host "Android: tap 'Найти локальный сервер без VPS'"
Write-Host ""
Write-Host "Press Ctrl+C to stop."

& $GatewayExe
