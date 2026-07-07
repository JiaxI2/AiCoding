# Deprecated: this fast-path check is superseded by bin\aicoding.exe governance lint --json.
# Kept as a temporary fallback for v0.1.x.
# Do not call from Taskfile smoke or Git hooks.

param(
    [ValidateSet("all", "pre-commit", "commit-msg")]
    [string]$Mode = "all",

    [string]$CommitMsgPath = ""
)

$ErrorActionPreference = "Stop"

function Fail([string]$Message) {
    Write-Error $Message
    exit 1
}

$repoRoot = (& git rev-parse --show-toplevel).Trim()
if (-not $repoRoot) { Fail "Not inside a Git repository." }
Set-Location $repoRoot

$requiredFiles = @(
    "README.md",
    "README_EN.md",
    "CHANGELOG.md",
    ".github/RELEASE_TEMPLATE.md",
    ".github/repository-governance.toml",
    ".githooks/pre-commit",
    ".githooks/commit-msg",
    "scripts/legacy/fast-path-replaced/verify-release-notes.ps1"
)

foreach ($file in $requiredFiles) {
    if (-not (Test-Path -LiteralPath $file)) {
        Fail "Required governance file missing: $file"
    }
}

$scanFiles = @("README.md", "README_CN.md", "README_EN.md", "CHANGELOG.md", ".github/repository-governance.toml")
foreach ($file in $scanFiles) {
    $content = Get-Content -LiteralPath $file -Raw -Encoding utf8
    if ($content -match "\{\{[^}]+\}\}|UNRESOLVED_PLACEHOLDER|TODO_PLACEHOLDER") {
        Fail "Unresolved placeholder found in $file"
    }
}


function Require-Content([string]$Path, [string]$Pattern, [string]$Message) {
    $content = Get-Content -LiteralPath $Path -Raw -Encoding utf8
    if ($content -notmatch $Pattern) {
        Fail $Message
    }
}

if (Test-Path -LiteralPath "README_CN.md") {
    $readmeContent = Get-Content -LiteralPath "README.md" -Raw -Encoding utf8
    $readmeHead = (Get-Content -LiteralPath "README.md" -Encoding utf8 | Select-Object -First 16) -join "`n"
    $readmeTop = (Get-Content -LiteralPath "README.md" -Encoding utf8 | Select-Object -First 24) -join "`n"
    if ($readmeHead -notmatch "README_CN\.md") {
        Fail "README.md must include a visible top-of-file README_CN.md link for bilingual switching."
    }
    if (-not (Test-Path -LiteralPath "README_EN.md")) {
        Fail "README_EN.md is required as the file-level English README."
    }
    if ($readmeHead -notmatch "README_EN\.md") {
        Fail "README.md must include a visible top-of-file README_EN.md link for English switching."
    }
    if ($readmeHead -match "README\.md#english") {
        Fail "README.md must not use an in-page English anchor; link to README_EN.md instead."
    }
    if (-not $readmeTop.Contains("AiCoding 是")) {
        Fail "README.md must be the Chinese-first default repository entry."
    }
    $requiredReadmeHeadTokens = @(
        @{ token = "img.shields.io/github/v/release/JiaxI2/AiCoding"; message = "README.md must keep the Release badge link." },
        @{ token = "https://go.dev/"; message = "README.md must keep the Go URL badge link." },
        @{ token = "https://learn.microsoft.com/powershell/"; message = "README.md must keep the PowerShell URL badge link." },
        @{ token = "https://www.python.org/"; message = "README.md must keep the Python URL badge link." },
        @{ token = "https://taskfile.dev/"; message = "README.md must keep the Taskfile URL badge link." },
        @{ token = "github/license/JiaxI2/AiCoding"; message = "README.md must keep the License badge link." }
    )
    foreach ($item in $requiredReadmeHeadTokens) {
        if (-not $readmeHead.Contains($item.token)) { Fail $item.message }
    }
    $requiredReadmeUrlTokens = @(
        @{ token = "## 环境 URL / Environment URLs"; message = "README.md must keep the Environment URLs section." },
        @{ token = "https://github.com/JiaxI2/AiCoding/releases/latest"; message = "README.md must keep the latest release URL." },
        @{ token = "https://github.com/JiaxI2/AiCoding/releases"; message = "README.md must keep the releases URL." },
        @{ token = "https://github.com/JiaxI2/AiCoding/tags"; message = "README.md must keep the tags URL." },
        @{ token = "[CHANGELOG.md](CHANGELOG.md)"; message = "README.md must keep the CHANGELOG link." },
        @{ token = "[CodingKit/README.md](CodingKit/README.md)"; message = "README.md must keep the CodingKit README link." }
    )
    foreach ($item in $requiredReadmeUrlTokens) {
        if (-not $readmeContent.Contains($item.token)) { Fail $item.message }
    }
    $governanceContent = Get-Content -LiteralPath ".github/repository-governance.toml" -Raw -Encoding utf8
    if ($governanceContent -notlike '*primary_language = "zh-CN"*') {
        Fail ".github/repository-governance.toml must set README primary_language to zh-CN."
    }
    if ($governanceContent -notlike '*secondary_language_surface = "top-file-language-switch-and-github-about"*') {
        Fail ".github/repository-governance.toml must route README_CN.md through the top file-level language switch and GitHub About/Homepage."
    }
    if ($governanceContent -notlike '*english_language_file = "README_EN.md"*') {
        Fail ".github/repository-governance.toml must define README_EN.md as the English README file."
    }
    if ($governanceContent -notlike '*quick_environment_preview = true*') {
        Fail ".github/repository-governance.toml must require the clickable README environment preview."
    }
    if ($governanceContent -notmatch '\[github_about\]' -or $governanceContent -notlike '*require_bilingual = true*') {
        Fail ".github/repository-governance.toml must require bilingual GitHub About metadata."
    }
}

Require-Content "README.md" "Git Governance Standard|Git 治理标准" "README.md must document the Git governance standard: branch/environment, commit types, single-commit rules, and release typed summaries."
Require-Content "README.md" "feat.+fix.+docs.+style.+refactor.+perf.+test.+chore|feat.+fix.+docs.+build.+ci.+chore" "README.md must document the standard commit type taxonomy."
Require-Content "README.md" "main.+master.+develop.+feature.+test.+release.+hotfix|main.+develop.+feature.+test.+release.+hotfix" "README.md must document branch naming and environment mapping."
Require-Content "README.md" "Release.+type|Release.+typed|按类型汇总|主类型" "README.md must document that Release notes group commits by type and state the primary release type."
Require-Content "README_EN.md" "Git Governance Standard|Git 治理标准" "README_EN.md must document the Git governance standard."
Require-Content "README_EN.md" "feat.+fix.+docs.+style.+refactor.+perf.+test.+chore|feat.+fix.+docs.+build.+ci.+chore" "README_EN.md must document the standard commit type taxonomy."
$governance = Get-Content -LiteralPath ".github/repository-governance.toml" -Raw -Encoding utf8
if ($governance -notmatch 'notes_template\s*=\s*"\.github/RELEASE_TEMPLATE\.md"') {
    Fail ".github/repository-governance.toml must declare the release notes template."
}
if ($governance -notmatch 'notes_validator\s*=\s*"scripts/legacy/fast-path-replaced/verify-release-notes\.ps1"') {
    Fail ".github/repository-governance.toml must declare the release notes validator."
}
if ($governance -notmatch 'required_bilingual_sections') {
    Fail ".github/repository-governance.toml must require bilingual release notes sections."
}
& (Join-Path $repoRoot "scripts/legacy/fast-path-replaced/verify-release-notes.ps1") -Path ".github/RELEASE_TEMPLATE.md" -AllowPlaceholders -Json | Out-Null
if ($LASTEXITCODE -ne 0) {
    Fail ".github/RELEASE_TEMPLATE.md must pass scripts/legacy/fast-path-replaced/verify-release-notes.ps1."
}
$changelogMode = ""
if ($governance -match '(?m)^mode\s*=\s*"([^"]+)"') {
    $changelogMode = $Matches[1]
}

$changelog = Get-Content -LiteralPath "CHANGELOG.md" -Raw -Encoding utf8
if ($changelogMode -eq "unreleased" -and $changelog -notmatch "\[Unreleased\]") {
    Fail "CHANGELOG.md must contain [Unreleased] when changelog.mode is unreleased."
}
if ($changelog -notmatch "\*\*(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([^)]*\))?\*\*") {
    Fail "CHANGELOG.md must include typed entries such as **docs** or **chore**."
}

if ($Mode -eq "all" -or $Mode -eq "pre-commit") {
    $staged = @(& git diff --cached --name-only --diff-filter=ACMR)
    if ($staged.Count -gt 0 -and -not ($staged -contains "CHANGELOG.md")) {
        if ($env:AICODING_SKIP_CHANGELOG -ne "1") {
            Fail "CHANGELOG.md must be staged for normal commits. Set AICODING_SKIP_CHANGELOG=1 only for an approved exclusion."
        }
    }
}

if ($Mode -eq "commit-msg") {
    if (-not $CommitMsgPath) { Fail "Commit message path is required." }
    $subject = (Get-Content -LiteralPath $CommitMsgPath -Encoding utf8 | Where-Object { $_ -and -not $_.StartsWith("#") } | Select-Object -First 1)
    if (-not $subject) { Fail "Commit message subject is empty." }
    if ($subject -notmatch '^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([^)]+\))?: ') {
        Fail ('Commit subject must start with an allowed type and optional scope. Got: ' + $subject)
    }
    if ($subject -notmatch ': .{8,}$') {
        Fail ('Commit subject summary must be at least 8 characters. Got: ' + $subject)
    }
}

Write-Host "Git governance lint passed ($Mode)."
