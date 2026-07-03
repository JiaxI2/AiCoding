[CmdletBinding()]
param(
    [string]$RepoRoot = "",
    [switch]$Json,
    [ValidateSet("Smoke", "Full", "Release")]
    [string]$Profile = "Smoke"
)
$ErrorActionPreference = "Stop"

$repo = if ($RepoRoot) {
    (Resolve-Path -LiteralPath $RepoRoot).Path
} else {
    (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
}

$checks = New-Object System.Collections.Generic.List[object]
$errors = New-Object System.Collections.Generic.List[string]

function Add-Check {
    param(
        [string]$Name,
        [bool]$Ok,
        [string]$Message,
        [object]$Data = $null
    )

    $checks.Add([pscustomobject]@{ name = $Name; ok = $Ok; message = $Message; data = $Data }) | Out-Null
    if (-not $Ok) { $errors.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null }
}

function Resolve-RepoPath {
    param([Parameter(Mandatory=$true)][string]$Path)
    return (Join-Path $repo ($Path -replace '/', '\'))
}

function Read-JsonFile {
    param([string]$Path, [string]$Name)
    try {
        $raw = Get-Content -LiteralPath $Path -Raw -Encoding UTF8
        $value = $raw | ConvertFrom-Json
        Add-Check $Name $true "parsed"
        return [pscustomobject]@{ ok = $true; raw = $raw; value = $value }
    } catch {
        Add-Check $Name $false $_.Exception.Message
        return [pscustomobject]@{ ok = $false; raw = ""; value = $null }
    }
}

function Test-JsonSchema {
    param([string]$RawJson, [string]$SchemaPath, [string]$Name)
    try {
        $ok = $RawJson | Test-Json -SchemaFile $SchemaPath
        Add-Check $Name ([bool]$ok) ($(if ($ok) { "valid" } else { "invalid" }))
        return [bool]$ok
    } catch {
        Add-Check $Name $false $_.Exception.Message
        return $false
    }
}

function Invoke-Capture {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [hashtable]$Environment = @{},
        [int]$TimeoutSeconds = 20
    )

    $psi = [System.Diagnostics.ProcessStartInfo]::new()
    $psi.FileName = $FilePath
    foreach ($argument in $Arguments) { [void]$psi.ArgumentList.Add($argument) }
    $psi.WorkingDirectory = $repo
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    foreach ($key in $Environment.Keys) { $psi.Environment[$key] = [string]$Environment[$key] }

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $psi
    $timedOut = $false
    [void]$process.Start()
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    if ($TimeoutSeconds -gt 0) {
        if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
            $timedOut = $true
            try { $process.Kill($true) } catch { try { $process.Kill() } catch { } }
            $process.WaitForExit()
        }
    } else {
        $process.WaitForExit()
    }
    try { $stdout = $stdoutTask.GetAwaiter().GetResult() } catch { $stdout = "" }
    try { $stderr = $stderrTask.GetAwaiter().GetResult() } catch { $stderr = "" }

    [pscustomobject]@{
        exitCode = $(if ($timedOut) { 124 } else { $process.ExitCode })
        stdout = $stdout
        stderr = $stderr
        timedOut = $timedOut
        timeoutSeconds = $TimeoutSeconds
    }
}

function Convert-CommandJson {
    param([string]$Text, [string]$Name)
    try { return $Text | ConvertFrom-Json } catch {
        Add-Check $Name $false ("failed to parse command JSON: {0}" -f $_.Exception.Message)
        return $null
    }
}

$registryPath = Resolve-RepoPath "config/kit-registry.json"
$registrySchemaPath = Resolve-RepoPath "config/schemas/kit-registry.schema.json"
$manifestSchemaPath = Resolve-RepoPath "config/schemas/kit-manifest.schema.json"
$aicodingKitPath = Resolve-RepoPath "scripts/aicoding-kit.ps1"
$aicodingSkillPath = Resolve-RepoPath "scripts/aicoding-skill.ps1"
$verifyCodexKitPath = Resolve-RepoPath "scripts/verify-codex-kit.ps1"
$freshCloneTestPath = Resolve-RepoPath "scripts/test-kit-fresh-clone.ps1"
$verifyCommonCodePath = Resolve-RepoPath "scripts/verify-common-code.ps1"
$verifyHooksPath = Resolve-RepoPath "scripts/verify-hooks.ps1"

foreach ($scriptName in @("install-all.ps1", "verify-all.ps1", "test-all.ps1", "update-all.ps1", "export-all.ps1", "uninstall-all.ps1")) {
    $forbiddenPath = Join-Path (Resolve-RepoPath "scripts") $scriptName
    Add-Check ("forbidden-script:{0}" -f $scriptName) (-not (Test-Path -LiteralPath $forbiddenPath)) "not present" @{ path = $forbiddenPath }
}

foreach ($required in @(
    @{ path = $registryPath; name = "registry.exists"; message = "missing config/kit-registry.json" },
    @{ path = $registrySchemaPath; name = "registry.schema.exists"; message = "missing config/schemas/kit-registry.schema.json" },
    @{ path = $manifestSchemaPath; name = "manifest.schema.exists"; message = "missing config/schemas/kit-manifest.schema.json" }
)) {
    if (-not (Test-Path -LiteralPath $required.path -PathType Leaf)) { Add-Check $required.name $false $required.message }
}

$registryRead = if (Test-Path -LiteralPath $registryPath -PathType Leaf) { Read-JsonFile $registryPath "registry.parse" } else { $null }
if ($registryRead -and $registryRead.ok -and (Test-Path -LiteralPath $registrySchemaPath -PathType Leaf)) {
    [void](Test-JsonSchema $registryRead.raw $registrySchemaPath "registry.schema")
}

$enabledKits = @()
$manifestRecords = @()
if ($registryRead -and $registryRead.value) {
    $enabledKits = @($registryRead.value.kits | Where-Object { $_.enabled -eq $true })
    Add-Check "registry.enabled-count" ($enabledKits.Count -gt 0) ("enabled kits: {0}" -f $enabledKits.Count) @{ count = $enabledKits.Count }

    foreach ($kit in $enabledKits) {
        $manifestPath = Resolve-RepoPath ([string]$kit.manifest)
        Add-Check ("manifest.exists:{0}" -f $kit.id) (Test-Path -LiteralPath $manifestPath -PathType Leaf) ([string]$kit.manifest)
        if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) { continue }

        $manifestRead = Read-JsonFile $manifestPath ("manifest.parse:{0}" -f $kit.id)
        if (-not $manifestRead.ok) { continue }
        if (Test-Path -LiteralPath $manifestSchemaPath -PathType Leaf) {
            [void](Test-JsonSchema $manifestRead.raw $manifestSchemaPath ("manifest.schema:{0}" -f $kit.id))
        }

        $manifest = $manifestRead.value
        Add-Check ("manifest.id:{0}" -f $kit.id) ($manifest.id -eq $kit.id) ("manifest.id={0}; registry.id={1}" -f $manifest.id, $kit.id)
        Add-Check ("manifest.mode:{0}" -f $kit.id) (@("script-adapter", "declarative") -contains $manifest.mode) ("mode={0}" -f $manifest.mode)
        Add-Check ("manifest.kind:{0}" -f $kit.id) (@($manifest.kind).Count -gt 0) ("kind count={0}" -f @($manifest.kind).Count)

        foreach ($commandProperty in @($manifest.commands.PSObject.Properties)) {
            $command = $commandProperty.Value
            if ($command.type -ne "powershell-script") { continue }
            $scriptPath = Resolve-RepoPath ([string]$command.path)
            Add-Check ("manifest.command.path:{0}.{1}" -f $kit.id, $commandProperty.Name) (Test-Path -LiteralPath $scriptPath -PathType Leaf) ([string]$command.path)
        }

        $manifestRecords += [pscustomobject]@{ registry = $kit; manifest = $manifest; path = $manifestPath }
    }
}

if (-not (Test-Path -LiteralPath $aicodingKitPath -PathType Leaf)) { Add-Check "aicoding-kit.exists" $false "missing scripts/aicoding-kit.ps1" }

if (Test-Path -LiteralPath $verifyCodexKitPath -PathType Leaf) {
    $verifyCodexText = Get-Content -LiteralPath $verifyCodexKitPath -Raw -Encoding UTF8
    $usesSmoke = ($verifyCodexText -match "test-kit-fresh-clone\.ps1") -and ($verifyCodexText -match "-Profile', 'Smoke'")
    $usesHeavyProfile = $verifyCodexText -match "-Profile', '(Full|Release)'"
    Add-Check "profile.verify-codex.default-smoke" ($usesSmoke -and -not $usesHeavyProfile) "verify-codex-kit default fresh clone gate uses Smoke only" @{ usesSmoke = $usesSmoke; usesHeavyProfile = $usesHeavyProfile }
} else {
    Add-Check "profile.verify-codex.exists" $false "missing scripts/verify-codex-kit.ps1"
}

if (Test-Path -LiteralPath $freshCloneTestPath -PathType Leaf) {
    $freshCloneText = Get-Content -LiteralPath $freshCloneTestPath -Raw -Encoding UTF8
    $defaultModeSmoke = $freshCloneText -match '\[string\]\$Mode\s*=\s*"Smoke"'
    $defaultProfileSmoke = $freshCloneText -match '\[string\]\$Profile\s*=\s*"Smoke"'
    Add-Check "profile.fresh-clone.default-smoke" ($defaultModeSmoke -and $defaultProfileSmoke) "test-kit-fresh-clone defaults to Smoke" @{ defaultModeSmoke = $defaultModeSmoke; defaultProfileSmoke = $defaultProfileSmoke }
} else {
    Add-Check "profile.fresh-clone.exists" $false "missing scripts/test-kit-fresh-clone.ps1"
}

$pwshCommand = Get-Command pwsh -ErrorAction SilentlyContinue
$shell = if ($pwshCommand) { $pwshCommand.Source } else { "powershell" }
$skipLifecycleEnv = @{ AICODING_SKIP_KIT_LIFECYCLE = "1"; AICODING_SKIP_FRESH_CLONE = "1" }
if ($env:AICODING_FRESH_CLONE_LIGHT -eq "1") { $skipLifecycleEnv.AICODING_FRESH_CLONE_LIGHT = "1" }

if (Test-Path -LiteralPath $freshCloneTestPath -PathType Leaf) {
    $packageRoot = Resolve-RepoPath ".aicoding/packages"
    $packageCountBefore = if (Test-Path -LiteralPath $packageRoot) { @(Get-ChildItem -LiteralPath $packageRoot -Recurse -File -ErrorAction SilentlyContinue).Count } else { 0 }
    $smokeResult = Invoke-Capture $shell @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $freshCloneTestPath, "-Profile", "Smoke", "-Json") @{ AICODING_SKIP_FRESH_CLONE = "1"; AICODING_SKIP_KIT_LIFECYCLE = "1" } 20
    Add-Check "profile.fresh-clone.smoke.exit" ($smokeResult.exitCode -eq 0) ("exit={0}" -f $smokeResult.exitCode) @{ stderr = $smokeResult.stderr.Trim(); timedOut = $smokeResult.timedOut }
    $smokeJson = if ($smokeResult.exitCode -eq 0) { Convert-CommandJson $smokeResult.stdout "profile.fresh-clone.smoke.json" } else { $null }
    if ($smokeJson) {
        $copied = @($smokeJson.steps | Where-Object { $_.name -eq "source.copy" }).Count -gt 0
        $realExport = @($smokeJson.steps | Where-Object { $_.name -eq "aicoding-kit.export" -or $_.name -eq "bundle.sha256" }).Count -gt 0
        Add-Check "profile.fresh-clone.smoke.profile" ([string]$smokeJson.profile -eq "Smoke") ("profile={0}" -f $smokeJson.profile)
        Add-Check "profile.fresh-clone.smoke.no-full-copy" (-not $copied) "Smoke must not copy full worktree" @{ sourceCopyStepPresent = $copied }
        Add-Check "profile.fresh-clone.smoke.no-real-export" (-not $realExport) "Smoke must not run real export/package" @{ realExportStepPresent = $realExport }
    }
    $packageCountAfter = if (Test-Path -LiteralPath $packageRoot) { @(Get-ChildItem -LiteralPath $packageRoot -Recurse -File -ErrorAction SilentlyContinue).Count } else { 0 }
    Add-Check "profile.fresh-clone.smoke.no-package-write" ($packageCountBefore -eq $packageCountAfter) "Smoke must not write .aicoding/packages" @{ before = $packageCountBefore; after = $packageCountAfter }
}

if (Test-Path -LiteralPath $aicodingKitPath -PathType Leaf) {
    $listResult = Invoke-Capture $shell @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $aicodingKitPath, "list", "-Json") $skipLifecycleEnv 20
    Add-Check "aicoding-kit.list.exit" ($listResult.exitCode -eq 0) ("exit={0}" -f $listResult.exitCode) @{ stderr = $listResult.stderr.Trim(); timedOut = $listResult.timedOut }
    $listJson = if ($listResult.exitCode -eq 0) { Convert-CommandJson $listResult.stdout "aicoding-kit.list.json" } else { $null }
    if ($listJson) {
        $listedEnabledIds = @($listJson.kits | Where-Object { $_.enabled -eq $true } | ForEach-Object { $_.id })
        $missingListed = @($enabledKits | Where-Object { $listedEnabledIds -notcontains $_.id } | ForEach-Object { $_.id })
        Add-Check "aicoding-kit.list.enabled" ($missingListed.Count -eq 0) ("missing: {0}" -f ($missingListed -join ", ")) @{ expected = @($enabledKits.id); listed = $listedEnabledIds }
    }

    $statusResult = Invoke-Capture $shell @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $aicodingKitPath, "status", "-All", "-Json") $skipLifecycleEnv 20
    Add-Check "aicoding-kit.status.exit" ($statusResult.exitCode -eq 0) ("exit={0}" -f $statusResult.exitCode) @{ stderr = $statusResult.stderr.Trim(); timedOut = $statusResult.timedOut }
    $statusJson = if ($statusResult.exitCode -eq 0) { Convert-CommandJson $statusResult.stdout "aicoding-kit.status.json" } else { $null }
    if ($statusJson) {
        $statusCount = @($statusJson.kits).Count
        $summaryCount = if ($null -ne $statusJson.summary.total) { [int]$statusJson.summary.total } else { -1 }
        Add-Check "aicoding-kit.status.count" (($statusCount -eq $enabledKits.Count) -and ($summaryCount -eq $enabledKits.Count)) ("kits={0}; summary.total={1}; enabled={2}" -f $statusCount, $summaryCount, $enabledKits.Count)
    }

    foreach ($actionName in @("test", "verify")) {
        $resultName = "aicoding-kit.$actionName-smoke"
        $cmdResult = Invoke-Capture $shell @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $aicodingKitPath, $actionName, "-All", "-Profile", "Smoke", "-Json") $skipLifecycleEnv 20
        Add-Check "$resultName.exit" ($cmdResult.exitCode -eq 0) ("exit={0}" -f $cmdResult.exitCode) @{ stderr = $cmdResult.stderr.Trim(); timedOut = $cmdResult.timedOut }
        $cmdJson = if ($cmdResult.exitCode -eq 0) { Convert-CommandJson $cmdResult.stdout "$resultName.json" } else { $null }
        if ($cmdJson) {
            $cmdTotal = [int]$cmdJson.summary.total
            $cmdOk = [int]$cmdJson.summary.ok
            $cmdFailed = [int]$cmdJson.summary.failed
            $allSmoke = @($cmdJson.kits | Where-Object { $_.status -ne "smoke" }).Count -eq 0
            Add-Check "$resultName.count" (($cmdTotal -eq $enabledKits.Count) -and ($cmdOk -eq $enabledKits.Count) -and ($cmdFailed -eq 0) -and $allSmoke) ("smoke={0}/{1}; enabled={2}" -f $cmdOk, $cmdTotal, $enabledKits.Count) @{ allSmoke = $allSmoke }
        }
    }

    foreach ($actionName in @("skills", "verify-skills")) {
        $resultName = "v2.skill-routing.$actionName"
        $cmdResult = Invoke-Capture $shell @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $aicodingKitPath, $actionName, "-All", "-Json") $skipLifecycleEnv 20
        Add-Check "$resultName.exit" ($cmdResult.exitCode -eq 0) ("exit={0}" -f $cmdResult.exitCode) @{ stderr = $cmdResult.stderr.Trim(); timedOut = $cmdResult.timedOut }
        $cmdJson = if ($cmdResult.exitCode -eq 0) { Convert-CommandJson $cmdResult.stdout "$resultName.json" } else { $null }
        if ($cmdJson) {
            Add-Check "$resultName.count" (([int]$cmdJson.summary.total -eq $enabledKits.Count) -and ([int]$cmdJson.summary.failed -eq 0)) ("ok={0}/{1}; enabled={2}" -f $cmdJson.summary.ok, $cmdJson.summary.total, $enabledKits.Count) @{ declaredSkills = $cmdJson.summary.declaredSkills }
        }
    }
}

if (Test-Path -LiteralPath $aicodingSkillPath -PathType Leaf) {
    $skillSourcesResult = Invoke-Capture $shell @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $aicodingSkillPath, "sources", "-Json") @{} 20
    Add-Check "v2.skill-sources.exit" ($skillSourcesResult.exitCode -eq 0) ("exit={0}" -f $skillSourcesResult.exitCode) @{ stderr = $skillSourcesResult.stderr.Trim(); timedOut = $skillSourcesResult.timedOut }
    $skillSourcesJson = if ($skillSourcesResult.exitCode -eq 0) { Convert-CommandJson $skillSourcesResult.stdout "v2.skill-sources.json" } else { $null }
    if ($skillSourcesJson) { Add-Check "v2.skill-sources.ok" ([bool]$skillSourcesJson.ok) "skill source registry command returned ok" @{ sourceCount = @($skillSourcesJson.data.sources).Count } }
} else {
    Add-Check "v2.skill-sources.exists" $false "missing scripts/aicoding-skill.ps1"
}

foreach ($gate in @(
    @{ name = "v2.common.verify"; path = $verifyCommonCodePath },
    @{ name = "v2.hooks.verify"; path = $verifyHooksPath }
)) {
    if (Test-Path -LiteralPath $gate.path -PathType Leaf) {
        $gateResult = Invoke-Capture $shell @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $gate.path, "-Json") @{} 20
        Add-Check "$($gate.name).exit" ($gateResult.exitCode -eq 0) ("exit={0}" -f $gateResult.exitCode) @{ stderr = $gateResult.stderr.Trim(); timedOut = $gateResult.timedOut }
        $gateJson = if ($gateResult.exitCode -eq 0) { Convert-CommandJson $gateResult.stdout "$($gate.name).json" } else { $null }
        if ($gateJson) { Add-Check "$($gate.name).ok" ([bool]$gateJson.ok) "$($gate.name) smoke verification" }
    } else {
        Add-Check "$($gate.name).exists" $false ("missing {0}" -f ([System.IO.Path]::GetRelativePath($repo, $gate.path)))
    }
}

$result = [pscustomobject]@{
    schemaVersion = 1
    ok = ($errors.Count -eq 0)
    repoRoot = $repo
    profile = $Profile
    enabledKitCount = $enabledKits.Count
    checkedManifestCount = $manifestRecords.Count
    checks = $checks
    errors = @($errors)
}

if ($Json) {
    $result | ConvertTo-Json -Depth 30
} elseif ($result.ok) {
    Write-Host ("AiCoding kit lifecycle verification passed ({0}/{0} enabled kits)." -f $enabledKits.Count)
} else {
    $errors | ForEach-Object { Write-Error $_ }
}

if (-not $result.ok) { exit 1 }