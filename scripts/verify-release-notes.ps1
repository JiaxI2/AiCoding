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

# Chinese section names use \uXXXX regex escapes so this script stays pure
# ASCII and immune to source-file encoding corruption.
function New-SectionPattern {
    param(
        [string]$Chinese,
        [string]$English
    )
    return ('(?im)^##\s*({0}\s*/\s*{1}|{1}\s*/\s*{0})\s*$' -f $Chinese, $English)
}

$requiredSections = @(
    @{ id = "summary"; pattern = (New-SectionPattern '\u6458\u8981' 'Summary') },
    @{ id = "whats-changed"; pattern = (New-SectionPattern '\u53d8\u66f4\u5185\u5bb9' 'What''s Changed') },
    @{ id = "compatibility"; pattern = (New-SectionPattern '\u517c\u5bb9\u6027' 'Compatibility') },
    @{ id = "deprecations"; pattern = (New-SectionPattern '\u5e9f\u5f03\u9879' 'Deprecations') },
    @{ id = "release-notes"; pattern = (New-SectionPattern '\u53d1\u5e03\u8bf4\u660e' 'Release Notes') },
    @{ id = "full-changelog"; pattern = (New-SectionPattern '\u5b8c\u6574\u53d8\u66f4' 'Full Changelog') },
    @{ id = "new-contributors"; pattern = (New-SectionPattern '\u65b0\u8d21\u732e\u8005' 'New Contributors') },
    @{ id = "known-issues"; pattern = (New-SectionPattern '\u5df2\u77e5\u95ee\u9898' 'Known Issues') },
    @{ id = "traceability"; pattern = (New-SectionPattern '\u53ef\u8ffd\u6eaf\u6027' 'Traceability') },
    @{ id = "assets"; pattern = (New-SectionPattern '\u8d44\u4ea7' 'Assets') }
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
