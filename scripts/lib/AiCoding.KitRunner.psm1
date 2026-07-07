Import-Module (Join-Path $PSScriptRoot "AiCoding.KitRegistry.psm1") -Force
Import-Module (Join-Path $PSScriptRoot "AiCoding.KitPackage.psm1") -Force
Import-Module (Join-Path $PSScriptRoot "AiCoding.KitSkills.psm1") -Force

function Quote-AiCodingProcessArgument {
    param([string]$Value)

    if ($null -eq $Value) { return '""' }
    if ($Value.Length -eq 0) { return '""' }
    if ($Value -notmatch '[\s"]') { return $Value }
    $escaped = $Value -replace '(\\*)"', '$1$1\"'
    $escaped = $escaped -replace '(\\+)$', '$1$1'
    return '"' + $escaped + '"'
}

function Join-AiCodingProcessArguments {
    param([object[]]$Arguments)

    $parts = @()
    foreach ($arg in $Arguments) { $parts += (Quote-AiCodingProcessArgument ([string]$arg)) }
    return ($parts -join " ")
}

function Resolve-AiCodingToken {
    param(
        [Parameter(Mandatory=$true)]$Value,
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$Kit
    )

    if ($null -eq $Value) { return $null }
    $text = [string]$Value
    $text = $text.Replace('${repoRoot}', $RepoRoot)
    $text = $text.Replace('${kitId}', [string]$Kit.id)
    $text = $text.Replace('${version}', [string]$Kit.manifest.version)
    return $text
}

function Invoke-AiCodingProcessCapture {
    param(
        [Parameter(Mandatory=$true)][string]$Command,
        [object[]]$Arguments = @(),
        [string]$WorkingDirectory = "",
        [int]$TimeoutSeconds = 20
    )

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $Command
    $psi.Arguments = Join-AiCodingProcessArguments -Arguments $Arguments
    if ($WorkingDirectory) { $psi.WorkingDirectory = $WorkingDirectory }
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true

    $p = New-Object System.Diagnostics.Process
    $p.StartInfo = $psi
    $timedOut = $false
    [void]$p.Start()
    $stdoutTask = $p.StandardOutput.ReadToEndAsync()
    $stderrTask = $p.StandardError.ReadToEndAsync()

    if ($TimeoutSeconds -gt 0) {
        $completed = $p.WaitForExit($TimeoutSeconds * 1000)
        if (-not $completed) {
            $timedOut = $true
            try { $p.Kill($true) } catch { try { $p.Kill() } catch { } }
            $p.WaitForExit()
        }
    } else {
        $p.WaitForExit()
    }

    try { $stdout = $stdoutTask.GetAwaiter().GetResult() } catch { $stdout = "" }
    try { $stderr = $stderrTask.GetAwaiter().GetResult() } catch { $stderr = "" }
    $exitCode = if ($timedOut) { 124 } else { $p.ExitCode }

    return [pscustomobject]@{
        command = $Command
        arguments = $Arguments
        workingDirectory = $WorkingDirectory
        exitCode = $exitCode
        stdout = $stdout
        stderr = $stderr
        timedOut = $timedOut
        timeoutSeconds = $TimeoutSeconds
    }
}

function ConvertFrom-AiCodingJsonOutput {
    param([string]$Text)

    if ([string]::IsNullOrWhiteSpace($Text)) { return $null }
    $trimmed = $Text.Trim()
    if (-not ($trimmed.StartsWith("{") -or $trimmed.StartsWith("["))) { return $null }
    try { return $trimmed | ConvertFrom-Json } catch { return $null }
}

function New-AiCodingKitActionResult {
    param(
        [string]$Action,
        [string]$KitId,
        [bool]$Ok,
        [string]$Status,
        [string]$Message,
        $Data = $null,
        [int]$ExitCode = 0,
        [string]$Stdout = "",
        [string]$Stderr = ""
    )

    return [pscustomobject]@{
        id = $KitId
        action = $Action
        ok = [bool]$Ok
        status = $Status
        message = $Message
        exitCode = $ExitCode
        data = $Data
        stdout = $Stdout
        stderr = $Stderr
    }
}

function Get-AiCodingCommandDefinition {
    param(
        [Parameter(Mandatory=$true)]$Kit,
        [Parameter(Mandatory=$true)][string]$Action
    )

    $commands = $Kit.manifest.commands
    if (-not $commands.PSObject.Properties.Name.Contains($Action)) { return $null }
    return $commands.$Action
}

function Invoke-AiCodingKitSmokeTest {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$Kit,
        [string]$Action = "test"
    )

    $errors = @()
    if (-not (Test-Path -LiteralPath $Kit.manifestPath -PathType Leaf)) { $errors += "manifest missing" }
    if ($Kit.manifest.id -ne $Kit.id) { $errors += "manifest id mismatch" }
    if (@("script-adapter", "declarative") -notcontains $Kit.manifest.mode) { $errors += "invalid mode" }
    if (@($Kit.manifest.kind).Count -eq 0) { $errors += "empty kind" }
    foreach ($commandProperty in @($Kit.manifest.commands.PSObject.Properties)) {
        $command = $commandProperty.Value
        if ($command.type -ne "powershell-script") { continue }
        $path = Resolve-AiCodingKitPath -RepoRoot $RepoRoot -RelativePath ([string]$command.path)
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            $errors += ("missing command script: {0}.{1} -> {2}" -f $Kit.id, $commandProperty.Name, $command.path)
        }
    }

    $ok = ($errors.Count -eq 0)
    return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $ok -Status ($(if ($ok) { "smoke" } else { "failed" })) -Message ("smoke profile manifest {0}" -f $Action) -Data @{ profile = "Smoke"; errors = $errors }
}

function Invoke-AiCodingKitCommand {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$Kit,
        [Parameter(Mandatory=$true)][string]$Action,
        [Parameter(Mandatory=$true)]$CommandDefinition,
        [switch]$Json,
        [switch]$DryRun,
        [switch]$Zip,
        [int]$TimeoutSeconds = 20
    )

    $dryRunNoExecuteActions = @("install", "update", "uninstall")

    switch ($CommandDefinition.type) {
        "powershell-script" {
            if ($DryRun -and ($dryRunNoExecuteActions -contains $Action) -and $CommandDefinition.supportsDryRun -ne $true) {
                return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $true -Status "skipped" -Message "Dry-run skipped command without supportsDryRun: $($CommandDefinition.path)"
            }

            $scriptPath = Resolve-AiCodingKitPath -RepoRoot $RepoRoot -RelativePath $CommandDefinition.path
            if (-not (Test-Path -LiteralPath $scriptPath)) {
                return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $false -Status "missing" -Message "Script not found: $($CommandDefinition.path)"
            }

            $pwsh = Get-Command pwsh -ErrorAction SilentlyContinue
            $shell = if ($pwsh) { $pwsh.Source } else { "powershell" }
            $args = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $scriptPath)
            foreach ($arg in @($CommandDefinition.args)) {
                $args += (Resolve-AiCodingToken -Value $arg -RepoRoot $RepoRoot -Kit $Kit)
            }
            if ($Json -and $CommandDefinition.supportsJson -ne $false) { $args += "-Json" }
            if ($DryRun -and $CommandDefinition.supportsDryRun -eq $true) { $args += "-DryRun" }
            if ($CommandDefinition.extraArgs) {
                foreach ($arg in @($CommandDefinition.extraArgs)) {
                    $args += (Resolve-AiCodingToken -Value $arg -RepoRoot $RepoRoot -Kit $Kit)
                }
            }

            $capture = Invoke-AiCodingProcessCapture -Command $shell -Arguments $args -WorkingDirectory $RepoRoot -TimeoutSeconds $TimeoutSeconds
            $parsed = ConvertFrom-AiCodingJsonOutput -Text $capture.stdout
            $status = if ($capture.timedOut) { "timeout" } elseif ($capture.exitCode -eq 0) { "ok" } else { "failed" }
            $message = if ($capture.timedOut) { "Timed out after $($capture.timeoutSeconds)s: $($CommandDefinition.path)" } else { $CommandDefinition.path }
            return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok ($capture.exitCode -eq 0) -Status $status -Message $message -Data $parsed -ExitCode $capture.exitCode -Stdout $capture.stdout -Stderr $capture.stderr
        }

        "external-command" {
            if ($DryRun -and ($dryRunNoExecuteActions -contains $Action) -and $CommandDefinition.supportsDryRun -ne $true) {
                return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $true -Status "skipped" -Message "Dry-run skipped command without supportsDryRun: $($CommandDefinition.executable)"
            }

            $exe = Resolve-AiCodingToken -Value $CommandDefinition.executable -RepoRoot $RepoRoot -Kit $Kit
            $args = @()
            foreach ($arg in @($CommandDefinition.args)) {
                $args += (Resolve-AiCodingToken -Value $arg -RepoRoot $RepoRoot -Kit $Kit)
            }
            $capture = Invoke-AiCodingProcessCapture -Command $exe -Arguments $args -WorkingDirectory $RepoRoot -TimeoutSeconds $TimeoutSeconds
            $parsed = ConvertFrom-AiCodingJsonOutput -Text $capture.stdout
            $status = if ($capture.timedOut) { "timeout" } elseif ($capture.exitCode -eq 0) { "ok" } else { "failed" }
            $message = if ($capture.timedOut) { "Timed out after $($capture.timeoutSeconds)s: $exe" } else { $exe }
            return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok ($capture.exitCode -eq 0) -Status $status -Message $message -Data $parsed -ExitCode $capture.exitCode -Stdout $capture.stdout -Stderr $capture.stderr
        }

        "composed" {
            $results = @()
            $ok = $true
            foreach ($step in @($CommandDefinition.steps)) {
                $stepDef = Get-AiCodingCommandDefinition -Kit $Kit -Action $step
                if (-not $stepDef) {
                    $results += New-AiCodingKitActionResult -Action $step -KitId $Kit.id -Ok $false -Status "missing" -Message "Missing composed step: $step"
                    $ok = $false
                    continue
                }
                $stepResult = Invoke-AiCodingKitCommand -RepoRoot $RepoRoot -Kit $Kit -Action $step -CommandDefinition $stepDef -Json:$Json -DryRun:$DryRun -Zip:$Zip -TimeoutSeconds $TimeoutSeconds
                $results += $stepResult
                if (-not $stepResult.ok) { $ok = $false }
            }
            return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $ok -Status ($(if ($ok) { "ok" } else { "failed" })) -Message "composed action" -Data @{ steps = $results }
        }

        "builtin-check" {
            $missing = @()
            foreach ($rel in @($CommandDefinition.requiredPaths)) {
                $path = Resolve-AiCodingKitPath -RepoRoot $RepoRoot -RelativePath $rel
                if (-not (Test-Path -LiteralPath $path)) { $missing += $rel }
            }
            $ok = ($missing.Count -eq 0)
            return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $ok -Status ($(if ($ok) { "ok" } else { "missing" })) -Message "builtin path check" -Data @{ missing = $missing; requiredPaths = @($CommandDefinition.requiredPaths) }
        }

        "builtin-package" {
            $packageResult = Export-AiCodingKit -RepoRoot $RepoRoot -Kit $Kit -CommandDefinition $CommandDefinition -DryRun:$DryRun -Zip:$Zip
            return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $packageResult.ok -Status $packageResult.status -Message $packageResult.message -Data $packageResult.data
        }

        "unsupported" {
            if ($DryRun) {
                return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $true -Status "skipped" -Message ("Dry-run skipped unsupported action: {0}" -f ([string]$CommandDefinition.reason))
            }

            return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $false -Status "unsupported" -Message ([string]$CommandDefinition.reason)
        }

        default {
            return New-AiCodingKitActionResult -Action $Action -KitId $Kit.id -Ok $false -Status "unsupported" -Message "Unsupported command type: $($CommandDefinition.type)"
        }
    }
}

function Invoke-AiCodingKitAction {
    param(
        [string]$RepoRoot = "",
        [Parameter(Mandatory=$true)][string]$Action,
        [string]$Kit = "",
        [switch]$All,
        [switch]$Json,
        [switch]$DryRun,
        [switch]$Zip,
        [ValidateSet("Smoke", "Full", "Release")]
        [string]$Profile = "Smoke"
    )

    $root = Resolve-AiCodingKitRepoRoot -RepoRoot $RepoRoot

    if ($Action -eq "list") {
        $kits = @(Get-AiCodingKitRegistry -RepoRoot $root)
        return [pscustomobject]@{
            schemaVersion = 2
            action = "list"
            mode = "registry"
            ok = $true
            summary = @{ total = $kits.Count; enabled = @($kits | Where-Object { $_.enabled }).Count }
            kits = @($kits | ForEach-Object {
                [pscustomobject]@{
                    id = $_.id
                    enabled = $_.enabled
                    order = $_.order
                    name = $_.manifest.name
                    version = $_.manifest.version
                    kind = @($_.manifest.kind)
                    mode = $_.manifest.mode
                    manifest = $_.manifestRelativePath
                }
            })
        }
    }

    if ($Action -eq "doctor") {
        $doctor = Test-AiCodingKitRegistry -RepoRoot $root
        return [pscustomobject]@{
            schemaVersion = 2
            action = "doctor"
            mode = "registry"
            ok = [bool]$doctor.ok
            summary = @{ total = $doctor.total; failed = @($doctor.errors).Count }
            kits = @()
            errors = @($doctor.errors)
        }
    }

    if ($Action -eq "skills" -or $Action -eq "verify-skills") {
        if ($All -and $Kit) { throw "Use either -All or -Kit, not both." }
        if (-not $All -and -not $Kit) { throw "Action '$Action' requires -Kit <id> or -All." }
        $results = if ($Action -eq "skills") {
            @(Get-AiCodingKitSkills -RepoRoot $root -Kit $Kit -All:$All)
        } else {
            @(Test-AiCodingKitSkills -RepoRoot $root -Kit $Kit -All:$All)
        }
        $failed = @($results | Where-Object { -not $_.ok })
        return [pscustomobject]@{
            schemaVersion = 2
            action = $Action
            mode = $(if ($All) { "all" } else { "kit" })
            ok = ($failed.Count -eq 0)
            summary = @{
                total = $results.Count
                ok = @($results | Where-Object { $_.ok }).Count
                failed = $failed.Count
                declaredSkills = (@($results | ForEach-Object { @($_.data.skills).Count }) | Measure-Object -Sum).Sum
            }
            kits = $results
            bundle = $null
        }
    }

    if ($All -and $Kit) { throw "Use either -All or -Kit, not both." }
    if (-not $All -and -not $Kit) { throw "Action '$Action' requires -Kit <id> or -All." }

    $kits = if ($All) { @(Get-AiCodingKitRegistry -RepoRoot $root -Enabled) } else { @(Get-AiCodingKitRegistry -RepoRoot $root -Kit $Kit) }
    if ($kits.Count -eq 0) { throw "No kit matched." }

    $results = @()
    $timeoutSeconds = switch ($Profile) {
        "Smoke" { 20 }
        "Full" { 120 }
        "Release" { 300 }
    }
    foreach ($entry in $kits) {
        $cmd = Get-AiCodingCommandDefinition -Kit $entry -Action $Action
        if (-not $cmd) {
            if ($DryRun) {
                $results += New-AiCodingKitActionResult -Action $Action -KitId $entry.id -Ok $true -Status "skipped" -Message "Dry-run skipped undefined action in manifest."
            } else {
                $results += New-AiCodingKitActionResult -Action $Action -KitId $entry.id -Ok $false -Status "unsupported" -Message "Action not defined in manifest."
            }
            continue
        }
        if (($Action -eq "test" -or $Action -eq "verify") -and $Profile -eq "Smoke") {
            $results += Invoke-AiCodingKitSmokeTest -RepoRoot $root -Kit $entry -Action $Action
            continue
        }
        $results += Invoke-AiCodingKitCommand -RepoRoot $root -Kit $entry -Action $Action -CommandDefinition $cmd -Json:$Json -DryRun:$DryRun -Zip:$Zip -TimeoutSeconds $timeoutSeconds
    }

    $failed = @($results | Where-Object { -not $_.ok })
    $skipped = @($results | Where-Object { $_.status -eq "unsupported" -or $_.status -eq "skipped" })
    $bundle = $null
    if ($Action -eq "export" -and $All -and $Zip -and $failed.Count -eq 0) {
        $bundle = Export-AiCodingKitBundle -RepoRoot $root -KitPackageResults $results -DryRun:$DryRun -Zip:$Zip
    }
    $overallOk = ($failed.Count -eq 0) -and (($null -eq $bundle) -or $bundle.ok)

    return [pscustomobject]@{
        schemaVersion = 2
        action = $Action
        mode = $(if ($All) { "all" } else { "kit" })
        profile = $(if ($Action -eq "test" -or $Action -eq "verify") { $Profile } else { $null })
        ok = $overallOk
        summary = @{
            total = $results.Count
            ok = @($results | Where-Object { $_.ok }).Count
            failed = $failed.Count
            skipped = $skipped.Count
        }
        kits = $results
        bundle = $bundle
    }
}

Export-ModuleMember -Function Invoke-AiCodingKitAction, Invoke-AiCodingKitCommand
