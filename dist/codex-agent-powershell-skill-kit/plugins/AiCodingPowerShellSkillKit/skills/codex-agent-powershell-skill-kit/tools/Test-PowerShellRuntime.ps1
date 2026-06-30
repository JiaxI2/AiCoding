[CmdletBinding()]
param(
    [switch]$Json,
    [int]$MinimumMajor = 7,
    [switch]$AllowWindowsPowerShellFallback
)

$ErrorActionPreference = 'Stop'

function New-Result {
    param(
        [string]$Runtime,
        [string]$Path,
        [string]$Version,
        [string]$Edition,
        [bool]$Ok,
        [string[]]$Messages
    )
    [pscustomobject]@{
        Runtime = $Runtime
        Path = $Path
        Version = $Version
        Edition = $Edition
        Ok = $Ok
        Messages = $Messages
    }
}

$messages = New-Object System.Collections.Generic.List[string]
$pwsh = Get-Command pwsh -ErrorAction SilentlyContinue
if ($pwsh) {
    $infoJson = & $pwsh.Source -NoProfile -Command '[pscustomobject]@{Version=$PSVersionTable.PSVersion.ToString();PSEdition=$PSVersionTable.PSEdition} | ConvertTo-Json -Compress'
    $info = $infoJson | ConvertFrom-Json
    $major = [version]$info.Version
    if ($major.Major -ge $MinimumMajor) {
        $result = New-Result -Runtime 'pwsh' -Path $pwsh.Source -Version $info.Version -Edition $info.PSEdition -Ok $true -Messages @('PowerShell 7+ runtime detected.')
    } else {
        $result = New-Result -Runtime 'pwsh' -Path $pwsh.Source -Version $info.Version -Edition $info.PSEdition -Ok $false -Messages @("pwsh found but major version is lower than $MinimumMajor.")
    }
} else {
    $legacy = Get-Command powershell.exe -ErrorAction SilentlyContinue
    if ($legacy -and $AllowWindowsPowerShellFallback) {
        $infoJson = & $legacy.Source -NoProfile -Command '[pscustomobject]@{Version=$PSVersionTable.PSVersion.ToString();PSEdition=$PSVersionTable.PSEdition} | ConvertTo-Json -Compress'
        $info = $infoJson | ConvertFrom-Json
        $result = New-Result -Runtime 'powershell.exe' -Path $legacy.Source -Version $info.Version -Edition $info.PSEdition -Ok $true -Messages @('Fallback to Windows PowerShell allowed by parameter.')
    } elseif ($legacy) {
        $result = New-Result -Runtime 'powershell.exe' -Path $legacy.Source -Version '5.1-or-legacy' -Edition 'Desktop' -Ok $false -Messages @('pwsh not found. Windows PowerShell fallback exists but is not allowed by default.')
    } else {
        $result = New-Result -Runtime 'none' -Path '' -Version '' -Edition '' -Ok $false -Messages @('No PowerShell runtime found.')
    }
}

if ($Json) {
    $result | ConvertTo-Json -Depth 5
} else {
    $result | Format-List
}

if (-not $result.Ok) { throw ($result.Messages -join '; ') }
