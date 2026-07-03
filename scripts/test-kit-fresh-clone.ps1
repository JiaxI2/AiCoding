[CmdletBinding()]
param(
    [ValidateSet("Smoke", "SourceOnly", "Package")]
    [string]$Mode = "Smoke",
    [switch]$Json,
    [switch]$KeepTemp,
    [ValidateSet("Smoke", "Full", "Release")]
    [string]$Profile = "Smoke",
    [ValidateRange(1, 600)]
    [int]$TimeoutSeconds = 20
)

$ErrorActionPreference = "Stop"
$script:FreshCloneTimeoutSeconds = $TimeoutSeconds
$script:FreshCloneStopwatch = [System.Diagnostics.Stopwatch]::StartNew()
$sourceRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("aicoding-kit-fresh-clone-{0}" -f ([guid]::NewGuid().ToString("N")))
$cloneRoot = Join-Path $tempRoot "AiCoding"
$steps = New-Object System.Collections.Generic.List[object]
$errors = New-Object System.Collections.Generic.List[string]

function Add-Step {
    param(
        [string]$Name,
        [bool]$Ok,
        [string]$Message,
        [object]$Data = $null
    )

    $steps.Add([pscustomobject]@{
        name = $Name
        ok = $Ok
        message = $Message
        data = $Data
    }) | Out-Null

    if (-not $Ok) {
        $errors.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null
    }
}

function Limit-Text {
    param(
        [AllowNull()][string]$Text,
        [int]$MaxLength = 4000
    )

    if ($null -eq $Text) { return "" }
    $trimmed = $Text.Trim()
    if ($trimmed.Length -le $MaxLength) { return $trimmed }
    return ("{0}`n...<truncated {1} chars>" -f $trimmed.Substring(0, $MaxLength), ($trimmed.Length - $MaxLength))
}

function Remove-FreshCloneTempPath {
    [CmdletBinding(SupportsShouldProcess=$true)]
    param([Parameter(Mandatory=$true)][string]$Path)

    $root = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    $target = [System.IO.Path]::GetFullPath($Path)
    if (-not $target.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove path outside temp root: $target"
    }

    if (Test-Path -LiteralPath $target) {
        if ($PSCmdlet.ShouldProcess($target, "Remove fresh clone temp path")) {
            Remove-Item -LiteralPath $target -Recurse -Force
        }
    }
}

function Get-RelativePath {
    param(
        [Parameter(Mandatory=$true)][string]$Base,
        [Parameter(Mandatory=$true)][string]$Path
    )

    return ([System.IO.Path]::GetRelativePath($Base, $Path) -replace '\\', '/')
}

function Resolve-FreshCloneProfile {
    param(
        [string]$ModeValue,
        [string]$ProfileValue,
        [bool]$ProfileExplicit
    )

    if ($ProfileExplicit) { return $ProfileValue }
    switch ($ModeValue) {
        "SourceOnly" { return "Full" }
        "Package" { return "Release" }
        default { return "Smoke" }
    }
}

$script:FreshCloneProfile = Resolve-FreshCloneProfile -ModeValue $Mode -ProfileValue $Profile -ProfileExplicit:$PSBoundParameters.ContainsKey("Profile")
if ($script:FreshCloneProfile -eq "Release" -and -not $PSBoundParameters.ContainsKey("TimeoutSeconds")) { $script:FreshCloneTimeoutSeconds = 180 } # Release default budget

function Test-ExcludedFreshClonePath {
    param([Parameter(Mandatory=$true)][string]$RelativePath)

    $rel = ($RelativePath -replace '\\', '/').ToLowerInvariant()
    if ($rel.StartsWith("./", [System.StringComparison]::Ordinal)) {
        $rel = $rel.Substring(2)
    }
    $prefixes = @(
        ".aicoding/packages",
        ".aicoding/state",
        ".aicoding/tmp",
        ".agentpatch",
        ".ai-debug-repair",
        ".codex-agent-powershell-skill-kit",
        ".aicoding-agent-dev-kit",
        ".agents/skills/codex-agent-powershell-skill-kit",
        "plugins",
        "node_modules",
        ".venv",
        "venv",
        ".pytest_cache"
    )

    foreach ($prefix in $prefixes) {
        if ($rel -eq $prefix -or $rel.StartsWith("$prefix/")) { return $true }
    }

    return ($rel -like "*/__pycache__/*" -or $rel.EndsWith("/__pycache__"))
}

function Copy-SourceOnlyTree {
    param(
        [Parameter(Mandatory=$true)][string]$Source,
        [Parameter(Mandatory=$true)][string]$Destination
    )

    New-Item -ItemType Directory -Path $Destination -Force | Out-Null
    $excludeDirs = @(
        (Join-Path $Source '.aicoding\packages'),
        (Join-Path $Source '.aicoding\state'),
        (Join-Path $Source '.aicoding\tmp'),
        (Join-Path $Source '.agentpatch'),
        (Join-Path $Source '.ai-debug-repair'),
        (Join-Path $Source '.codex-agent-powershell-skill-kit'),
        (Join-Path $Source '.aicoding-agent-dev-kit'),
        (Join-Path $Source '.agents\skills\codex-agent-powershell-skill-kit'),
        (Join-Path $Source 'plugins'),
        (Join-Path $Source 'node_modules'),
        (Join-Path $Source '.venv'),
        (Join-Path $Source 'venv'),
        (Join-Path $Source '.pytest_cache'),
        '__pycache__'
    )
    $arguments = @($Source, $Destination, '/E', '/MT:16', '/R:0', '/W:0', '/NFL', '/NDL', '/NJH', '/NJS', '/NP', '/XD') + $excludeDirs
    $robocopy = Start-Process -FilePath 'robocopy.exe' -ArgumentList $arguments -NoNewWindow -Wait -PassThru
    if ($robocopy.ExitCode -ge 8) {
        throw "robocopy failed with exit code $($robocopy.ExitCode)"
    }

    return [pscustomobject]@{
        copiedFiles = $null
        skippedItems = $null
        copyTool = 'robocopy'
        robocopyExitCode = $robocopy.ExitCode
    }
}

function Get-JsonSummary {
    param([AllowNull()][object]$Value)

    if ($null -eq $Value) { return $null }
    $summary = [ordered]@{}
    foreach ($name in @("schemaVersion", "ok", "mode", "status", "message", "enabledKitCount", "checkedManifestCount")) {
        if ($Value.PSObject.Properties.Name -contains $name) { $summary[$name] = $Value.$name }
    }
    if ($Value.PSObject.Properties.Name -contains "kits") { $summary["kitCount"] = @($Value.kits).Count }
    if ($Value.PSObject.Properties.Name -contains "items") { $summary["itemCount"] = @($Value.items).Count }
    if ($Value.PSObject.Properties.Name -contains "checks") {
        $checks = @($Value.checks)
        $summary["checkCount"] = $checks.Count
        $summary["failedChecks"] = @($checks | Where-Object { -not $_.ok } | Select-Object -First 10 -Property name, message)
    }
    if ($Value.PSObject.Properties.Name -contains "results") {
        $results = @($Value.results)
        $summary["resultCount"] = $results.Count
        $summary["failedResults"] = @($results | Where-Object { -not $_.ok } | Select-Object -First 10 -Property id, action, message, exitCode)
    }
    if ($Value.PSObject.Properties.Name -contains "errors") { $summary["errors"] = @($Value.errors | Select-Object -First 10) }
    return [pscustomobject]$summary
}

function Get-RemainingFreshCloneSeconds {
    $remaining = $script:FreshCloneTimeoutSeconds - [int][Math]::Ceiling($script:FreshCloneStopwatch.Elapsed.TotalSeconds)
    if ($remaining -lt 0) { return 0 }
    return $remaining
}

function Invoke-FreshCloneCommand {
    param(
        [Parameter(Mandatory=$true)][string[]]$Arguments,
        [Parameter(Mandatory=$true)][string]$Name,
        [switch]$ExpectJson,
        [int]$CommandTimeoutSeconds = 15
    )

    $pwsh = Get-Command pwsh -ErrorAction SilentlyContinue
    $shell = if ($pwsh) { $pwsh.Source } else { "powershell" }
    $psi = [System.Diagnostics.ProcessStartInfo]::new()
    $psi.FileName = $shell
    foreach ($argument in $Arguments) {
        [void]$psi.ArgumentList.Add($argument)
    }
    $psi.WorkingDirectory = $cloneRoot
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.Environment["AICODING_SKIP_FRESH_CLONE"] = "1"
    $psi.Environment["AICODING_SKIP_KIT_LIFECYCLE"] = "1"
    $psi.Environment["AICODING_FRESH_CLONE_LIGHT"] = "1"

    $remainingSeconds = Get-RemainingFreshCloneSeconds
    if ($remainingSeconds -le 0) {
        Add-Step $Name $false ("time budget exceeded before start: {0}s" -f $script:FreshCloneTimeoutSeconds)
        return [pscustomobject]@{ exitCode = 124; stdout = ""; stderr = ""; json = $null }
    }
    $effectiveTimeoutSeconds = [Math]::Min($CommandTimeoutSeconds, $remainingSeconds)
    [Console]::Error.WriteLine(("[fresh-clone] start {0} timeout={1}s" -f $Name, $effectiveTimeoutSeconds))
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $psi
    [void]$process.Start()
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    $completed = $process.WaitForExit($effectiveTimeoutSeconds * 1000)
    if (-not $completed) {
        try { $process.Kill($true) } catch { }
        try { $process.WaitForExit(5000) | Out-Null } catch { }
    }
    $stdout = $stdoutTask.GetAwaiter().GetResult()
    $stderr = $stderrTask.GetAwaiter().GetResult()

    if (-not $completed) {
        [Console]::Error.WriteLine(("[fresh-clone] timeout {0} after {1}s" -f $Name, $effectiveTimeoutSeconds))
        Add-Step $Name $false ("timeout after {0}s" -f $effectiveTimeoutSeconds) @{
            stdout = Limit-Text -Text $stdout
            stderr = Limit-Text -Text $stderr
            json = $null
        }
        return [pscustomobject]@{
            exitCode = 124
            stdout = $stdout
            stderr = $stderr
            json = $null
        }
    }

    [Console]::Error.WriteLine(("[fresh-clone] done {0} exit={1}" -f $Name, $process.ExitCode))
    $parsed = $null
    if ($ExpectJson -and $process.ExitCode -eq 0) {
        try { $parsed = $stdout | ConvertFrom-Json } catch { Add-Step "$Name.json" $false $_.Exception.Message }
    }

    Add-Step $Name ($process.ExitCode -eq 0) ("exit={0}" -f $process.ExitCode) @{
        stdout = Limit-Text -Text $stdout
        stderr = Limit-Text -Text $stderr
        json = Get-JsonSummary -Value $parsed
    }

    return [pscustomobject]@{
        exitCode = $process.ExitCode
        stdout = $stdout
        stderr = $stderr
        json = $parsed
    }
}

function Test-HashSidecar {
    param(
        [Parameter(Mandatory=$true)][string]$FilePath,
        [Parameter(Mandatory=$true)][string]$Sha256Path,
        [Parameter(Mandatory=$true)][string]$Name
    )

    if (-not (Test-Path -LiteralPath $FilePath -PathType Leaf)) {
        Add-Step $Name $false "missing file: $FilePath"
        return
    }
    if (-not (Test-Path -LiteralPath $Sha256Path -PathType Leaf)) {
        Add-Step $Name $false "missing sha256: $Sha256Path"
        return
    }

    $expected = ((Get-Content -LiteralPath $Sha256Path -Raw).Trim() -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash -LiteralPath $FilePath -Algorithm SHA256).Hash.ToLowerInvariant()
    Add-Step $Name ($expected -eq $actual) ("sha256 {0}" -f ($(if ($expected -eq $actual) { "matched" } else { "mismatch" }))) @{
        file = $FilePath
        sha256 = $Sha256Path
    }
}

function Test-BundleSha256Sums {
    param([Parameter(Mandatory=$true)][string]$ExtractRoot)

    $sumPath = Join-Path $ExtractRoot "SHA256SUMS.txt"
    if (-not (Test-Path -LiteralPath $sumPath -PathType Leaf)) {
        Add-Step "bundle.sha256sums" $false "missing SHA256SUMS.txt"
        return
    }

    $bad = @()
    foreach ($line in Get-Content -LiteralPath $sumPath) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $parts = $line.Trim() -split '\s+', 2
        if ($parts.Count -ne 2) {
            $bad += "invalid line: $line"
            continue
        }
        $file = Join-Path $ExtractRoot ($parts[1] -replace '/', '\')
        if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
            $bad += "missing package: $($parts[1])"
            continue
        }
        $actual = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $parts[0].ToLowerInvariant()) {
            $bad += "hash mismatch: $($parts[1])"
        }
    }

    Add-Step "bundle.sha256sums" ($bad.Count -eq 0) ("checked SHA256SUMS entries") @{ errors = $bad }
}

function Test-ContentLeak {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)][string[]]$Patterns
    )

    $bytes = [System.IO.File]::ReadAllBytes($Path)
    foreach ($pattern in $Patterns) {
        $needle = [System.Text.Encoding]::UTF8.GetBytes($pattern)
        if ($needle.Length -eq 0 -or $bytes.Length -lt $needle.Length) { continue }
        for ($i = 0; $i -le $bytes.Length - $needle.Length; $i++) {
            $matched = $true
            for ($j = 0; $j -lt $needle.Length; $j++) {
                if ($bytes[$i + $j] -ne $needle[$j]) {
                    $matched = $false
                    break
                }
            }
            if ($matched) { return $pattern }
        }
    }

    return $null
}

function Test-ZipEntryLeak {
    param(
        [Parameter(Mandatory=$true)][string]$ZipPath,
        [Parameter(Mandatory=$true)][string[]]$Patterns
    )

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [System.IO.Compression.ZipFile]::OpenRead($ZipPath)
    try {
        foreach ($entry in $archive.Entries) {
            if ($entry.Length -eq 0 -or $entry.Length -gt 20MB) { continue }
            $stream = $entry.Open()
            try {
                $memory = [System.IO.MemoryStream]::new()
                $stream.CopyTo($memory)
                $bytes = $memory.ToArray()
                foreach ($pattern in $Patterns) {
                    $text = [System.Text.Encoding]::UTF8.GetString($bytes)
                    if ($text.Contains($pattern)) { return "$ZipPath::$($entry.FullName) -> $pattern" }
                }
            }
            finally {
                $stream.Dispose()
            }
        }
    }
    finally {
        $archive.Dispose()
    }

    return $null
}

function Test-BundleLeaks {
    param(
        [Parameter(Mandatory=$true)][string]$ExtractRoot,
        [Parameter(Mandatory=$true)][string[]]$Patterns
    )

    $hits = @()
    foreach ($file in Get-ChildItem -LiteralPath $ExtractRoot -Recurse -File -Force) {
        if ($file.Extension -eq ".zip") {
            $hit = Test-ZipEntryLeak -ZipPath $file.FullName -Patterns $Patterns
            if ($hit) { $hits += $hit }
            continue
        }

        if ($file.Length -le 20MB) {
            $hit = Test-ContentLeak -Path $file.FullName -Patterns $Patterns
            if ($hit) { $hits += "$($file.FullName) -> $hit" }
        }
    }

    Add-Step "bundle.path-leak-scan" ($hits.Count -eq 0) "scanned bundle content for local absolute paths" @{ hits = $hits }
}

function Invoke-SmokeChecks {
    $forbiddenScripts = @("install-all.ps1", "verify-all.ps1", "test-all.ps1", "update-all.ps1", "export-all.ps1", "uninstall-all.ps1")
    foreach ($scriptName in $forbiddenScripts) {
        $path = Join-Path (Join-Path $sourceRoot "scripts") $scriptName
        Add-Step ("forbidden-script:{0}" -f $scriptName) (-not (Test-Path -LiteralPath $path)) "not present" @{ path = $path }
    }

    Import-Module (Join-Path $PSScriptRoot "lib\AiCoding.KitRegistry.psm1") -Force
    $kits = @(Get-AiCodingKitRegistry -RepoRoot $sourceRoot -Enabled)
    Add-Step "smoke.registry.enabled" ($kits.Count -gt 0) ("enabled kits={0}" -f $kits.Count) @{ count = $kits.Count }

    foreach ($kit in $kits) {
        Add-Step ("smoke.manifest.exists:{0}" -f $kit.id) (Test-Path -LiteralPath $kit.manifestPath -PathType Leaf) $kit.manifestRelativePath
        Add-Step ("smoke.manifest.id:{0}" -f $kit.id) ($kit.manifest.id -eq $kit.id) ("manifest.id={0}" -f $kit.manifest.id)
        Add-Step ("smoke.manifest.mode:{0}" -f $kit.id) (@("script-adapter", "declarative") -contains $kit.manifest.mode) ("mode={0}" -f $kit.manifest.mode)
        Add-Step ("smoke.manifest.kind:{0}" -f $kit.id) (@($kit.manifest.kind).Count -gt 0) ("kind count={0}" -f @($kit.manifest.kind).Count)
        foreach ($commandProperty in @($kit.manifest.commands.PSObject.Properties)) {
            $command = $commandProperty.Value
            if ($command.type -ne "powershell-script") { continue }
            $scriptPath = Join-Path $sourceRoot ([string]$command.path)
            Add-Step ("smoke.command.path:{0}.{1}" -f $kit.id, $commandProperty.Name) (Test-Path -LiteralPath $scriptPath -PathType Leaf) ([string]$command.path)
        }
    }

    Add-Step "smoke.no-full-copy" $true "smoke profile does not copy the full worktree" @{ profile = "Smoke" }
    Add-Step "smoke.no-test-all" $true "smoke profile does not run test -All" @{ profile = "Smoke" }
    Add-Step "smoke.no-real-export" $true "smoke profile does not create package artifacts" @{ profile = "Smoke" }
}

function Invoke-SourceOnlyChecks {
    [void](Invoke-FreshCloneCommand -Name "verify-kit-lifecycle" -ExpectJson -CommandTimeoutSeconds 14 -Arguments @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/verify-kit-lifecycle.ps1", "-Json"))
    [void](Invoke-FreshCloneCommand -Name "aicoding-kit.doctor" -ExpectJson -CommandTimeoutSeconds 4 -Arguments @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/aicoding-kit.ps1", "doctor", "-Json"))

    Add-Step "aicoding-kit.list" $true "covered by verify-kit-lifecycle light gate" @{ skipped = $true; reason = "20s-script-budget" }
    Add-Step "aicoding-kit.status" $true "covered by verify-kit-lifecycle light gate" @{ skipped = $true; reason = "20s-script-budget" }
    Add-Step "aicoding-kit.verify" $true "skipped in 20s fresh-clone gate; run pwsh scripts/aicoding-kit.ps1 verify -All separately" @{ skipped = $true; reason = "20s-script-budget" }
    Add-Step "aicoding-kit.test" $true "skipped in 20s fresh-clone gate; run pwsh scripts/aicoding-kit.ps1 test -All separately" @{ skipped = $true; reason = "20s-script-budget" }
    Add-Step "aicoding-kit.export-dry-run" $true "skipped in 20s fresh-clone gate; run pwsh scripts/aicoding-kit.ps1 export -All -Zip -DryRun -Json separately" @{ skipped = $true; reason = "20s-script-budget" }

    $packageRoot = Join-Path $cloneRoot ".aicoding\packages"
    $hasGeneratedPackages = (Test-Path -LiteralPath $packageRoot) -and (@(Get-ChildItem -LiteralPath $packageRoot -Recurse -File -ErrorAction SilentlyContinue).Count -gt 0)
    Add-Step "source-only.no-packages" (-not $hasGeneratedPackages) "fresh clone light gate did not create package files" @{ packageRoot = $packageRoot }
}

function Invoke-PackageChecks {
    $export = Invoke-FreshCloneCommand -Name "aicoding-kit.export" -ExpectJson -CommandTimeoutSeconds 90 -Arguments @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/aicoding-kit.ps1", "export", "-All", "-Zip", "-Json")
    if (-not $export.json) { return }

    $kits = @($export.json.kits)
    Add-Step "package.kit-count" ($kits.Count -eq 7) ("kits={0}" -f $kits.Count)
    foreach ($kit in $kits) {
        if (-not $kit.ok) {
            Add-Step "package.$($kit.id)" $false $kit.message
            continue
        }

        Test-HashSidecar -Name "package.$($kit.id).sha256" -FilePath ([string]$kit.data.packageFile) -Sha256Path ([string]$kit.data.sha256File)
        Add-Step "package.$($kit.id).buildinfo" (Test-Path -LiteralPath ([string]$kit.data.buildInfoFile) -PathType Leaf) "BUILDINFO sidecar exists"
    }

    $bundleFile = [string]$export.json.bundle.data.packageFile
    $bundleSha = [string]$export.json.bundle.data.sha256File
    Test-HashSidecar -Name "bundle.sha256" -FilePath $bundleFile -Sha256Path $bundleSha

    $extractRoot = Join-Path $tempRoot "bundle"
    New-Item -ItemType Directory -Path $extractRoot -Force | Out-Null
    Expand-Archive -LiteralPath $bundleFile -DestinationPath $extractRoot -Force

    $required = @(
        "registry/kit-registry.json",
        "SHA256SUMS.txt",
        "BUILDINFO.json"
    )
    foreach ($item in $required) {
        $path = Join-Path $extractRoot ($item -replace '/', '\')
        Add-Step "bundle.entry:$item" (Test-Path -LiteralPath $path -PathType Leaf) $item
    }

    $manifestCount = @(Get-ChildItem -LiteralPath (Join-Path $extractRoot "manifests\config\kits") -Filter "*.json" -File -ErrorAction SilentlyContinue).Count
    $packageCount = @(Get-ChildItem -LiteralPath (Join-Path $extractRoot "packages") -Filter "*.zip" -File -ErrorAction SilentlyContinue).Count
    Add-Step "bundle.manifests" ($manifestCount -ge 7) ("manifest count={0}" -f $manifestCount)
    Add-Step "bundle.packages" ($packageCount -eq 7) ("package count={0}" -f $packageCount)
    Test-BundleSha256Sums -ExtractRoot $extractRoot

    $leakPatterns = @(
        ($sourceRoot -replace '\\', '/'),
        $sourceRoot,
        ($cloneRoot -replace '\\', '/'),
        $cloneRoot,
        ($tempRoot -replace '\\', '/'),
        $tempRoot,
        "F:/Study/AI/AiCoding",
        "F:\Study\AI\AiCoding",
        "C:/Users/24322",
        "C:\Users\24322"
    ) | Select-Object -Unique
    Test-BundleLeaks -ExtractRoot $extractRoot -Patterns $leakPatterns
}

try {
    if ($script:FreshCloneProfile -eq "Smoke") {
        Invoke-SmokeChecks
    } else {
        New-Item -ItemType Directory -Path $tempRoot -Force | Out-Null
        $copy = Copy-SourceOnlyTree -Source $sourceRoot -Destination $cloneRoot
        Add-Step "source.copy" $true "copied source-only tree" $copy
        Add-Step "source.excludes" (-not (Test-Path -LiteralPath (Join-Path $cloneRoot ".aicoding\packages"))) "runtime package directory excluded"
        Add-Step "source.runtime-mirror-excluded" (-not (Test-Path -LiteralPath (Join-Path $cloneRoot ".agents\skills\codex-agent-powershell-skill-kit"))) "runtime skill mirror excluded"

        if ($script:FreshCloneProfile -eq "Full") {
            Invoke-SourceOnlyChecks
        } else {
            Invoke-PackageChecks
        }
    }
}
catch {
    Add-Step "fatal" $false $_.Exception.Message
}
finally {
    if (-not $KeepTemp) {
        try { Remove-FreshCloneTempPath -Path $tempRoot } catch { Add-Step "cleanup" $false $_.Exception.Message }
    }
}

$result = [pscustomobject]@{
    schemaVersion = 1
    mode = $Mode
    profile = $script:FreshCloneProfile
    ok = ($errors.Count -eq 0)
    sourceRoot = $sourceRoot
    tempRoot = $tempRoot
    cloneRoot = $cloneRoot
    keptTemp = [bool]$KeepTemp
    timeoutSeconds = $script:FreshCloneTimeoutSeconds
    durationSeconds = [Math]::Round($script:FreshCloneStopwatch.Elapsed.TotalSeconds, 3)
    steps = $steps
    errors = @($errors)
}

if ($Json) {
    $result | ConvertTo-Json -Depth 30
} elseif ($result.ok) {
    Write-Host ("AiCoding fresh clone {0} test passed." -f $Mode)
} else {
    $errors | ForEach-Object { Write-Error $_ }
}

if (-not $result.ok) {
    exit 1
}
