Import-Module (Join-Path $PSScriptRoot "AiCoding.KitRegistry.psm1") -Force

function Resolve-AiCodingPackageToken {
    param(
        [Parameter(Mandatory=$true)]$Value,
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$Kit
    )

    $text = [string]$Value
    $text = $text.Replace('${repoRoot}', $RepoRoot)
    $text = $text.Replace('${kitId}', [string]$Kit.id)
    $text = $text.Replace('${version}', [string]$Kit.manifest.version)
    return $text
}

function ConvertTo-AiCodingPackageRelativePath {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)][string]$Path
    )

    $root = (Resolve-Path -LiteralPath $RepoRoot).Path.TrimEnd('\')
    $full = (Resolve-Path -LiteralPath $Path).Path
    if (-not $full.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Path is outside repository: $full"
    }
    return $full.Substring($root.Length).TrimStart('\') -replace '\\', '/'
}

function ConvertTo-AiCodingGlobRegex {
    param([Parameter(Mandatory=$true)][string]$Pattern)

    $normalized = $Pattern -replace '\\', '/'
    if ($normalized.StartsWith('./')) { $normalized = $normalized.Substring(2) }
    $regex = [regex]::Escape($normalized)
    $regex = $regex.Replace('\*\*/', '(.*/)?')
    $regex = $regex.Replace('\*\*', '.*')
    $regex = $regex.Replace('\*', '[^/]*')
    $regex = $regex.Replace('\?', '[^/]')
    return "^$regex$"
}

function Test-AiCodingPackageGlobMatch {
    param(
        [Parameter(Mandatory=$true)][string]$RelativePath,
        [Parameter(Mandatory=$true)][string]$Pattern
    )

    $rel = $RelativePath -replace '\\', '/'
    $regex = ConvertTo-AiCodingGlobRegex -Pattern $Pattern
    return [regex]::IsMatch($rel, $regex, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)
}

function Get-AiCodingGitInfo {
    param([Parameter(Mandatory=$true)][string]$RepoRoot)

    $commit = ""
    $branch = ""
    try { $commit = (& git -C $RepoRoot rev-parse HEAD 2>$null) } catch { $commit = "" }
    try { $branch = (& git -C $RepoRoot branch --show-current 2>$null) } catch { $branch = "" }
    if ([string]::IsNullOrWhiteSpace($commit)) { $commit = "unknown" }
    if ([string]::IsNullOrWhiteSpace($branch)) { $branch = "unknown" }
    return [pscustomobject]@{
        commit = [string]$commit
        branch = [string]$branch
    }
}

function Remove-AiCodingPackageGeneratedPath {
    [CmdletBinding(SupportsShouldProcess=$true)]
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)][string]$Path,
        [switch]$Recurse
    )

    $packageRoot = [System.IO.Path]::GetFullPath((Join-Path $RepoRoot ".aicoding\packages")).TrimEnd('\') + '\'
    $target = [System.IO.Path]::GetFullPath($Path)
    if (-not $target.StartsWith($packageRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove path outside package root: $target"
    }

    if (Test-Path -LiteralPath $target) {
        if ($PSCmdlet.ShouldProcess($target, "Remove generated package path")) {
            Remove-Item -LiteralPath $target -Force -Recurse:$Recurse
        }
    }
}

function Get-AiCodingPackageFiles {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)][string[]]$Include,
        [string[]]$Exclude = @()
    )

    $filesByRelativePath = @{}
    $missingIncludes = @()

    foreach ($pattern in $Include) {
        $normalized = $pattern -replace '\\', '/'
        if ($normalized.StartsWith('./')) { $normalized = $normalized.Substring(2) }
        $matched = @()

        if ($normalized.EndsWith('/**')) {
            $baseRel = $normalized.Substring(0, $normalized.Length - 3)
            $basePath = Resolve-AiCodingKitPath -RepoRoot $RepoRoot -RelativePath $baseRel
            if (Test-Path -LiteralPath $basePath -PathType Container) {
                $matched = @(Get-ChildItem -LiteralPath $basePath -Recurse -File -Force)
            }
        } elseif ($normalized.Contains('**')) {
            $regex = ConvertTo-AiCodingGlobRegex -Pattern $normalized
            $matched = @(Get-ChildItem -LiteralPath $RepoRoot -Recurse -File -Force | Where-Object {
                $rel = ConvertTo-AiCodingPackageRelativePath -RepoRoot $RepoRoot -Path $_.FullName
                [regex]::IsMatch($rel, $regex, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)
            })
        } elseif ($normalized.Contains('*') -or $normalized.Contains('?')) {
            $wildcardPath = Join-Path $RepoRoot ($normalized -replace '/', '\')
            $matchedItems = @(Get-ChildItem -Path $wildcardPath -Force -ErrorAction SilentlyContinue)
            foreach ($item in $matchedItems) {
                if ($item.PSIsContainer) {
                    $matched += @(Get-ChildItem -LiteralPath $item.FullName -Recurse -File -Force)
                } else {
                    $matched += $item
                }
            }
        } else {
            $path = Resolve-AiCodingKitPath -RepoRoot $RepoRoot -RelativePath $normalized
            if (Test-Path -LiteralPath $path -PathType Leaf) {
                $matched = @(Get-Item -LiteralPath $path)
            } elseif (Test-Path -LiteralPath $path -PathType Container) {
                $matched = @(Get-ChildItem -LiteralPath $path -Recurse -File -Force)
            }
        }

        if ($matched.Count -eq 0) {
            $missingIncludes += $pattern
            continue
        }

        foreach ($file in $matched) {
            $rel = ConvertTo-AiCodingPackageRelativePath -RepoRoot $RepoRoot -Path $file.FullName
            $filesByRelativePath[$rel] = $file.FullName
        }
    }

    $included = @()
    $excluded = @()
    foreach ($rel in ($filesByRelativePath.Keys | Sort-Object)) {
        $isExcluded = $false
        foreach ($excludePattern in $Exclude) {
            if (Test-AiCodingPackageGlobMatch -RelativePath $rel -Pattern $excludePattern) {
                $isExcluded = $true
                break
            }
        }

        if ($isExcluded) {
            $excluded += $rel
        } else {
            $included += [pscustomobject]@{
                relativePath = $rel
                fullPath = $filesByRelativePath[$rel]
            }
        }
    }

    return [pscustomobject]@{
        included = $included
        excluded = $excluded
        missingIncludes = $missingIncludes
    }
}

function New-AiCodingPackageBuildInfo {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$Kit,
        [Parameter(Mandatory=$true)][string]$PackageFile,
        [Parameter(Mandatory=$true)][string]$Sha256File,
        [Parameter(Mandatory=$true)][int]$IncludedFilesCount
    )

    $git = Get-AiCodingGitInfo -RepoRoot $RepoRoot
    return [ordered]@{
        schemaVersion = 1
        kitId = [string]$Kit.id
        kitName = [string]$Kit.manifest.name
        version = [string]$Kit.manifest.version
        kind = @($Kit.manifest.kind)
        manifestPath = [string]$Kit.manifestRelativePath
        registryPath = "config/kit-registry.json"
        gitCommit = $git.commit
        gitBranch = $git.branch
        createdAt = (Get-Date).ToUniversalTime().ToString("o")
        includedFilesCount = $IncludedFilesCount
        packageFile = [System.IO.Path]::GetFileName($PackageFile)
        sha256File = [System.IO.Path]::GetFileName($Sha256File)
    }
}

function Export-AiCodingKit {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$Kit,
        [Parameter(Mandatory=$true)]$CommandDefinition,
        [switch]$DryRun,
        [switch]$Zip
    )

    if (-not $Zip) {
        return [pscustomobject]@{
            ok = $false
            status = "unsupported"
            message = "Export requires -Zip in Phase 1.6."
            data = $null
        }
    }

    $packageRoot = Join-Path $RepoRoot ".aicoding\packages"
    $include = @($CommandDefinition.include)
    $exclude = @($CommandDefinition.exclude)
    $outputName = Resolve-AiCodingPackageToken -Value $CommandDefinition.outputName -RepoRoot $RepoRoot -Kit $Kit
    $packageFile = Join-Path $packageRoot $outputName
    $sha256File = "$packageFile.sha256"
    $buildInfoFile = Join-Path $packageRoot ("{0}.BUILDINFO.json" -f [System.IO.Path]::GetFileNameWithoutExtension($outputName))
    $files = Get-AiCodingPackageFiles -RepoRoot $RepoRoot -Include $include -Exclude $exclude
    $wouldWrite = @($packageFile, $sha256File, $buildInfoFile)

    if ($DryRun) {
        return [pscustomobject]@{
            ok = $true
            status = "dry-run"
            message = "export dry-run"
            data = [ordered]@{
                zip = $true
                output = $packageFile
                packageFile = $packageFile
                sha256File = $sha256File
                buildInfoFile = $buildInfoFile
                include = $include
                exclude = $exclude
                wouldInclude = @($files.included.relativePath)
                wouldExclude = @($files.excluded)
                wouldWrite = $wouldWrite
                missingIncludes = @($files.missingIncludes)
            }
        }
    }

    if ($files.missingIncludes.Count -gt 0) {
        return [pscustomobject]@{
            ok = $false
            status = "missing"
            message = "Export include paths did not match files."
            data = [ordered]@{
                missingIncludes = @($files.missingIncludes)
            }
        }
    }

    New-Item -ItemType Directory -Path $packageRoot -Force | Out-Null
    $stageRoot = Join-Path $packageRoot (".staging\{0}-{1}" -f $Kit.id, ([guid]::NewGuid().ToString("N")))
    New-Item -ItemType Directory -Path $stageRoot -Force | Out-Null

    try {
        foreach ($file in @($files.included)) {
            $destination = Join-Path $stageRoot ($file.relativePath -replace '/', '\')
            $destinationDirectory = Split-Path -Parent $destination
            New-Item -ItemType Directory -Path $destinationDirectory -Force | Out-Null
            Copy-Item -LiteralPath $file.fullPath -Destination $destination -Force
        }

        $buildInfo = New-AiCodingPackageBuildInfo -RepoRoot $RepoRoot -Kit $Kit -PackageFile $packageFile -Sha256File $sha256File -IncludedFilesCount @($files.included).Count
        $buildInfoJson = $buildInfo | ConvertTo-Json -Depth 20
        Set-Content -LiteralPath (Join-Path $stageRoot "BUILDINFO.json") -Value $buildInfoJson -Encoding UTF8

        Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $packageFile
        Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $sha256File
        Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $buildInfoFile

        Compress-Archive -Path (Join-Path $stageRoot "*") -DestinationPath $packageFile -Force
        $hash = Get-FileHash -LiteralPath $packageFile -Algorithm SHA256
        Set-Content -LiteralPath $sha256File -Value ("{0}  {1}" -f $hash.Hash.ToLowerInvariant(), [System.IO.Path]::GetFileName($packageFile)) -Encoding ASCII
        Set-Content -LiteralPath $buildInfoFile -Value $buildInfoJson -Encoding UTF8

        return [pscustomobject]@{
            ok = $true
            status = "ok"
            message = "export package"
            data = [ordered]@{
                zip = $true
                packageFile = $packageFile
                sha256File = $sha256File
                buildInfoFile = $buildInfoFile
                sha256 = $hash.Hash.ToLowerInvariant()
                includedFilesCount = @($files.included).Count
                excludedFilesCount = @($files.excluded).Count
                include = $include
                exclude = $exclude
            }
        }
    }
    finally {
        if (Test-Path -LiteralPath $stageRoot) {
            Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $stageRoot -Recurse
        }
    }
}

function Export-AiCodingKitBundle {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)]$KitPackageResults,
        [switch]$DryRun,
        [switch]$Zip
    )

    if (-not $Zip) {
        return [pscustomobject]@{
            ok = $false
            status = "unsupported"
            message = "Bundle export requires -Zip."
            data = $null
        }
    }

    $packageRoot = Join-Path $RepoRoot ".aicoding\packages"
    $timestamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd-HHmmss")
    $bundleName = "aicoding-kit-bundle-$timestamp.zip"
    $bundleFile = Join-Path $packageRoot $bundleName
    $sha256File = "$bundleFile.sha256"
    $buildInfoFile = Join-Path $packageRoot ("{0}.BUILDINFO.json" -f [System.IO.Path]::GetFileNameWithoutExtension($bundleName))
    $packageFiles = @($KitPackageResults | Where-Object { $_.ok -and $_.data.packageFile } | ForEach-Object { [string]$_.data.packageFile })

    if ($DryRun) {
        return [pscustomobject]@{
            ok = $true
            status = "dry-run"
            message = "bundle dry-run"
            data = [ordered]@{
                packageFile = $bundleFile
                sha256File = $sha256File
                buildInfoFile = $buildInfoFile
                wouldInclude = @(
                    "registry/kit-registry.json",
                    "manifests/config/kits/*.json",
                    "packages/*.zip",
                    "SHA256SUMS.txt",
                    "BUILDINFO.json"
                )
                wouldWrite = @($bundleFile, $sha256File, $buildInfoFile)
                packages = $packageFiles
            }
        }
    }

    New-Item -ItemType Directory -Path $packageRoot -Force | Out-Null
    $stageRoot = Join-Path $packageRoot (".staging\bundle-{0}" -f ([guid]::NewGuid().ToString("N")))
    New-Item -ItemType Directory -Path $stageRoot -Force | Out-Null

    try {
        $registryDestination = Join-Path $stageRoot "registry\kit-registry.json"
        New-Item -ItemType Directory -Path (Split-Path -Parent $registryDestination) -Force | Out-Null
        Copy-Item -LiteralPath (Join-Path $RepoRoot "config\kit-registry.json") -Destination $registryDestination -Force

        $manifestDestinationRoot = Join-Path $stageRoot "manifests\config\kits"
        New-Item -ItemType Directory -Path $manifestDestinationRoot -Force | Out-Null
        Get-ChildItem -LiteralPath (Join-Path $RepoRoot "config\kits") -Filter "*.json" -File | ForEach-Object {
            Copy-Item -LiteralPath $_.FullName -Destination (Join-Path $manifestDestinationRoot $_.Name) -Force
        }

        $packageDestinationRoot = Join-Path $stageRoot "packages"
        New-Item -ItemType Directory -Path $packageDestinationRoot -Force | Out-Null
        $sumLines = @()
        foreach ($packageFile in $packageFiles) {
            if (-not (Test-Path -LiteralPath $packageFile -PathType Leaf)) {
                throw "Missing package file for bundle: $packageFile"
            }
            $packageName = [System.IO.Path]::GetFileName($packageFile)
            Copy-Item -LiteralPath $packageFile -Destination (Join-Path $packageDestinationRoot $packageName) -Force
            $hash = Get-FileHash -LiteralPath $packageFile -Algorithm SHA256
            $sumLines += ("{0}  packages/{1}" -f $hash.Hash.ToLowerInvariant(), $packageName)
        }
        Set-Content -LiteralPath (Join-Path $stageRoot "SHA256SUMS.txt") -Value $sumLines -Encoding ASCII

        $git = Get-AiCodingGitInfo -RepoRoot $RepoRoot
        $bundleBuildInfo = [ordered]@{
            schemaVersion = 1
            bundleId = "aicoding-kit-bundle"
            version = $timestamp
            registryPath = "config/kit-registry.json"
            manifestsPath = "config/kits"
            gitCommit = $git.commit
            gitBranch = $git.branch
            createdAt = (Get-Date).ToUniversalTime().ToString("o")
            packageCount = $packageFiles.Count
            packageFile = [System.IO.Path]::GetFileName($bundleFile)
            sha256File = [System.IO.Path]::GetFileName($sha256File)
            packages = @($packageFiles | ForEach-Object { [System.IO.Path]::GetFileName($_) })
        }
        $bundleBuildInfoJson = $bundleBuildInfo | ConvertTo-Json -Depth 20
        Set-Content -LiteralPath (Join-Path $stageRoot "BUILDINFO.json") -Value $bundleBuildInfoJson -Encoding UTF8

        Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $bundleFile
        Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $sha256File
        Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $buildInfoFile

        Compress-Archive -Path (Join-Path $stageRoot "*") -DestinationPath $bundleFile -Force
        $bundleHash = Get-FileHash -LiteralPath $bundleFile -Algorithm SHA256
        Set-Content -LiteralPath $sha256File -Value ("{0}  {1}" -f $bundleHash.Hash.ToLowerInvariant(), [System.IO.Path]::GetFileName($bundleFile)) -Encoding ASCII
        Set-Content -LiteralPath $buildInfoFile -Value $bundleBuildInfoJson -Encoding UTF8

        return [pscustomobject]@{
            ok = $true
            status = "ok"
            message = "bundle package"
            data = [ordered]@{
                packageFile = $bundleFile
                sha256File = $sha256File
                buildInfoFile = $buildInfoFile
                sha256 = $bundleHash.Hash.ToLowerInvariant()
                packageCount = $packageFiles.Count
                packages = @($packageFiles | ForEach-Object { [System.IO.Path]::GetFileName($_) })
            }
        }
    }
    finally {
        if (Test-Path -LiteralPath $stageRoot) {
            Remove-AiCodingPackageGeneratedPath -RepoRoot $RepoRoot -Path $stageRoot -Recurse
        }
    }
}

Export-ModuleMember -Function Export-AiCodingKit, Export-AiCodingKitBundle
