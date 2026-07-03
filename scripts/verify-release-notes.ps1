[CmdletBinding()]
param(
    [string]$Path = "",
    [string]$Text = "",
    [string]$Tag = "",
    [string]$Repo = "JiaxI2/AiCoding",
    [switch]$AllowPlaceholders,
    [switch]$Json
)

$ErrorActionPreference = "Stop"

$checks = New-Object System.Collections.Generic.List[object]
$errors = New-Object System.Collections.Generic.List[string]

function Add-Check {
    param(
        [string]$Name,
        [bool]$Ok,
        [string]$Message
    )

    $checks.Add([pscustomobject]@{
        name = $Name
        ok = $Ok
        message = $Message
    }) | Out-Null

    if (-not $Ok) {
        $errors.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null
    }
}

function Get-ReleaseBody {
    if ($Text) { return $Text }
    if ($Path) {
        if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
            throw "Release notes file not found: $Path"
        }
        return (Get-Content -LiteralPath $Path -Raw -Encoding UTF8)
    }
    if ($Tag) {
        $gh = Get-Command gh -ErrorAction SilentlyContinue
        if (-not $gh) { throw "GitHub CLI gh is required when -Tag is used." }
        $raw = & $gh.Source release view $Tag --repo $Repo --json body
        if ($LASTEXITCODE -ne 0) { throw "gh release view failed for tag: $Tag" }
        return (($raw | ConvertFrom-Json).body)
    }
    throw "Provide -Path, -Text, or -Tag."
}

$body = Get-ReleaseBody

$requiredSections = @(
    @{ id = "summary"; pattern = '(?im)^##\s*(摘要\s*/\s*Summary|Summary\s*/\s*摘要)\s*$' },
    @{ id = "whats-changed"; pattern = '(?im)^##\s*(变更内容\s*/\s*What''s Changed|What''s Changed\s*/\s*变更内容)\s*$' },
    @{ id = "compatibility"; pattern = '(?im)^##\s*(兼容性\s*/\s*Compatibility|Compatibility\s*/\s*兼容性)\s*$' },
    @{ id = "deprecations"; pattern = '(?im)^##\s*(废弃项\s*/\s*Deprecations|Deprecations\s*/\s*废弃项)\s*$' },
    @{ id = "release-notes"; pattern = '(?im)^##\s*(发布说明\s*/\s*Release Notes|Release Notes\s*/\s*发布说明)\s*$' },
    @{ id = "full-changelog"; pattern = '(?im)^##\s*(完整变更\s*/\s*Full Changelog|Full Changelog\s*/\s*完整变更)\s*$' },
    @{ id = "new-contributors"; pattern = '(?im)^##\s*(新贡献者\s*/\s*New Contributors|New Contributors\s*/\s*新贡献者)\s*$' },
    @{ id = "known-issues"; pattern = '(?im)^##\s*(已知问题\s*/\s*Known Issues|Known Issues\s*/\s*已知问题)\s*$' },
    @{ id = "traceability"; pattern = '(?im)^##\s*(可追溯性\s*/\s*Traceability|Traceability\s*/\s*可追溯性)\s*$' },
    @{ id = "assets"; pattern = '(?im)^##\s*(资产\s*/\s*Assets|Assets\s*/\s*资产)\s*$' }
)

foreach ($section in $requiredSections) {
    Add-Check ("section.{0}" -f $section.id) ([regex]::IsMatch($body, $section.pattern)) "required release notes section"
}

if (-not $AllowPlaceholders) {
    Add-Check "content.no-placeholders" (-not ($body -match '\{\{[^}]+\}\}|<[^>\r\n]+>|TODO_PLACEHOLDER|TBD_PLACEHOLDER')) "release notes must not contain template placeholders"
    Add-Check "traceability.commit" ($body -match '(?im)^\s*-\s*\*\*Commit\*\*\s*:') "Traceability must include Commit"
    Add-Check "traceability.verification" ($body -match '(?im)^\s*-\s*\*\*Verification\*\*\s*:') "Traceability must include Verification"
}

$result = [pscustomobject]@{
    schemaVersion = 1
    ok = ($errors.Count -eq 0)
    source = $(if ($Path) { $Path } elseif ($Tag) { ("{0}:{1}" -f $Repo, $Tag) } else { "text" })
    checks = $checks
    errors = @($errors)
}

if ($Json) {
    $result | ConvertTo-Json -Depth 20
} elseif ($result.ok) {
    Write-Host "Release notes verification passed."
} else {
    $errors | ForEach-Object { Write-Error $_ }
}

if (-not $result.ok) { exit 1 }
