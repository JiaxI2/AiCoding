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
    "CHANGELOG.md",
    ".github/repository-governance.toml",
    ".githooks/pre-commit",
    ".githooks/commit-msg"
)

foreach ($file in $requiredFiles) {
    if (-not (Test-Path -LiteralPath $file)) {
        Fail "Required governance file missing: $file"
    }
}

$scanFiles = @("README.md", "CHANGELOG.md", ".github/repository-governance.toml")
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
    $readmeHead = (Get-Content -LiteralPath "README.md" -Encoding utf8 | Select-Object -First 8) -join "`n"
    if ($readmeHead -notmatch "README_CN\.md") {
        Fail "README.md must include a visible top-of-file link to README_CN.md when README_CN.md exists."
    }
}

Require-Content "README.md" "Git Governance Standard|Git 治理标准" "README.md must document the Git governance standard: branch/environment, commit types, single-commit rules, and release typed summaries."
Require-Content "README.md" "feat.+fix.+docs.+style.+refactor.+perf.+test.+chore|feat.+fix.+docs.+build.+ci.+chore" "README.md must document the standard commit type taxonomy."
Require-Content "README.md" "main.+master.+develop.+feature.+test.+release.+hotfix|main.+develop.+feature.+test.+release.+hotfix" "README.md must document branch naming and environment mapping."
Require-Content "README.md" "Release.+type|Release.+typed|按类型汇总|主类型" "README.md must document that Release notes group commits by type and state the primary release type."
$governance = Get-Content -LiteralPath ".github/repository-governance.toml" -Raw -Encoding utf8
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
