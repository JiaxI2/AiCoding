[CmdletBinding()]
param(
    [Parameter(Position=0, Mandatory=$true)]
    [ValidateSet("list", "install", "status", "verify", "test", "update", "uninstall", "export", "doctor", "skills", "verify-skills")]
    [string]$Action,

    [string]$Kit = "",
    [switch]$All,
    [switch]$Json,
    [switch]$DryRun,
    [switch]$Zip,
    [string]$RepoRoot = "",
    [ValidateSet("Smoke", "Full", "Release")]
    [string]$Profile = "Smoke"
)

$ErrorActionPreference = "Stop"

Import-Module (Join-Path $PSScriptRoot "lib\AiCoding.KitRunner.psm1") -Force

try {
    $result = Invoke-AiCodingKitAction -RepoRoot $RepoRoot -Action $Action -Kit $Kit -All:$All -Json:$Json -DryRun:$DryRun -Zip:$Zip -Profile $Profile

    if ($Json) {
        $result | ConvertTo-Json -Depth 80
    } elseif ($Action -eq "list") {
        $result.kits | Format-Table -AutoSize id, enabled, order, version, mode, manifest
    } else {
        Write-Host ("AiCoding Kit Lifecycle Summary")
        Write-Host ("Action: {0}" -f $result.action)
        Write-Host ("Mode: {0}" -f $result.mode)
        foreach ($kitResult in @($result.kits)) {
            $label = if ($kitResult.ok) { "OK" } else { "FAIL" }
            Write-Host ("[{0}] {1} - {2}" -f $label, $kitResult.id, $kitResult.status)
            if (-not $kitResult.ok -and $kitResult.stderr) { Write-Host $kitResult.stderr.Trim() }
        }
        Write-Host ("Total: {0}" -f $result.summary.total)
        Write-Host ("OK:    {0}" -f $result.summary.ok)
        Write-Host ("Fail:  {0}" -f $result.summary.failed)
    }

    if (-not $result.ok) { exit 1 }
}
catch {
    $errorResult = [ordered]@{
        schemaVersion = 2
        action = $Action
        ok = $false
        error = $_.Exception.Message
    }
    if ($Json) { $errorResult | ConvertTo-Json -Depth 20 } else { Write-Error $_.Exception.Message }
    exit 1
}
