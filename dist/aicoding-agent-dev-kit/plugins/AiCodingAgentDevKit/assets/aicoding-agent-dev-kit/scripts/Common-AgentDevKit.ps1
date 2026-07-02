function Resolve-AgentDevKitRepoRoot {
    param([string]$RepoRoot = ".")
    return (Resolve-Path -LiteralPath $RepoRoot).Path
}

function Write-AgentDevKitJson {
    param([hashtable]$Data, [switch]$Json)
    if ($Json) {
        $Data | ConvertTo-Json -Depth 10
    } else {
        foreach ($k in $Data.Keys) { Write-Host "$($k): $($Data[$k])" }
    }
}

function New-AgentDevKitDirectory {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Copy-AgentDevKitTree {
    param([string]$Source, [string]$Destination)
    if (Test-Path -LiteralPath $Source) {
        New-AgentDevKitDirectory $Destination
        Copy-Item -LiteralPath (Join-Path $Source "*") -Destination $Destination -Recurse -Force
    }
}

function Write-AgentDevKitState {
    param([string]$RepoRoot, [array]$Files)
    $stateDir = Join-Path $RepoRoot ".agent-dev-kit"
    New-AgentDevKitDirectory $stateDir
    $state = [ordered]@{
        schema = "aicoding-agent-dev-kit.install-state.v1"
        version = "0.11.1"
        installedAt = (Get-Date).ToString("o")
        ownedFiles = $Files
    }
    $state | ConvertTo-Json -Depth 10 | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $stateDir "install-state.json")
}
