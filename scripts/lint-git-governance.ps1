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

$changelog = Get-Content -LiteralPath "CHANGELOG.md" -Raw -Encoding utf8
if ($changelog -notmatch "\[Unreleased\]") {
    Fail "CHANGELOG.md must contain [Unreleased]."
}
if ($changelog -notmatch "\*\*(feat|fix|docs|style|refactor|perf|test|build|ci|chore)\*\*") {
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
    $pattern = "^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([a-z0-9._/-]+\))?: .{8,}$"
    if ($subject -notmatch $pattern) {
        Fail "Commit subject must match <type>(<scope>): <summary>. Got: $subject"
    }
}

Write-Host "Git governance lint passed ($Mode)."