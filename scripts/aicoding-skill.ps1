[CmdletBinding(SupportsShouldProcess=$true)]
param(
    [Parameter(Position=0, Mandatory=$true)]
    [ValidateSet("sources", "add-source", "download", "install", "verify", "update", "remove", "create", "adopt", "list")]
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
    [switch]$Json,
    [switch]$Force
)

$ErrorActionPreference = "Stop"

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
                @{ name = "third-party-cache"; path = (Resolve-RepoPath ".aicoding/skill-cache/third-party") }
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
                    "run pwsh scripts/aicoding-kit.ps1 verify-skills -Kit $Kit -Json",
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
