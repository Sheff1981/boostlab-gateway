$ErrorActionPreference = "Stop"

$ruleName = "BOOSTLAB Local Probe UDP 51821"
$existing = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue

if ($existing) {
    Remove-NetFirewallRule -DisplayName $ruleName
    Write-Host "Removed: $ruleName"
} else {
    Write-Host "Firewall rule not found: $ruleName"
}
