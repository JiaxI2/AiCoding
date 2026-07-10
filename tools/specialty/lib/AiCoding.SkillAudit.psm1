function New-AiCodingSkillAuditDirectory {
    param([Parameter(Mandatory=$true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
        [void](New-Item -ItemType Directory -Path $Path -Force)
    }
}

function Get-AiCodingAuditObjectValue {
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

function ConvertTo-AiCodingAuditRelativePath {
    param(
        [Parameter(Mandatory=$true)][string]$Root,
        [Parameter(Mandatory=$true)][string]$Path
    )

    return ([System.IO.Path]::GetRelativePath($Root, $Path) -replace '\\', '/')
}

function Read-AiCodingSkillAuditReport {
    param([Parameter(Mandatory=$true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $null }
    return (Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json)
}

function Write-AiCodingSkillAuditReport {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$Report
    )

    New-AiCodingSkillAuditDirectory -Path (Split-Path -Parent $Path)
    $Report | ConvertTo-Json -Depth 40 | Set-Content -LiteralPath $Path -Encoding UTF8
}

function Add-AiCodingSkillAuditLog {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)][string]$Event,
        [object]$Data = $null
    )

    New-AiCodingSkillAuditDirectory -Path (Split-Path -Parent $Path)
    $entry = [ordered]@{
        schemaVersion = 1
        createdAt = (Get-Date).ToUniversalTime().ToString('o')
        event = $Event
        data = $Data
    }
    ($entry | ConvertTo-Json -Depth 12 -Compress) | Add-Content -LiteralPath $Path -Encoding UTF8
}

function New-AiCodingAuditFinding {
    param(
        [string]$Severity,
        [string]$Category,
        [string]$File,
        [int]$Line = 0,
        [string]$Message
    )

    return [pscustomobject]@{
        severity = $Severity
        category = $Category
        file = $File
        line = $Line
        message = $Message
    }
}

function Get-AiCodingSkillRiskScore {
    param([object[]]$Findings = @())

    $score = 100
    foreach ($finding in @($Findings)) {
        switch ([string](Get-AiCodingAuditObjectValue -Object $finding -Name 'severity' -DefaultValue 'low')) {
            'critical' { $score -= 80; break }
            'high' { $score -= 50; break }
            'medium' { $score -= 15; break }
            default { $score -= 3; break }
        }
    }
    if ($score -lt 0) { return 0 }
    return $score
}

function Get-AiCodingSkillAuditFrontmatter {
    param([Parameter(Mandatory=$true)][string]$Path)

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return [pscustomobject]@{ ok = $false; data = @{}; errors = @('missing SKILL.md') }
    }

    try {
        $lines = Get-Content -LiteralPath $Path -Encoding UTF8
    } catch {
        return [pscustomobject]@{ ok = $false; data = @{}; errors = @('SKILL.md is not readable') }
    }

    $data = @{}
    $errors = @()
    if ($lines.Count -lt 3 -or $lines[0].Trim() -ne '---') {
        return [pscustomobject]@{ ok = $false; data = $data; errors = @('missing frontmatter') }
    }

    $end = -1
    for ($i = 1; $i -lt $lines.Count; $i++) {
        if ($lines[$i].Trim() -eq '---') { $end = $i; break }
    }
    if ($end -lt 0) {
        return [pscustomobject]@{ ok = $false; data = $data; errors = @('unterminated frontmatter') }
    }

    for ($i = 1; $i -lt $end; $i++) {
        if ($lines[$i] -notmatch '^\s*([A-Za-z0-9_-]+)\s*:\s*(.*)\s*$') { continue }
        $data[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'")
    }

    foreach ($key in @('name', 'description')) {
        if (-not $data.ContainsKey($key) -or [string]::IsNullOrWhiteSpace([string]$data[$key])) {
            $errors += "missing frontmatter.$key"
        }
    }

    return [pscustomobject]@{ ok = ($errors.Count -eq 0); data = $data; errors = $errors }
}

function Test-AiCodingAuditSkippedPath {
    param([Parameter(Mandatory=$true)][string]$RelativePath)

    foreach ($part in ($RelativePath -split '/')) {
        if ($part -in @('.git', 'node_modules', '.venv', 'venv', '__pycache__')) { return $true }
    }
    return $false
}

function Find-AiCodingAuditTokenLine {
    param(
        [string[]]$Lines,
        [string]$Token
    )

    for ($i = 0; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i].IndexOf($Token, [System.StringComparison]::OrdinalIgnoreCase) -ge 0) { return ($i + 1) }
    }
    return 1
}

function Invoke-AiCodingBuiltinSkillAudit {
    param(
        [Parameter(Mandatory=$true)][string]$Source,
        [Parameter(Mandatory=$true)][string]$Skill,
        [Parameter(Mandatory=$true)][string]$RepoPath,
        [Parameter(Mandatory=$true)][string]$SkillPath,
        [Parameter(Mandatory=$true)][string]$ReportPath,
        [Parameter(Mandatory=$true)][string]$LogPath
    )

    Add-AiCodingSkillAuditLog -Path $LogPath -Event 'audit.start' -Data @{ source = $Source; skill = $Skill; repoPath = $RepoPath; skillPath = $SkillPath }
    $findings = @()
    $checks = [ordered]@{
        skillMdExists = $false
        frontmatterValid = $false
        hasScripts = $false
        hasDependencyFile = $false
        hasHighRiskCommand = $false
    }

    $skillMd = Join-Path $SkillPath 'SKILL.md'
    $checks.skillMdExists = Test-Path -LiteralPath $skillMd -PathType Leaf
    if (-not (Test-Path -LiteralPath $SkillPath -PathType Container)) {
        $findings += New-AiCodingAuditFinding -Severity 'high' -Category 'structure' -File (ConvertTo-AiCodingAuditRelativePath -Root $RepoPath -Path $SkillPath) -Message 'Configured skillPath does not exist.'
    } elseif (-not $checks.skillMdExists) {
        $findings += New-AiCodingAuditFinding -Severity 'high' -Category 'structure' -File (ConvertTo-AiCodingAuditRelativePath -Root $RepoPath -Path $skillMd) -Message 'SKILL.md is missing.'
    } else {
        $frontmatter = Get-AiCodingSkillAuditFrontmatter -Path $skillMd
        $checks.frontmatterValid = [bool]$frontmatter.ok
        foreach ($frontmatterError in @($frontmatter.errors)) {
            $findings += New-AiCodingAuditFinding -Severity 'high' -Category 'structure' -File (ConvertTo-AiCodingAuditRelativePath -Root $RepoPath -Path $skillMd) -Message $frontmatterError
        }
    }

    $scriptExtensions = @('.ps1', '.sh', '.py', '.js', '.ts', '.bat', '.cmd')
    $dependencyNames = @('requirements.txt', 'package.json', 'pyproject.toml', 'uv.lock', 'package-lock.json')
    $blockTokens = @(
        'Invoke-Expression', 'EncodedCommand', 'FromBase64String', 'Start-Process powershell', 'Start-Process pwsh',
        'Remove-Item -Recurse -Force', 'Register-ScheduledTask', 'schtasks',
        'curl | sh', 'wget | sh', 'rm -rf', 'chmod 777', 'crontab', 'systemctl', 'launchctl'
    )
    $warnTokens = @('Set-ExecutionPolicy', 'New-ItemProperty', 'Set-ItemProperty', 'HKCU:', 'HKLM:', 'OPENAI_API_KEY', 'ANTHROPIC_API_KEY', 'GITHUB_TOKEN', 'SSH_PRIVATE_KEY', 'id_rsa', 'browser profile', 'Credential', 'eval(', 'exec(', 'subprocess', 'os.system', 'child_process', 'spawn(', 'execSync(', 'Invoke-WebRequest', 'curl', 'wget', 'requests.get', 'fetch(', 'git clone', '.env')

    foreach ($file in @(Get-ChildItem -LiteralPath $RepoPath -Recurse -File -Force -ErrorAction SilentlyContinue)) {
        $relative = ConvertTo-AiCodingAuditRelativePath -Root $RepoPath -Path $file.FullName
        if (Test-AiCodingAuditSkippedPath -RelativePath $relative) { continue }
        $leaf = $file.Name.ToLowerInvariant()
        $ext = $file.Extension.ToLowerInvariant()

        if ($ext -in $scriptExtensions) {
            $checks.hasScripts = $true
            $findings += New-AiCodingAuditFinding -Severity 'medium' -Category 'script' -File $relative -Message 'Executable script file found; review before trusting this third-party skill.'
        }
        if ($leaf -in $dependencyNames) {
            $checks.hasDependencyFile = $true
            $findings += New-AiCodingAuditFinding -Severity 'medium' -Category 'dependency' -File $relative -Message 'Dependency manifest found; install requires user trust.'
        }

        if ($file.Length -gt 1048576) { continue }
        try { $text = Get-Content -LiteralPath $file.FullName -Raw -Encoding UTF8 -ErrorAction Stop } catch { continue }
        $lines = $text -split "`r?`n"
        foreach ($token in $blockTokens) {
            if ($text.IndexOf($token, [System.StringComparison]::OrdinalIgnoreCase) -ge 0) {
                $checks.hasHighRiskCommand = $true
                $findings += New-AiCodingAuditFinding -Severity 'high' -Category 'high-risk' -File $relative -Line (Find-AiCodingAuditTokenLine -Lines $lines -Token $token) -Message ("High-risk token found: {0}" -f $token)
            }
        }
        foreach ($token in $warnTokens) {
            if ($text.IndexOf($token, [System.StringComparison]::OrdinalIgnoreCase) -ge 0) {
                $findings += New-AiCodingAuditFinding -Severity 'medium' -Category 'review' -File $relative -Line (Find-AiCodingAuditTokenLine -Lines $lines -Token $token) -Message ("Review token found: {0}" -f $token)
            }
        }
    }

    $hasBlock = @($findings | Where-Object { $_.severity -in @('critical', 'high') }).Count -gt 0
    $hasWarn = @($findings | Where-Object { $_.severity -eq 'medium' }).Count -gt 0
    $status = if ($hasBlock) { 'block' } elseif ($hasWarn) { 'warn' } else { 'pass' }
    $score = Get-AiCodingSkillRiskScore -Findings $findings
    if ($status -eq 'block' -and $score -gt 30) { $score = 30 }
    if ($status -eq 'warn' -and $score -gt 75) { $score = 75 }
    $summary = switch ($status) {
        'pass' { 'No obvious risk found by builtin audit.' }
        'warn' { 'Skill has scripts, dependencies, or review tokens. Re-run install-external with -AllowWarn to install.' }
        default { 'Skill audit found high-risk structure, credential, persistence, hidden execution, or destructive behavior.' }
    }

    $report = [pscustomobject]@{
        schemaVersion = 1
        source = $Source
        skill = $Skill
        provider = 'builtin'
        status = $status
        score = $score
        summary = $summary
        findings = @($findings)
        checks = [pscustomobject]$checks
        createdAt = (Get-Date).ToUniversalTime().ToString('o')
    }
    Write-AiCodingSkillAuditReport -Path $ReportPath -Report $report
    Add-AiCodingSkillAuditLog -Path $LogPath -Event 'audit.finish' -Data @{ status = $status; score = $score; findingCount = @($findings).Count }
    return $report
}

function Invoke-AiCodingObservedProcess {
    param(
        [Parameter(Mandatory=$true)][string]$Command,
        [Alias('Args')][string[]]$ArgumentList = @(),
        [string]$WorkingDirectory = (Get-Location).Path,
        [int]$TimeoutSec = 180
    )

    $psi = [System.Diagnostics.ProcessStartInfo]::new()
    $psi.FileName = $Command
    foreach ($processArg in @($ArgumentList)) { [void]$psi.ArgumentList.Add([string]$processArg) }
    $psi.WorkingDirectory = $WorkingDirectory
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.UseShellExecute = $false
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $psi
    $startedAt = Get-Date
    [void]$process.Start()
    $completed = $process.WaitForExit([Math]::Max(1, $TimeoutSec) * 1000)
    if (-not $completed) {
        try { $process.Kill($true) } catch { Write-Verbose ("failed to kill timed-out process: {0}" -f $_.Exception.Message) }
    }
    $stdout = $process.StandardOutput.ReadToEnd()
    $stderr = $process.StandardError.ReadToEnd()
    $finishedAt = Get-Date
    return [pscustomobject]@{
        command = $Command
        args = @($ArgumentList)
        exitCode = $(if ($completed) { $process.ExitCode } else { -1 })
        timedOut = (-not $completed)
        stdout = $stdout
        stderr = $stderr
        elapsedSec = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)
    }
}

function Invoke-AiCodingSkillAudit {
    param(
        [Parameter(Mandatory=$true)][string]$Source,
        [Parameter(Mandatory=$true)][string]$Skill,
        [Parameter(Mandatory=$true)][string]$RepoPath,
        [Parameter(Mandatory=$true)][string]$SkillPath,
        [Parameter(Mandatory=$true)][string]$ReportPath,
        [Parameter(Mandatory=$true)][string]$LogPath
    )

    try {
        return Invoke-AiCodingBuiltinSkillAudit -Source $Source -Skill $Skill -RepoPath $RepoPath -SkillPath $SkillPath -ReportPath $ReportPath -LogPath $LogPath
    } catch {
        $report = [pscustomobject]@{
            schemaVersion = 1
            source = $Source
            skill = $Skill
            provider = 'builtin'
            status = 'error'
            score = 0
            summary = $_.Exception.Message
            findings = @()
            checks = [pscustomobject]@{}
            createdAt = (Get-Date).ToUniversalTime().ToString('o')
        }
        Write-AiCodingSkillAuditReport -Path $ReportPath -Report $report
        Add-AiCodingSkillAuditLog -Path $LogPath -Event 'audit.error' -Data @{ error = $_.Exception.Message }
        return $report
    }
}

Export-ModuleMember -Function Invoke-AiCodingSkillAudit, Invoke-AiCodingBuiltinSkillAudit, Read-AiCodingSkillAuditReport, Write-AiCodingSkillAuditReport, Add-AiCodingSkillAuditLog, Get-AiCodingSkillRiskScore, Invoke-AiCodingObservedProcess
