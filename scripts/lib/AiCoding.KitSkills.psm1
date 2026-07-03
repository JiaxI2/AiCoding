Import-Module (Join-Path $PSScriptRoot "AiCoding.KitRegistry.psm1") -Force

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
        $line = $lines[$i]
        if ($line -notmatch '^\s*([A-Za-z0-9_-]+)\s*:\s*(.*)\s*$') { continue }
        $key = $Matches[1]
        $value = $Matches[2].Trim().Trim('"').Trim("'")
        $data[$key] = $value
    }

    foreach ($required in @("name", "description")) {
        if (-not $data.ContainsKey($required) -or [string]::IsNullOrWhiteSpace([string]$data[$required])) {
            $errors += "missing frontmatter.$required"
        }
    }

    return [pscustomobject]@{ ok = ($errors.Count -eq 0); data = $data; errors = $errors }
}

function Get-AiCodingKitSkillEntries {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$Kit
    )

    $entries = @()
    $skills = $Kit.manifest.skills
    if (-not $skills) { return @() }

    if ($skills.umbrella) {
        $u = $skills.umbrella
        $entries += [pscustomobject]@{
            kitId = $Kit.id
            id = [string]$u.id
            role = [string]$u.role
            path = [string]$u.path
            description = [string]$u.description
            tags = @($u.tags)
            absolutePath = Resolve-AiCodingKitPath -RepoRoot $RepoRoot -RelativePath ([string]$u.path)
            kind = "umbrella"
        }
    }

    foreach ($member in @($skills.members)) {
        $entries += [pscustomobject]@{
            kitId = $Kit.id
            id = [string]$member.id
            role = [string]$member.role
            path = [string]$member.path
            description = [string]$member.description
            tags = @($member.tags)
            absolutePath = Resolve-AiCodingKitPath -RepoRoot $RepoRoot -RelativePath ([string]$member.path)
            kind = "member"
        }
    }

    return @($entries)
}

function Get-AiCodingKitSkills {
    param(
        [string]$RepoRoot = "",
        [string]$Kit = "",
        [switch]$All
    )

    $root = Resolve-AiCodingKitRepoRoot -RepoRoot $RepoRoot
    if ($All -and $Kit) { throw "Use either -All or -Kit, not both." }
    if (-not $All -and -not $Kit) { throw "skills requires -Kit <id> or -All." }

    $kits = if ($All) { @(Get-AiCodingKitRegistry -RepoRoot $root -Enabled) } else { @(Get-AiCodingKitRegistry -RepoRoot $root -Kit $Kit) }
    $results = @()
    foreach ($entry in $kits) {
        $skills = @(Get-AiCodingKitSkillEntries -RepoRoot $root -Kit $entry | ForEach-Object {
            [pscustomobject]@{
                id = $_.id
                role = $_.role
                kind = $_.kind
                path = $_.path
                description = $_.description
                tags = @($_.tags)
            }
        })
        $results += [pscustomobject]@{
            id = $entry.id
            action = "skills"
            ok = $true
            status = "ok"
            message = "declared skills"
            exitCode = 0
            data = @{ count = $skills.Count; skills = $skills }
            stdout = ""
            stderr = ""
        }
    }
    return @($results)
}

function Test-AiCodingKitSkills {
    param(
        [string]$RepoRoot = "",
        [string]$Kit = "",
        [switch]$All
    )

    $root = Resolve-AiCodingKitRepoRoot -RepoRoot $RepoRoot
    if ($All -and $Kit) { throw "Use either -All or -Kit, not both." }
    if (-not $All -and -not $Kit) { throw "verify-skills requires -Kit <id> or -All." }

    $kits = if ($All) { @(Get-AiCodingKitRegistry -RepoRoot $root -Enabled) } else { @(Get-AiCodingKitRegistry -RepoRoot $root -Kit $Kit) }
    $results = @()
    foreach ($entry in $kits) {
        $errors = @()
        $skillEntries = @(Get-AiCodingKitSkillEntries -RepoRoot $root -Kit $entry)
        $ids = @{}
        $umbrellaCount = @($skillEntries | Where-Object { $_.kind -eq "umbrella" }).Count
        if ($umbrellaCount -gt 1) { $errors += "more than one umbrella skill" }

        foreach ($skill in $skillEntries) {
            if ([string]::IsNullOrWhiteSpace($skill.id)) { $errors += "empty skill id"; continue }
            if ($ids.ContainsKey($skill.id)) { $errors += "duplicate skill id: $($skill.id)" } else { $ids[$skill.id] = $true }
            if ($skill.kind -eq "umbrella" -and @("router", "umbrella") -notcontains $skill.role) { $errors += "invalid umbrella role: $($skill.id) -> $($skill.role)" }
            if ($skill.kind -eq "member" -and $skill.role -ne "subskill") { $errors += "invalid member role: $($skill.id) -> $($skill.role)" }
            if ([string]::IsNullOrWhiteSpace($skill.path)) { $errors += "missing skill path: $($skill.id)"; continue }
            $frontmatter = Get-AiCodingSkillFrontmatter -Path $skill.absolutePath
            if (-not $frontmatter.ok) {
                foreach ($err in @($frontmatter.errors)) { $errors += "$($skill.id): $err" }
            }
        }

        $ok = ($errors.Count -eq 0)
        $results += [pscustomobject]@{
            id = $entry.id
            action = "verify-skills"
            ok = $ok
            status = $(if ($ok) { "ok" } else { "failed" })
            message = "skill declarations"
            exitCode = $(if ($ok) { 0 } else { 1 })
            data = @{ count = $skillEntries.Count; errors = $errors; skills = @($skillEntries | ForEach-Object { @{ id = $_.id; role = $_.role; path = $_.path; kind = $_.kind } }) }
            stdout = ""
            stderr = ""
        }
    }
    return @($results)
}

Export-ModuleMember -Function Get-AiCodingKitSkills, Test-AiCodingKitSkills, Get-AiCodingKitSkillEntries, Get-AiCodingSkillFrontmatter