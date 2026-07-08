[CmdletBinding(SupportsShouldProcess=$true)]
param(
    [Parameter(Position=0, Mandatory=$true)]
    [ValidateSet("sources", "add-source", "download", "install", "verify", "update", "remove", "create", "adopt", "list", "install-external", "status-external", "verify-external")]
    [string]$Action,

    [string]$Name = "",
    [string]$Url = "",
    [string]$Source = "",
    [string]$Skill = "",
    [ValidateSet("Draft", "RepoLocal", "Kit")]
    [string]$Scope = "Draft",
    [string]$Kit = "",
    [string]$Pin = "",
    [string]$RepoRoot = "",
    [ValidateSet("RepoLocal", "CodexUser", "Both")]
    [string]$Target = "RepoLocal",
    [int]$TimeoutSec = 180,
    [switch]$AllowWarn,
    [switch]$NoDeps,
    [switch]$PreferZip,
    [switch]$Resume,
    [switch]$Json,
    [switch]$Force
)

$ErrorActionPreference = "Stop"

$externalSkillActions = @("install-external", "status-external", "verify-external")
$skillAuditModule = Join-Path $PSScriptRoot "lib\AiCoding.SkillAudit.psm1"
if (Test-Path -LiteralPath $skillAuditModule -PathType Leaf) {
    Import-Module $skillAuditModule -Force
} elseif ($Action -in $externalSkillActions) {
    throw "Missing skill audit module: $skillAuditModule"
}
$repo = if ($RepoRoot) {
    (Resolve-Path -LiteralPath $RepoRoot).Path
} else {
    (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
}

function Resolve-RepoPath {
    param([Parameter(Mandatory=$true)][string]$Path)
    return (Join-Path $repo ($Path -replace "/", "\"))
}

function Test-AiCodingChildPath {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)][string]$Parent
    )

    $full = [System.IO.Path]::GetFullPath($Path)
    $root = [System.IO.Path]::GetFullPath($Parent).TrimEnd("\") + "\"
    return $full.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)
}

function New-AiCodingDirectory {
    param([Parameter(Mandatory=$true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
        [void](New-Item -ItemType Directory -Path $Path -Force)
    }
}

function Read-AiCodingJsonFile {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [object]$DefaultValue
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $DefaultValue }
    return (Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json)
}

function Write-AiCodingJsonFile {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$Value
    )

    New-AiCodingDirectory -Path (Split-Path -Parent $Path)
    $Value | ConvertTo-Json -Depth 40 | Set-Content -LiteralPath $Path -Encoding UTF8
}

function New-AiCodingSkillResult {
    param(
        [bool]$Ok,
        [string]$Status,
        [string]$Message,
        [object]$Data = $null,
        [string[]]$Errors = @()
    )

    return [pscustomobject]@{
        schemaVersion = 1
        action = $Action
        ok = $Ok
        status = $Status
        message = $Message
        data = $Data
        errors = @($Errors)
    }
}

function Complete-AiCodingSkillAction {
    param([Parameter(Mandatory=$true)]$Result)

    if ($Json) {
        $Result | ConvertTo-Json -Depth 60
    } elseif ($Result.ok) {
        Write-Host $Result.message
    } else {
        foreach ($err in @($Result.errors)) { Write-Error $err }
        if (@($Result.errors).Count -eq 0) { Write-Error $Result.message }
    }

    if (-not $Result.ok) { exit 1 }
}

function Assert-AiCodingSkillId {
    param([Parameter(Mandatory=$true)][string]$Value)
    if ($Value -notmatch "^[a-z0-9][a-z0-9-]*[a-z0-9]$") {
        throw "Skill id must be lowercase kebab-case: $Value"
    }
}

function Get-AiCodingSkillSourcesPath {
    return (Resolve-RepoPath "config/skill-sources.json")
}

function Get-AiCodingSkillSources {
    $default = [pscustomobject]@{ schemaVersion = 1; sources = @() }
    return Read-AiCodingJsonFile -Path (Get-AiCodingSkillSourcesPath) -DefaultValue $default
}

function Save-AiCodingSkillSources {
    param([Parameter(Mandatory=$true)]$Config)
    Write-AiCodingJsonFile -Path (Get-AiCodingSkillSourcesPath) -Value $Config
}

function Get-AiCodingSkillFrontmatter {
    param([Parameter(Mandatory=$true)][string]$Path)

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return [pscustomobject]@{ ok = $false; data = @{}; errors = @("missing SKILL.md") }
    }

    $lines = Get-Content -LiteralPath $Path -Encoding UTF8
    $errors = @()
    $data = @{}
    if ($lines.Count -lt 3 -or $lines[0].Trim() -ne "---") {
        return [pscustomobject]@{ ok = $false; data = $data; errors = @("missing frontmatter") }
    }

    $end = -1
    for ($i = 1; $i -lt $lines.Count; $i++) {
        if ($lines[$i].Trim() -eq "---") { $end = $i; break }
    }
    if ($end -lt 0) {
        return [pscustomobject]@{ ok = $false; data = $data; errors = @("unterminated frontmatter") }
    }

    for ($i = 1; $i -lt $end; $i++) {
        if ($lines[$i] -notmatch "^\s*([A-Za-z0-9_-]+)\s*:\s*(.*)\s*$") { continue }
        $data[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'")
    }

    foreach ($key in @("name", "description")) {
        if (-not $data.ContainsKey($key) -or [string]::IsNullOrWhiteSpace([string]$data[$key])) {
            $errors += "missing frontmatter.$key"
        }
    }

    return [pscustomobject]@{ ok = ($errors.Count -eq 0); data = $data; errors = $errors }
}

function Find-AiCodingSkillCandidate {
    param(
        [Parameter(Mandatory=$true)][string]$SkillId,
        [string]$PreferredSource = ""
    )

    $candidates = @()
    if ($PreferredSource) {
        $candidates += [pscustomobject]@{
            path = (Resolve-RepoPath ".aicoding/skill-cache/third-party/$PreferredSource/$SkillId")
            trust = "third-party"
            source = $PreferredSource
        }
    }

    $candidates += [pscustomobject]@{
        path = (Resolve-RepoPath ".aicoding/user-skills/$SkillId")
        trust = "user-created"
        source = "user-draft"
    }
    $candidates += [pscustomobject]@{
        path = (Resolve-RepoPath ".agents/skills/$SkillId")
        trust = "repo-local"
        source = "repo-local"
    }

    $cacheRoot = Resolve-RepoPath ".aicoding/skill-cache/third-party"
    if (Test-Path -LiteralPath $cacheRoot -PathType Container) {
        foreach ($sourceDir in @(Get-ChildItem -LiteralPath $cacheRoot -Directory -ErrorAction SilentlyContinue)) {
            $candidates += [pscustomobject]@{
                path = (Join-Path $sourceDir.FullName $SkillId)
                trust = "third-party"
                source = $sourceDir.Name
            }
        }
    }

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath (Join-Path $candidate.path "SKILL.md") -PathType Leaf) {
            return [pscustomobject]@{
                found = $true
                path = $candidate.path
                trust = $candidate.trust
                source = $candidate.source
                searched = @($candidates.path)
            }
        }
    }

    return [pscustomobject]@{
        found = $false
        path = ""
        trust = ""
        source = ""
        searched = @($candidates.path)
    }
}

function Test-AiCodingSkillContent {
    param(
        [Parameter(Mandatory=$true)][string]$SkillId,
        [string]$PreferredSource = ""
    )

    Assert-AiCodingSkillId -Value $SkillId
    $candidate = Find-AiCodingSkillCandidate -SkillId $SkillId -PreferredSource $PreferredSource
    if (-not $candidate.found) {
        return New-AiCodingSkillResult -Ok $false -Status "missing" -Message "Skill not found." -Data @{ searched = $candidate.searched } -Errors @("SKILL.md not found for $SkillId")
    }

    $skillMd = Join-Path $candidate.path "SKILL.md"
    $frontmatter = Get-AiCodingSkillFrontmatter -Path $skillMd
    $errors = @($frontmatter.errors)
    $text = Get-Content -LiteralPath $skillMd -Raw -Encoding UTF8

    $secretPatterns = @(
        "AKIA[0-9A-Z]{16}",
        "BEGIN (RSA|OPENSSH|EC) PRIVATE KEY",
        "(?i)api[_-]?key\s*[:=]\s*['""]?[A-Za-z0-9_\-]{20,}",
        "(?i)password\s*[:=]\s*['""]?\S{8,}"
    )
    foreach ($pattern in $secretPatterns) {
        if ($text -match $pattern) { $errors += "possible secret pattern: $pattern" }
    }

    $absolutePathPatterns = @(
        "[A-Za-z]:\\Users\\",
        "F:\\Study\\AI\\AiCoding",
        "C:\\Users\\24322"
    )
    foreach ($pattern in $absolutePathPatterns) {
        if ($text -match [regex]::Escape($pattern)) { $errors += "local absolute path reference: $pattern" }
    }

    $ok = ($errors.Count -eq 0)
    return New-AiCodingSkillResult -Ok $ok -Status ($(if ($ok) { "verified" } else { "failed" })) -Message "Skill verification completed." -Data @{
        skillId = $SkillId
        path = $candidate.path
        trust = $candidate.trust
        source = $candidate.source
        frontmatter = $frontmatter.data
    } -Errors $errors
}

function Get-AiCodingSkillFiles {
    param([Parameter(Mandatory=$true)][string]$Path)
    return @(Get-ChildItem -LiteralPath $Path -Recurse -File | ForEach-Object {
        [System.IO.Path]::GetRelativePath($Path, $_.FullName) -replace "\\", "/"
    })
}

function Get-AiCodingSourceRecord {
    param([Parameter(Mandatory=$true)][string]$SourceName)
    $config = Get-AiCodingSkillSources
    $sourceRecord = @($config.sources | Where-Object { $_.name -eq $SourceName }) | Select-Object -First 1
    if (-not $sourceRecord) { throw "Unknown skill source: $SourceName" }
    return $sourceRecord
}

function Copy-AiCodingDirectorySafe {
    param(
        [Parameter(Mandatory=$true)][string]$SourcePath,
        [Parameter(Mandatory=$true)][string]$DestinationPath,
        [Parameter(Mandatory=$true)][string]$AllowedRoot,
        [switch]$Replace
    )

    if (-not (Test-AiCodingChildPath -Path $DestinationPath -Parent $AllowedRoot)) {
        throw "Refusing to write outside allowed root: $DestinationPath"
    }
    if (Test-Path -LiteralPath $DestinationPath) {
        if (-not $Replace) { throw "Destination already exists: $DestinationPath" }
        Remove-Item -LiteralPath $DestinationPath -Recurse -Force
    }
    New-AiCodingDirectory -Path (Split-Path -Parent $DestinationPath)
    Copy-Item -LiteralPath $SourcePath -Destination $DestinationPath -Recurse
}

function Invoke-AiCodingSkillDownload {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [Parameter(Mandatory=$true)][string]$SkillId,
        [string]$PinnedRef = "",
        [switch]$Replace
    )

    Assert-AiCodingSkillId -Value $SkillId
    $sourceRecord = Get-AiCodingSourceRecord -SourceName $SourceName
    if ($sourceRecord.type -ne "git") { throw "Only git skill sources are supported in v2.0." }

    $zipState = Invoke-AiCodingExternalZipDownload -SourceName $SourceName -SkillId $SkillId -SourceRecord $sourceRecord -Paths $paths -PinnedRef $PinnedRef -Timeout $Timeout
    if ($zipState) { return $zipState }

    $git = Get-Command git -ErrorAction SilentlyContinue
    if (-not $git) { throw "git was not found on PATH." }

    $cacheRoot = Resolve-RepoPath ".aicoding/skill-cache/third-party/$SourceName"
    $destination = Join-Path $cacheRoot $SkillId
    $tmpRoot = Resolve-RepoPath ".aicoding/tmp/skill-download"
    New-AiCodingDirectory -Path $tmpRoot
    $clonePath = Join-Path $tmpRoot ("{0}-{1}" -f $SourceName, [guid]::NewGuid().ToString("N"))

    $cloneArgs = @("clone", "--depth", "1", [string]$sourceRecord.url, $clonePath)
    $clone = & $git.Source @cloneArgs 2>&1
    if ($LASTEXITCODE -ne 0) { throw "git clone failed: $clone" }

    $effectivePin = if ($PinnedRef) { $PinnedRef } elseif ($sourceRecord.pin) { [string]$sourceRecord.pin } else { "" }
    if ($effectivePin) {
        $checkout = & $git.Source -C $clonePath checkout $effectivePin 2>&1
        if ($LASTEXITCODE -ne 0) { throw "git checkout failed: $checkout" }
    }

    $skillPath = Join-Path $clonePath $SkillId
    if (-not (Test-Path -LiteralPath (Join-Path $skillPath "SKILL.md") -PathType Leaf)) {
        $skillPath = Join-Path (Join-Path $clonePath "skills") $SkillId
    }
    if (-not (Test-Path -LiteralPath (Join-Path $skillPath "SKILL.md") -PathType Leaf)) {
        $match = @(Get-ChildItem -LiteralPath $clonePath -Recurse -File -Filter SKILL.md | Where-Object {
            Split-Path -Leaf (Split-Path -Parent $_.FullName) -eq $SkillId
        } | Select-Object -First 1)
        if ($match.Count -gt 0) { $skillPath = Split-Path -Parent $match[0].FullName }
    }
    if (-not (Test-Path -LiteralPath (Join-Path $skillPath "SKILL.md") -PathType Leaf)) {
        throw "Could not find $SkillId/SKILL.md in source $SourceName."
    }

    Copy-AiCodingDirectorySafe -SourcePath $skillPath -DestinationPath $destination -AllowedRoot $cacheRoot -Replace:$Replace
    if (Test-AiCodingChildPath -Path $clonePath -Parent $tmpRoot) {
        Remove-Item -LiteralPath $clonePath -Recurse -Force
    }

    return [pscustomobject]@{
        skillId = $SkillId
        source = $SourceName
        url = [string]$sourceRecord.url
        pin = $effectivePin
        path = $destination
    }
}

function Get-AiCodingObjectValue {
    param(
        [object]$Object,
        [string]$Name,
        [object]$DefaultValue = $null
    )

    if ($null -eq $Object) { return $DefaultValue }
    if ($Object -is [hashtable] -and $Object.ContainsKey($Name)) { return $Object[$Name] }
    $property = $Object.PSObject.Properties[$Name]
    if ($property) { return $property.Value }
    return $DefaultValue
}

function Assert-AiCodingSourceName {
    param([Parameter(Mandatory=$true)][string]$Value)
    if ($Value -notmatch "^[A-Za-z0-9][A-Za-z0-9._-]*[A-Za-z0-9]$") {
        throw "Source name must be a safe name: $Value"
    }
}

function Resolve-AiCodingExternalSkillId {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [string]$SkillName = "",
        [object]$SourceRecord = $null
    )

    $resolved = if (-not [string]::IsNullOrWhiteSpace($SkillName)) { $SkillName } else { [string](Get-AiCodingObjectValue -Object $SourceRecord -Name "skill" -DefaultValue $SourceName) }
    Assert-AiCodingSkillId -Value $resolved
    return $resolved
}

function Get-AiCodingExternalSkillPaths {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [Parameter(Mandatory=$true)][string]$SkillId
    )

    Assert-AiCodingSourceName -Value $SourceName
    Assert-AiCodingSkillId -Value $SkillId
    $cacheRoot = Resolve-RepoPath ".aicoding/skill-cache/external/$SourceName"
    $codexRoot = Join-Path $HOME ".codex\skills"
    return [pscustomobject]@{
        cacheRoot = $cacheRoot
        cacheRepo = Join-Path $cacheRoot "repo"
        installState = Join-Path $cacheRoot "install-state.json"
        installLog = Join-Path $cacheRoot "install-log.ndjson"
        auditReport = Join-Path $cacheRoot "audit-report.json"
        auditLog = Join-Path $cacheRoot "install-log.ndjson"
        downloadState = Join-Path $cacheRoot "install-state.json"
        repoLocalSkill = Resolve-RepoPath ".agents/skills/$SkillId"
        codexSkill = Join-Path $codexRoot $SkillId
        codexRoot = $codexRoot
    }
}

function ConvertTo-AiCodingRelativePath {
    param(
        [Parameter(Mandatory=$true)][string]$Root,
        [Parameter(Mandatory=$true)][string]$Path
    )
    return ([System.IO.Path]::GetRelativePath($Root, $Path) -replace "\\", "/")
}

function Get-AiCodingExternalSkillLocation {
    param(
        [Parameter(Mandatory=$true)]$SourceRecord,
        [Parameter(Mandatory=$true)][string]$SkillId,
        [Parameter(Mandatory=$true)][string]$RepoPath
    )

    $configured = [string](Get-AiCodingObjectValue -Object $SourceRecord -Name "skillPath" -DefaultValue "")
    $candidates = @()
    if (-not [string]::IsNullOrWhiteSpace($configured)) { $candidates += $configured }
    $candidates += @($SkillId, "skills/$SkillId", ".")
    foreach ($candidate in @($candidates | Select-Object -Unique)) {
        $full = Join-Path $RepoPath ($candidate -replace "/", "\")
        if (Test-Path -LiteralPath (Join-Path $full "SKILL.md") -PathType Leaf) {
            return [pscustomobject]@{
                relativePath = ($candidate -replace "\\", "/")
                path = $full
                skillMd = Join-Path $full "SKILL.md"
            }
        }
    }
    throw "Could not find $SkillId/SKILL.md in external repo. Configure skillPath in config/skill-sources.json."
}

function Read-AiCodingLogTail {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [int]$Count = 20
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return @() }
    return @(Get-Content -LiteralPath $Path -Tail $Count -Encoding UTF8)
}

function Add-AiCodingExternalInstallLog {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)][string]$Event,
        [object]$Data = $null
    )

    New-AiCodingDirectory -Path (Split-Path -Parent $Path)
    $entry = [ordered]@{
        schemaVersion = 1
        createdAt = (Get-Date).ToUniversalTime().ToString("o")
        event = $Event
        data = $Data
    }
    ($entry | ConvertTo-Json -Depth 20 -Compress) | Add-Content -LiteralPath $Path -Encoding UTF8
}

function Write-AiCodingExternalDownloadState {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$State
    )
    Write-AiCodingJsonFile -Path $Path -Value $State
}

function Get-AiCodingGitHubZipRefs {
    param(
        [Parameter(Mandatory=$true)]$SourceRecord,
        [string]$PinnedRef = ""
    )

    if ([string]$SourceRecord.url -notmatch '^https://github\.com/([^/]+)/([^/.]+)(\.git)?$') { return @() }
    $owner = $Matches[1]
    $repoName = $Matches[2]
    $refs = @()
    if ($PinnedRef) {
        $refs += $PinnedRef
    } else {
        $branch = [string](Get-AiCodingObjectValue -Object $SourceRecord -Name "branch" -DefaultValue "")
        if ($branch) { $refs += $branch }
        $refs += @("main", "master")
    }
    return @($refs | Select-Object -Unique | ForEach-Object {
        [pscustomobject]@{ ref = $_; url = "https://github.com/$owner/$repoName/archive/$_.zip" }
    })
}

function Expand-AiCodingExternalZip {
    param(
        [Parameter(Mandatory=$true)][string]$ZipPath,
        [Parameter(Mandatory=$true)][string]$DestinationPath,
        [Parameter(Mandatory=$true)][string]$AllowedRoot
    )

    if (-not (Test-AiCodingChildPath -Path $DestinationPath -Parent $AllowedRoot)) { throw "Refusing to write outside external cache: $DestinationPath" }
    $extractRoot = Join-Path (Split-Path -Parent $ZipPath) "extract"
    if (Test-Path -LiteralPath $extractRoot) { Remove-Item -LiteralPath $extractRoot -Recurse -Force }
    New-AiCodingDirectory -Path $extractRoot
    Expand-Archive -LiteralPath $ZipPath -DestinationPath $extractRoot -Force
    $repoRoot = @(Get-ChildItem -LiteralPath $extractRoot -Directory | Select-Object -First 1)
    if ($repoRoot.Count -eq 0) { throw "Downloaded zip did not contain a repository directory." }
    if (Test-Path -LiteralPath $DestinationPath) { Remove-Item -LiteralPath $DestinationPath -Recurse -Force }
    New-AiCodingDirectory -Path (Split-Path -Parent $DestinationPath)
    Move-Item -LiteralPath $repoRoot[0].FullName -Destination $DestinationPath
}

function Invoke-AiCodingExternalZipDownload {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [Parameter(Mandatory=$true)][string]$SkillId,
        [Parameter(Mandatory=$true)]$SourceRecord,
        [Parameter(Mandatory=$true)]$Paths,
        [string]$PinnedRef = "",
        [int]$Timeout = 180
    )

    $zipRefs = @(Get-AiCodingGitHubZipRefs -SourceRecord $SourceRecord -PinnedRef $PinnedRef)
    if ($zipRefs.Count -eq 0) { return $null }
    $startedAt = Get-Date
    $tmpRoot = Resolve-RepoPath ".aicoding/tmp/external-download"
    New-AiCodingDirectory -Path $tmpRoot
    $workRoot = Join-Path $tmpRoot ("{0}-{1}" -f $SourceName, [guid]::NewGuid().ToString("N"))
    New-AiCodingDirectory -Path $workRoot
    try {
        foreach ($zipRef in $zipRefs) {
            $zipPath = Join-Path $workRoot ("{0}.zip" -f $zipRef.ref)
            $lastCommand = "Invoke-WebRequest zip:$($zipRef.ref)"
            try {
                Add-AiCodingExternalInstallLog -Path $Paths.installLog -Event "download.zip-start" -Data @{ ref = $zipRef.ref; url = $zipRef.url }
                Invoke-WebRequest -Uri $zipRef.url -OutFile $zipPath -TimeoutSec $Timeout -ErrorAction Stop | Out-Null
                Expand-AiCodingExternalZip -ZipPath $zipPath -DestinationPath $Paths.cacheRepo -AllowedRoot $Paths.cacheRoot
                $finishedAt = Get-Date
                $state = [pscustomobject]@{
                    schemaVersion = 1
                    source = $SourceName
                    skill = $SkillId
                    stage = "downloaded"
                    trust = "pending"
                    cacheRepo = $Paths.cacheRepo
                    auditReport = $Paths.auditReport
                    elapsedSec = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)
                    lastCommand = $lastCommand
                    lastExitCode = 0
                    updatedAt = $finishedAt.ToUniversalTime().ToString("o")
                }
                Write-AiCodingExternalDownloadState -Path $Paths.downloadState -State $state
                Add-AiCodingExternalInstallLog -Path $Paths.installLog -Event "download.zip-finish" -Data @{ ref = $zipRef.ref; elapsedSec = $state.elapsedSec }
                return $state
            } catch {
                Add-AiCodingExternalInstallLog -Path $Paths.installLog -Event "download.zip-failed" -Data @{ ref = $zipRef.ref; error = $_.Exception.Message }
            }
        }
        return $null
    } finally {
        if ((Test-Path -LiteralPath $workRoot) -and (Test-AiCodingChildPath -Path $workRoot -Parent $tmpRoot)) {
            Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

function Write-AiCodingExternalInstallState {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$State
    )
    Write-AiCodingJsonFile -Path $Path -Value $State
}

function Invoke-AiCodingExternalDownload {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [Parameter(Mandatory=$true)][string]$SkillId,
        [string]$PinnedRef = "",
        [int]$Timeout = 180,
        [switch]$PreferZipMode,
        [switch]$ResumeExisting,
        [switch]$Replace
    )

    $paths = Get-AiCodingExternalSkillPaths -SourceName $SourceName -SkillId $SkillId
    $sourceRecord = Get-AiCodingSourceRecord -SourceName $SourceName
    if ([string](Get-AiCodingObjectValue -Object $sourceRecord -Name "type" -DefaultValue "git") -ne "git") { throw "Only git external skill sources are supported." }
    $startedAt = Get-Date

    if ((Test-Path -LiteralPath $paths.cacheRoot -PathType Container) -and $Replace) {
        $allowedRoot = Resolve-RepoPath ".aicoding/skill-cache/external"
        if (-not (Test-AiCodingChildPath -Path $paths.cacheRoot -Parent $allowedRoot)) { throw "Refusing to clean external cache outside allowed root: $($paths.cacheRoot)" }
        Remove-Item -LiteralPath $paths.cacheRoot -Recurse -Force
    }

    if ((Test-Path -LiteralPath $paths.cacheRepo -PathType Container) -and -not $Replace) {
        $finishedAt = Get-Date
        $state = [pscustomobject]@{
            schemaVersion = 1
            source = $SourceName
            skill = $SkillId
            stage = $(if ($ResumeExisting) { "download_resumed" } else { "download_reused" })
            repoPath = $paths.cacheRepo
            mode = "git"
            preferZipRequested = [bool]$PreferZipMode
            startedAt = $startedAt.ToUniversalTime().ToString("o")
            finishedAt = $finishedAt.ToUniversalTime().ToString("o")
            elapsedSec = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)
            lastCommand = "reuse cache repo"
            lastExitCode = 0
        }
        Write-AiCodingExternalDownloadState -Path $paths.downloadState -State $state
        return $state
    }

    $zipState = Invoke-AiCodingExternalZipDownload -SourceName $SourceName -SkillId $SkillId -SourceRecord $sourceRecord -Paths $paths -PinnedRef $PinnedRef -Timeout $Timeout
    if ($zipState) { return $zipState }

    $git = Get-Command git -ErrorAction SilentlyContinue
    if (-not $git) { throw "git was not found on PATH." }
    New-AiCodingDirectory -Path $paths.cacheRoot
    $cloneArgs = @("clone", "--depth", "1", "--filter=blob:none", [string]$sourceRecord.url, $paths.cacheRepo)
    $cloneResult = Invoke-AiCodingObservedProcess -Command $git.Source -Args $cloneArgs -WorkingDirectory $repo -TimeoutSec $Timeout
    if ($cloneResult.exitCode -ne 0) {
        $failedAt = Get-Date
        Write-AiCodingExternalDownloadState -Path $paths.downloadState -State ([pscustomobject]@{
            schemaVersion = 1; source = $SourceName; skill = $SkillId; stage = "download_failed"; repoPath = $paths.cacheRepo; mode = "git"; preferZipRequested = [bool]$PreferZipMode; startedAt = $startedAt.ToUniversalTime().ToString("o"); finishedAt = $failedAt.ToUniversalTime().ToString("o"); elapsedSec = [Math]::Round(($failedAt - $startedAt).TotalSeconds, 3); lastCommand = "git clone --depth 1 --filter=blob:none"; lastExitCode = $cloneResult.exitCode; stderr = $cloneResult.stderr
        })
        throw "git clone failed: $($cloneResult.stderr.Trim())"
    }

    $effectivePin = if ($PinnedRef) { $PinnedRef } elseif (Get-AiCodingObjectValue -Object $sourceRecord -Name "pin" -DefaultValue "") { [string](Get-AiCodingObjectValue -Object $sourceRecord -Name "pin" -DefaultValue "") } else { "" }
    if ($effectivePin) {
        $checkoutResult = Invoke-AiCodingObservedProcess -Command $git.Source -Args @("-C", $paths.cacheRepo, "checkout", $effectivePin) -WorkingDirectory $repo -TimeoutSec $Timeout
        if ($checkoutResult.exitCode -ne 0) { throw "git checkout failed: $($checkoutResult.stderr.Trim())" }
    }
    $commitResult = Invoke-AiCodingObservedProcess -Command $git.Source -Args @("-C", $paths.cacheRepo, "rev-parse", "HEAD") -WorkingDirectory $repo -TimeoutSec 30
    $finished = Get-Date
    $state = [pscustomobject]@{
        schemaVersion = 1
        source = $SourceName
        skill = $SkillId
        stage = "downloaded"
        repoPath = $paths.cacheRepo
        repoUrl = [string]$sourceRecord.url
        pin = $effectivePin
        commit = $commitResult.stdout.Trim()
        mode = "git"
        preferZipRequested = [bool]$PreferZipMode
        startedAt = $startedAt.ToUniversalTime().ToString("o")
        finishedAt = $finished.ToUniversalTime().ToString("o")
        elapsedSec = [Math]::Round(($finished - $startedAt).TotalSeconds, 3)
        lastCommand = "git clone --depth 1 --filter=blob:none"
        lastExitCode = $cloneResult.exitCode
    }
    Write-AiCodingExternalDownloadState -Path $paths.downloadState -State $state
    return $state
}

function Test-AiCodingExternalAuditGate {
    param(
        [Parameter(Mandatory=$true)]$AuditReport,
        [switch]$AllowWarning,
        [object]$InstallState = $null
    )

    $auditStatus = [string]$AuditReport.status
    $accepted = [bool](Get-AiCodingObjectValue -Object $InstallState -Name "auditAccepted" -DefaultValue $false)
    if ($auditStatus -eq "pass") { return [pscustomobject]@{ ok = $true; status = "audit_pass"; message = "Skill audit passed." } }
    if ($auditStatus -eq "warn") {
        if ($AllowWarning -or $accepted) { return [pscustomobject]@{ ok = $true; status = "audit_warn_allowed"; message = "Skill audit warning accepted." } }
        return [pscustomobject]@{ ok = $false; status = "audit_warn"; message = "audit returned warn; re-run with -AllowWarn to install" }
    }
    return [pscustomobject]@{ ok = $false; status = "audit_$auditStatus"; message = "Skill audit status is $auditStatus; install stopped." }
}

function Install-AiCodingExternalDependencies {
    param(
        [Parameter(Mandatory=$true)]$SourceRecord,
        [Parameter(Mandatory=$true)][string]$CacheRepo,
        [int]$Timeout = 180,
        [switch]$SkipDeps
    )

    $pythonConfig = Get-AiCodingObjectValue -Object $SourceRecord -Name "python" -DefaultValue $null
    $pythonInstall = [bool](Get-AiCodingObjectValue -Object $pythonConfig -Name "install" -DefaultValue $false)
    if (-not $pythonInstall) { return [pscustomobject]@{ skipped = $true; reason = "python.install is not enabled" } }
    if ($SkipDeps) { return [pscustomobject]@{ skipped = $true; reason = "-NoDeps was provided" } }

    $dependencyFile = [string](Get-AiCodingObjectValue -Object $SourceRecord -Name "dependencyFile" -DefaultValue "")
    if ([string]::IsNullOrWhiteSpace($dependencyFile)) { return [pscustomobject]@{ skipped = $true; reason = "dependencyFile is not configured" } }
    $dependencyPath = Join-Path $CacheRepo ($dependencyFile -replace "/", "\")
    if (-not (Test-Path -LiteralPath $dependencyPath -PathType Leaf)) { throw "Configured dependencyFile does not exist: $dependencyFile" }

    $venvRel = [string](Get-AiCodingObjectValue -Object $pythonConfig -Name "venv" -DefaultValue ".venv")
    if ([string]::IsNullOrWhiteSpace($venvRel)) { $venvRel = ".venv" }
    $venvPath = Join-Path $CacheRepo ($venvRel -replace "/", "\")
    $venvPython = Join-Path $venvPath "Scripts\python.exe"
    if (-not (Test-Path -LiteralPath $venvPython -PathType Leaf)) {
        $python = Get-Command python -ErrorAction SilentlyContinue
        if (-not $python) { throw "python was not found on PATH for dependency installation." }
        $venvResult = Invoke-AiCodingObservedProcess -Command $python.Source -Args @("-m", "venv", $venvPath) -WorkingDirectory $CacheRepo -TimeoutSec $Timeout
        if ($venvResult.exitCode -ne 0) { throw "python venv creation failed: $($venvResult.stderr.Trim())" }
    }
    $pipResult = Invoke-AiCodingObservedProcess -Command $venvPython -Args @("-m", "pip", "install", "-r", $dependencyPath) -WorkingDirectory $CacheRepo -TimeoutSec $Timeout
    if ($pipResult.exitCode -ne 0) { throw "dependency installation failed: $($pipResult.stderr.Trim())" }
    return [pscustomobject]@{ skipped = $false; dependencyFile = $dependencyFile; venv = $venvPath; python = $venvPython; elapsedSec = $pipResult.elapsedSec }
}

function Install-AiCodingExternalSkillTargets {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [Parameter(Mandatory=$true)][string]$SkillId,
        [Parameter(Mandatory=$true)][ValidateSet("RepoLocal", "CodexUser", "Both")][string]$InstallTarget,
        [int]$Timeout = 180,
        [switch]$Replace,
        [switch]$SkipDeps
    )

    $paths = Get-AiCodingExternalSkillPaths -SourceName $SourceName -SkillId $SkillId
    $sourceRecord = Get-AiCodingSourceRecord -SourceName $SourceName
    $skillLocation = Get-AiCodingExternalSkillLocation -SourceRecord $sourceRecord -SkillId $SkillId -RepoPath $paths.cacheRepo
    $installed = [ordered]@{}
    if ($InstallTarget -in @("RepoLocal", "Both")) {
        if ((Test-Path -LiteralPath (Join-Path $paths.repoLocalSkill "SKILL.md") -PathType Leaf) -and -not $Replace) {
            $installed.repoLocal = [pscustomobject]@{ path = $paths.repoLocalSkill; reused = $true }
        } else {
            Copy-AiCodingDirectorySafe -SourcePath $skillLocation.path -DestinationPath $paths.repoLocalSkill -AllowedRoot (Resolve-RepoPath ".agents/skills") -Replace:$Replace
            $installed.repoLocal = [pscustomobject]@{ path = $paths.repoLocalSkill; reused = $false }
        }
    }
    if ($InstallTarget -in @("CodexUser", "Both")) {
        if ([string]::IsNullOrWhiteSpace($HOME)) { throw "HOME is not set; cannot resolve CodexUser skill root." }
        if ((Test-Path -LiteralPath (Join-Path $paths.codexSkill "SKILL.md") -PathType Leaf) -and -not $Replace) {
            $installed.codexUser = [pscustomobject]@{ path = $paths.codexSkill; reused = $true }
        } else {
            Copy-AiCodingDirectorySafe -SourcePath $skillLocation.path -DestinationPath $paths.codexSkill -AllowedRoot $paths.codexRoot -Replace:$Replace
            $installed.codexUser = [pscustomobject]@{ path = $paths.codexSkill; reused = $false }
        }
    }
    $dependency = Install-AiCodingExternalDependencies -SourceRecord $sourceRecord -CacheRepo $paths.cacheRepo -Timeout $Timeout -SkipDeps:$SkipDeps
    $state = [pscustomobject]@{
        schemaVersion = 1
        source = $SourceName
        skill = $SkillId
        target = $InstallTarget
        stage = "installed"
        cacheRepo = $paths.cacheRepo
        skillPath = $skillLocation.relativePath
        targets = [pscustomobject]$installed
        dependency = $dependency
        installedAt = (Get-Date).ToUniversalTime().ToString("o")
        installedBy = $env:USERNAME
    }
    Write-AiCodingJsonFile -Path $paths.installState -Value $state
    Add-AiCodingExternalInstallLog -Path $paths.installLog -Event "install.finish" -Data @{ target = $InstallTarget; paths = $installed; dependency = $dependency }
    return $state
}

function Get-AiCodingExternalStatus {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [Parameter(Mandatory=$true)][string]$SkillId
    )

    $paths = Get-AiCodingExternalSkillPaths -SourceName $SourceName -SkillId $SkillId
    $downloadState = Read-AiCodingJsonFile -Path $paths.downloadState -DefaultValue $null
    $installState = Read-AiCodingJsonFile -Path $paths.installState -DefaultValue $null
    $auditReport = Read-AiCodingSkillAuditReport -Path $paths.auditReport
    return [pscustomobject]@{
        source = $SourceName
        skill = $SkillId
        stage = [string](Get-AiCodingObjectValue -Object $installState -Name "stage" -DefaultValue "missing")
        trust = [string](Get-AiCodingObjectValue -Object $installState -Name "trust" -DefaultValue "pending")
        downloadStage = [string](Get-AiCodingObjectValue -Object $downloadState -Name "stage" -DefaultValue "missing")
        installStage = [string](Get-AiCodingObjectValue -Object $installState -Name "stage" -DefaultValue "not-installed")
        auditStatus = [string](Get-AiCodingObjectValue -Object $auditReport -Name "status" -DefaultValue "missing")
        auditScore = Get-AiCodingObjectValue -Object $auditReport -Name "score" -DefaultValue $null
        elapsedSec = Get-AiCodingObjectValue -Object $downloadState -Name "elapsedSec" -DefaultValue $null
        lastCommand = [string](Get-AiCodingObjectValue -Object $downloadState -Name "lastCommand" -DefaultValue "")
        lastExitCode = Get-AiCodingObjectValue -Object $downloadState -Name "lastExitCode" -DefaultValue $null
        paths = [pscustomobject]@{
            cache = $paths.cacheRoot
            cacheRepo = $paths.cacheRepo
            codexSkill = $paths.codexSkill
            repoLocalSkill = $paths.repoLocalSkill
            auditReport = $paths.auditReport
            installState = $paths.installState
            installLog = $paths.installLog
        }
        logTail = @(Read-AiCodingLogTail -Path $paths.installLog -Count 20)
    }
}


function Test-AiCodingExternalInstall {
    param(
        [Parameter(Mandatory=$true)][string]$SourceName,
        [Parameter(Mandatory=$true)][string]$SkillId,
        [Parameter(Mandatory=$true)][ValidateSet("RepoLocal", "CodexUser", "Both")][string]$InstallTarget,
        [switch]$AllowWarning,
        [switch]$SkipDeps
    )

    function New-ExternalVerifyCheck {
        param(
            [Parameter(Mandatory=$true)][string]$Name,
            [Parameter(Mandatory=$true)]$Ok,
            [Parameter(Mandatory=$true)][string]$Message,
            [object]$Data = $null
        )
        return [pscustomobject]@{ name = $Name; ok = [bool]$Ok; message = $Message; data = $Data }
    }

    $paths = Get-AiCodingExternalSkillPaths -SourceName $SourceName -SkillId $SkillId
    $sourceRecord = Get-AiCodingSourceRecord -SourceName $SourceName
    $checks = @()

    $state = Read-AiCodingJsonFile -Path $paths.installState -DefaultValue $null
    $report = Read-AiCodingSkillAuditReport -Path $paths.auditReport
    $checks += New-ExternalVerifyCheck "audit.report.exists" ($null -ne $report) "audit-report.json exists" @{ path = $paths.auditReport }
    if ($report) {
        $gate = Test-AiCodingExternalAuditGate -AuditReport $report -AllowWarning:$AllowWarning -InstallState $state
        $checks += New-ExternalVerifyCheck "audit.status.allowed" ([bool]$gate.ok) $gate.message @{ status = $report.status; score = $report.score }
    }
    $checks += New-ExternalVerifyCheck "cache.repo.exists" (Test-Path -LiteralPath $paths.cacheRepo -PathType Container) "external cache repo exists" @{ path = $paths.cacheRepo }
    if (Test-Path -LiteralPath $paths.cacheRepo -PathType Container) {
        try {
            $skillLocation = Get-AiCodingExternalSkillLocation -SourceRecord $sourceRecord -SkillId $SkillId -RepoPath $paths.cacheRepo
            $checks += New-ExternalVerifyCheck "cache.skill.exists" (Test-Path -LiteralPath $skillLocation.skillMd -PathType Leaf) "cache repo skillPath/SKILL.md exists" @{ skillPath = $skillLocation.relativePath }
        } catch {
            $checks += New-ExternalVerifyCheck "cache.skill.exists" $false $_.Exception.Message
        }
        $dependencyFile = [string](Get-AiCodingObjectValue -Object $sourceRecord -Name "dependencyFile" -DefaultValue "")
        if ($dependencyFile) {
            $depPath = Join-Path $paths.cacheRepo ($dependencyFile -replace "/", "\")
            $checks += New-ExternalVerifyCheck "dependency.file.exists" (Test-Path -LiteralPath $depPath -PathType Leaf) "configured dependencyFile exists" @{ path = $depPath }
        }
        $pythonConfig = Get-AiCodingObjectValue -Object $sourceRecord -Name "python" -DefaultValue $null
        if ([bool](Get-AiCodingObjectValue -Object $pythonConfig -Name "install" -DefaultValue $false) -and -not $SkipDeps) {
            $venvRel = [string](Get-AiCodingObjectValue -Object $pythonConfig -Name "venv" -DefaultValue ".venv")
            $venvPython = Join-Path (Join-Path $paths.cacheRepo ($venvRel -replace "/", "\")) "Scripts\python.exe"
            $checks += New-ExternalVerifyCheck "python.venv.exists" (Test-Path -LiteralPath $venvPython -PathType Leaf) "python.install=true venv python exists" @{ path = $venvPython }
        }
    }
    if ($InstallTarget -in @("RepoLocal", "Both")) {
        $repoSkillMd = Join-Path $paths.repoLocalSkill "SKILL.md"
        $checks += New-ExternalVerifyCheck "target.repoLocal.exists" (Test-Path -LiteralPath $repoSkillMd -PathType Leaf) "repo-local target SKILL.md exists" @{ path = $repoSkillMd }
    }
    if ($InstallTarget -in @("CodexUser", "Both")) {
        $codexSkillMd = Join-Path $paths.codexSkill "SKILL.md"
        $checks += New-ExternalVerifyCheck "target.codexUser.exists" (Test-Path -LiteralPath $codexSkillMd -PathType Leaf) "CodexUser target SKILL.md exists" @{ path = $codexSkillMd }
    }

    $checkList = @($checks)
    $failed = @($checkList | Where-Object { -not [bool]$_.ok })
    $errors = [string[]]@($failed | ForEach-Object { "{0}: {1}" -f $_.name, $_.message })
    return [pscustomobject]@{
        ok = [bool]($failed.Count -eq 0)
        checks = $checkList
        errors = $errors
    }
}
try {
    switch ($Action) {
        "sources" {
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "ok" -Message "Skill sources loaded." -Data (Get-AiCodingSkillSources))
        }

        "add-source" {
            if ([string]::IsNullOrWhiteSpace($Name)) { throw "-Name is required." }
            if ([string]::IsNullOrWhiteSpace($Url)) { throw "-Url is required." }
            $config = Get-AiCodingSkillSources
            $sources = @($config.sources | Where-Object { $_.name -ne $Name })
            $sources += [pscustomobject]@{
                name = $Name
                type = "git"
                url = $Url
                trust = "third-party"
                updatePolicy = "manual"
                pin = $Pin
            }
            $config.sources = @($sources | Sort-Object name)
            Save-AiCodingSkillSources -Config $config
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "updated" -Message "Skill source registered." -Data @{ name = $Name; url = $Url; pin = $Pin })
        }

        "list" {
            $roots = @(
                @{ name = "user-draft"; path = (Resolve-RepoPath ".aicoding/user-skills") },
                @{ name = "repo-local"; path = (Resolve-RepoPath ".agents/skills") },
                @{ name = "third-party-cache"; path = (Resolve-RepoPath ".aicoding/skill-cache/third-party") },
                @{ name = "external-cache"; path = (Resolve-RepoPath ".aicoding/skill-cache/external") }
            )
            $skills = @()
            foreach ($rootInfo in $roots) {
                if (-not (Test-Path -LiteralPath $rootInfo.path -PathType Container)) { continue }
                foreach ($skillFile in @(Get-ChildItem -LiteralPath $rootInfo.path -Recurse -File -Filter SKILL.md -ErrorAction SilentlyContinue)) {
                    $skills += [pscustomobject]@{
                        scope = $rootInfo.name
                        id = Split-Path -Leaf (Split-Path -Parent $skillFile.FullName)
                        path = [System.IO.Path]::GetRelativePath($repo, (Split-Path -Parent $skillFile.FullName)) -replace "\\", "/"
                    }
                }
            }
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "ok" -Message "Skills listed." -Data @{ sources = (Get-AiCodingSkillSources).sources; skills = $skills })
        }

        "create" {
            if ([string]::IsNullOrWhiteSpace($Skill)) { throw "-Skill is required." }
            Assert-AiCodingSkillId -Value $Skill
            if ($Scope -eq "Kit" -and [string]::IsNullOrWhiteSpace($Kit)) { throw "-Kit is required when -Scope Kit is used." }
            $draftRoot = Resolve-RepoPath ".aicoding/user-skills/$Skill"
            if (Test-Path -LiteralPath $draftRoot -and -not $Force) { throw "Draft skill already exists: $draftRoot" }
            New-AiCodingDirectory -Path $draftRoot
            $skillMd = Join-Path $draftRoot "SKILL.md"
            if ((Test-Path -LiteralPath $skillMd) -and -not $Force) { throw "SKILL.md already exists: $skillMd" }
            $content = @"
---
name: $Skill
description: Draft AiCoding skill. Replace this description before enabling.
---

# $Skill

## When To Use

Use this skill when the current repository needs the specific workflow described here.

## When Not To Use

Do not use this skill for unrelated repository maintenance or unsafe system changes.

## Verification

Run:

````powershell
pwsh scripts/aicoding-skill.ps1 verify -Skill $Skill -Json
````

## Examples

- Add a focused example before installing or adopting this skill.
"@
            Set-Content -LiteralPath $skillMd -Value $content -Encoding UTF8
            Write-AiCodingJsonFile -Path (Join-Path $draftRoot "skill.json") -Value ([pscustomobject]@{
                schemaVersion = 1
                skillId = $Skill
                scope = $Scope
                kit = $Kit
                createdAt = (Get-Date).ToUniversalTime().ToString("o")
                trust = "user-created"
            })
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "created" -Message "Draft skill created." -Data @{ skillId = $Skill; path = $draftRoot; scope = $Scope; kit = $Kit })
        }

        "verify" {
            if ([string]::IsNullOrWhiteSpace($Skill)) { throw "-Skill is required." }
            Complete-AiCodingSkillAction (Test-AiCodingSkillContent -SkillId $Skill -PreferredSource $Source)
        }

        "download" {
            if ([string]::IsNullOrWhiteSpace($Source)) { throw "-Source is required." }
            if ([string]::IsNullOrWhiteSpace($Skill)) { throw "-Skill is required." }
            $downloaded = Invoke-AiCodingSkillDownload -SourceName $Source -SkillId $Skill -PinnedRef $Pin -Replace:$Force
            $verified = Test-AiCodingSkillContent -SkillId $Skill -PreferredSource $Source
            $ok = $verified.ok
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $ok -Status ($(if ($ok) { "downloaded" } else { "quarantine" })) -Message "Skill downloaded into isolated cache." -Data @{ download = $downloaded; verification = $verified.data } -Errors @($verified.errors))
        }

        "status-external" {
            if ([string]::IsNullOrWhiteSpace($Source)) { throw "-Source is required." }
            $sourceRecord = Get-AiCodingSourceRecord -SourceName $Source
            $skillId = Resolve-AiCodingExternalSkillId -SourceName $Source -SkillName $Skill -SourceRecord $sourceRecord
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "ok" -Message "External skill status loaded." -Data (Get-AiCodingExternalStatus -SourceName $Source -SkillId $skillId))
        }

        "verify-external" {
            if ([string]::IsNullOrWhiteSpace($Source)) { throw "-Source is required." }
            $sourceRecord = Get-AiCodingSourceRecord -SourceName $Source
            $skillId = Resolve-AiCodingExternalSkillId -SourceName $Source -SkillName $Skill -SourceRecord $sourceRecord
            $verified = Test-AiCodingExternalInstall -SourceName $Source -SkillId $skillId -InstallTarget $Target -AllowWarning:$AllowWarn -SkipDeps:$NoDeps
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $verified.ok -Status ($(if ($verified.ok) { "verified" } else { "failed" })) -Message "External skill verification completed." -Data @{ checks = $verified.checks; status = (Get-AiCodingExternalStatus -SourceName $Source -SkillId $skillId) } -Errors $verified.errors)
        }

        "install-external" {
            if ([string]::IsNullOrWhiteSpace($Source)) { throw "-Source is required." }
            $sourceRecord = Get-AiCodingSourceRecord -SourceName $Source
            $skillId = Resolve-AiCodingExternalSkillId -SourceName $Source -SkillName $Skill -SourceRecord $sourceRecord
            $paths = Get-AiCodingExternalSkillPaths -SourceName $Source -SkillId $skillId
            $startedAt = Get-Date
            try {
                $downloadState = Invoke-AiCodingExternalDownload -SourceName $Source -SkillId $skillId -PinnedRef $Pin -Timeout $TimeoutSec -PreferZipMode:$PreferZip -ResumeExisting:$Resume -Replace:$Force
                $skillLocation = Get-AiCodingExternalSkillLocation -SourceRecord $sourceRecord -SkillId $skillId -RepoPath $paths.cacheRepo
                $report = Invoke-AiCodingSkillAudit -Source $Source -Skill $skillId -RepoPath $paths.cacheRepo -SkillPath $skillLocation.path -ReportPath $paths.auditReport -LogPath $paths.installLog
                $gate = Test-AiCodingExternalAuditGate -AuditReport $report -AllowWarning:$AllowWarn
                if (-not $gate.ok) {
                    $blockedState = [pscustomobject]@{
                        schemaVersion = 1
                        source = $Source
                        skill = $skillId
                        stage = $gate.status
                        trust = "pending"
                        auditAccepted = $false
                        cacheRepo = $paths.cacheRepo
                        auditReport = $paths.auditReport
                        audit = @{ status = $report.status; score = $report.score }
                        elapsedSec = $downloadState.elapsedSec
                        lastCommand = $downloadState.lastCommand
                        lastExitCode = $downloadState.lastExitCode
                        message = $gate.message
                        updatedAt = (Get-Date).ToUniversalTime().ToString("o")
                    }
                    Write-AiCodingExternalInstallState -Path $paths.installState -State $blockedState
                    Add-AiCodingExternalInstallLog -Path $paths.installLog -Event "install.stop" -Data @{ stage = $gate.status; message = $gate.message }
                    Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $false -Status $gate.status -Message $gate.message -Data @{ status = (Get-AiCodingExternalStatus -SourceName $Source -SkillId $skillId); auditReport = $paths.auditReport })
                }
                $installState = Install-AiCodingExternalSkillTargets -SourceName $Source -SkillId $skillId -InstallTarget $Target -Timeout $TimeoutSec -Replace:$Force -SkipDeps:$NoDeps
                $finishedAt = Get-Date
                $trust = if ($report.status -eq "pass") { "trusted" } else { "allowed-warn" }
                $finalInstallState = [pscustomobject]@{
                    schemaVersion = 1
                    source = $Source
                    skill = $skillId
                    target = $Target
                    stage = "installed"
                    trust = $trust
                    auditAccepted = $true
                    cacheRepo = $paths.cacheRepo
                    skillPath = $installState.skillPath
                    targets = $installState.targets
                    dependency = $installState.dependency
                    audit = [pscustomobject]@{ status = $report.status; score = $report.score }
                    auditReport = $paths.auditReport
                    elapsedSec = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)
                    lastCommand = $downloadState.lastCommand
                    lastExitCode = $downloadState.lastExitCode
                    installedAt = (Get-Date).ToUniversalTime().ToString("o")
                    installedBy = $env:USERNAME
                }
                Write-AiCodingExternalInstallState -Path $paths.installState -State $finalInstallState
                $verified = Test-AiCodingExternalInstall -SourceName $Source -SkillId $skillId -InstallTarget $Target -AllowWarning:$AllowWarn -SkipDeps:$NoDeps
                Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $verified.ok -Status ($(if ($verified.ok) { "installed" } else { "verify_failed" })) -Message "External skill installed from full repo cache." -Data @{ source = $Source; skill = $skillId; target = $Target; paths = @{ cacheRepo = $paths.cacheRepo; repoLocalSkill = $paths.repoLocalSkill; codexSkill = $paths.codexSkill }; audit = @{ status = $report.status; score = $report.score }; status = (Get-AiCodingExternalStatus -SourceName $Source -SkillId $skillId); verification = $verified.checks } -Errors $verified.errors)
            } catch {
                $finishedAt = Get-Date
                $failedState = [pscustomobject]@{
                    schemaVersion = 1
                    source = $Source
                    skill = $skillId
                    stage = "failed"
                    trust = "pending"
                    cacheRepo = $paths.cacheRepo
                    auditReport = $paths.auditReport
                    elapsedSec = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)
                    lastCommand = "install-external"
                    lastExitCode = 1
                    message = ("{0} {1}" -f $_.Exception.Message, $_.ScriptStackTrace)
                    updatedAt = (Get-Date).ToUniversalTime().ToString("o")
                }
                Write-AiCodingExternalInstallState -Path $paths.installState -State $failedState
                Add-AiCodingExternalInstallLog -Path $paths.installLog -Event "install.failed" -Data @{ error = $_.Exception.Message; stack = $_.ScriptStackTrace }
                Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $false -Status "failed" -Message ("{0} {1}" -f $_.Exception.Message, $_.ScriptStackTrace) -Data @{ status = (Get-AiCodingExternalStatus -SourceName $Source -SkillId $skillId) } -Errors @(("{0} {1}" -f $_.Exception.Message, $_.ScriptStackTrace)))
            }
        }
        "install" {
            if ([string]::IsNullOrWhiteSpace($Skill)) { throw "-Skill is required." }
            $verified = Test-AiCodingSkillContent -SkillId $Skill -PreferredSource $Source
            if (-not $verified.ok) { Complete-AiCodingSkillAction $verified }
            $sourcePath = [string]$verified.data.path
            $installRoot = Resolve-RepoPath ".agents/skills"
            $destination = Join-Path $installRoot $Skill
            if ([System.IO.Path]::GetFullPath($sourcePath).Equals([System.IO.Path]::GetFullPath($destination), [System.StringComparison]::OrdinalIgnoreCase)) {
                $files = Get-AiCodingSkillFiles -Path $destination
            } else {
                Copy-AiCodingDirectorySafe -SourcePath $sourcePath -DestinationPath $destination -AllowedRoot $installRoot -Replace:$Force
                $files = Get-AiCodingSkillFiles -Path $destination
            }
            $statePath = Resolve-RepoPath ".aicoding/state/skills/$Skill/install-state.json"
            $licenseFile = @(Get-ChildItem -LiteralPath $destination -File -Filter "LICENSE*" -ErrorAction SilentlyContinue | Select-Object -First 1)
            $state = [pscustomobject]@{
                schemaVersion = 1
                skillId = $Skill
                source = [string]$verified.data.source
                url = ""
                commit = "unknown"
                license = $(if ($licenseFile.Count -gt 0) { $licenseFile[0].Name } else { "unknown" })
                installedAt = (Get-Date).ToUniversalTime().ToString("o")
                installedBy = $env:USERNAME
                files = $files
                trust = [string]$verified.data.trust
                enabled = $true
            }
            if ($verified.data.source -and $verified.data.source -notin @("user-draft", "repo-local")) {
                $sourceRecord = Get-AiCodingSourceRecord -SourceName ([string]$verified.data.source)
                $state.url = [string]$sourceRecord.url
            }
            Write-AiCodingJsonFile -Path $statePath -Value $state
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "installed" -Message "Skill installed into repo-local runtime path." -Data @{ skillId = $Skill; path = $destination; state = $statePath })
        }

        "update" {
            if ([string]::IsNullOrWhiteSpace($Skill)) { throw "-Skill is required." }
            $statePath = Resolve-RepoPath ".aicoding/state/skills/$Skill/install-state.json"
            if (-not (Test-Path -LiteralPath $statePath -PathType Leaf)) { throw "No install state found for $Skill." }
            $state = Read-AiCodingJsonFile -Path $statePath -DefaultValue $null
            if ([string]::IsNullOrWhiteSpace($Pin)) { throw "Third-party skill update requires -Pin <commit-or-tag>." }
            if ([string]::IsNullOrWhiteSpace([string]$state.source) -or $state.source -in @("user-draft", "repo-local")) {
                throw "Only third-party cached skills use this update path."
            }
            $downloaded = Invoke-AiCodingSkillDownload -SourceName ([string]$state.source) -SkillId $Skill -PinnedRef $Pin -Replace
            $verified = Test-AiCodingSkillContent -SkillId $Skill -PreferredSource ([string]$state.source)
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $verified.ok -Status ($(if ($verified.ok) { "updated-cache" } else { "quarantine" })) -Message "Pinned update downloaded to cache; run install explicitly to enable." -Data @{ download = $downloaded; verification = $verified.data } -Errors @($verified.errors))
        }

        "remove" {
            if ([string]::IsNullOrWhiteSpace($Skill)) { throw "-Skill is required." }
            Assert-AiCodingSkillId -Value $Skill
            if (-not $Force) { throw "remove requires -Force to avoid accidental runtime skill deletion." }
            $installRoot = Resolve-RepoPath ".agents/skills"
            $target = Join-Path $installRoot $Skill
            $stateRoot = Resolve-RepoPath ".aicoding/state/skills/$Skill"
            if ((Test-Path -LiteralPath $target) -and (Test-AiCodingChildPath -Path $target -Parent $installRoot)) {
                if ($PSCmdlet.ShouldProcess($target, "Remove repo-local skill")) {
                    Remove-Item -LiteralPath $target -Recurse -Force
                }
            }
            if ((Test-Path -LiteralPath $stateRoot) -and (Test-AiCodingChildPath -Path $stateRoot -Parent (Resolve-RepoPath ".aicoding/state/skills"))) {
                if ($PSCmdlet.ShouldProcess($stateRoot, "Remove skill install state")) {
                    Remove-Item -LiteralPath $stateRoot -Recurse -Force
                }
            }
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "removed" -Message "Repo-local skill runtime state removed." -Data @{ skillId = $Skill })
        }

        "adopt" {
            if ([string]::IsNullOrWhiteSpace($Skill)) { throw "-Skill is required." }
            if ([string]::IsNullOrWhiteSpace($Kit)) { throw "-Kit is required." }
            $verified = Test-AiCodingSkillContent -SkillId $Skill
            if (-not $verified.ok) { Complete-AiCodingSkillAction $verified }
            $registry = Read-AiCodingJsonFile -Path (Resolve-RepoPath "config/kit-registry.json") -DefaultValue $null
            $kitEntry = @($registry.kits | Where-Object { $_.id -eq $Kit }) | Select-Object -First 1
            if (-not $kitEntry) { throw "Unknown kit: $Kit" }
            $plan = [pscustomobject]@{
                skillId = $Skill
                kitId = $Kit
                currentPath = [string]$verified.data.path
                requiredActions = @(
                    "copy-as-new into the kit canonical dist path",
                    "append config/kits/$Kit.json skills.members",
                    "run bin/aicoding.exe skill verify --all --profile Smoke --json",
                    "update docs and CHANGELOG"
                )
                policy = "v2.0 reports the adopt plan only; it does not silently rewrite canonical Kit content."
            }
            Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $true -Status "planned" -Message "Skill adopt plan created." -Data $plan)
        }
    }
}
catch {
    Complete-AiCodingSkillAction (New-AiCodingSkillResult -Ok $false -Status "failed" -Message $_.Exception.Message -Errors @($_.Exception.Message))
}
