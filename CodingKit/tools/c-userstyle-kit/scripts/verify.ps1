[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = Split-Path -Parent $PSScriptRoot
$configPath = Join-Path $root 'examples/c-kit.json'
$generatedPath = Join-Path $root 'generated-demo'
$advancedPath = Join-Path $generatedPath 'advanced'
$buildPath = Join-Path $root 'build/verify'
$pdfPath = Join-Path $root 'references/huawei-c-language-programming-standard-dkba-2826-2011-5.pdf'
$markdownPath = Join-Path $root 'references/huawei-c-language-programming-standard-dkba-2826-2011-5.md'

function Get-RequiredCommandPath {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $command = Get-Command -Name $Name -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $command) {
        throw "缺少验证所需命令：$Name"
    }
    return $command.Source
}

function Invoke-NativeCommand {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [Parameter()]
        [string[]]$ArgumentList = @()
    )

    & $FilePath @ArgumentList
    if ($LASTEXITCODE -ne 0) {
        throw "命令失败（exit=$LASTEXITCODE）：$FilePath $($ArgumentList -join ' ')"
    }
}

$go = Get-RequiredCommandPath -Name 'go'
$python = Get-RequiredCommandPath -Name 'python'
$uv = Get-RequiredCommandPath -Name 'uv'
$gcc = Get-RequiredCommandPath -Name 'gcc'
$gxx = Get-RequiredCommandPath -Name 'g++'
$clang = Get-RequiredCommandPath -Name 'clang'
$clangxx = Get-RequiredCommandPath -Name 'clang++'

Set-Location -LiteralPath $root
New-Item -ItemType Directory -Force -Path $buildPath | Out-Null

$config = Get-Content -LiteralPath $configPath -Raw | ConvertFrom-Json
$gccFlags = [string[]]$config.gates.gcc.flags
$clangFlags = [string[]]$config.gates.clang.flags
$headerCFlags = [string[]]$config.gates.headerC.flags
$headerCxxFlags = [string[]]$config.gates.headerCxx.flags

Write-Output '[1/9] Go 单元测试（含 lint 正例、负例与生成器测试）'
Invoke-NativeCommand -FilePath $go -ArgumentList @('test', './...')

Write-Output '[2/9] PDF 与 Markdown 61 页、139 条款完整性'
$pdfVerifyArguments = @(
    'run',
    '--with',
    'pdfplumber==0.11.7',
    'python',
    'tools/pdf-reference/verify_reference.py',
    '--pdf',
    $pdfPath,
    '--markdown',
    $markdownPath
)
Invoke-NativeCommand -FilePath $uv -ArgumentList $pdfVerifyArguments

Write-Output '[3/9] 规则目录 139/139 确定性检查'
Invoke-NativeCommand -FilePath $python -ArgumentList @(
    '-X',
    'utf8',
    'tools/rules/build_rule_catalog.py',
    '--check'
)

Write-Output '[4/9] c-kit、snippets 与规则目录 JSON Schema'
Invoke-NativeCommand -FilePath $python -ArgumentList @(
    '-X',
    'utf8',
    'tools/json/validate_json_contracts.py'
)

Write-Output '[5/9] 确定性生成简单入口和公开高级规则覆盖样例'
Invoke-NativeCommand -FilePath $go -ArgumentList @(
    'run',
    './cmd/cstylekit',
    'demo',
    '--config',
    $configPath,
    '--out',
    $generatedPath
)

$generatedFiles = @(
    (Join-Path $generatedPath 'demo.c'),
    (Join-Path $generatedPath 'demo.h'),
    (Join-Path $advancedPath 'state_machine.c'),
    (Join-Path $advancedPath 'state_machine.h'),
    (Join-Path $advancedPath 'protocol.c'),
    (Join-Path $advancedPath 'protocol.h'),
    (Join-Path $advancedPath 'fixed_pool.c'),
    (Join-Path $advancedPath 'fixed_pool.h'),
    (Join-Path $advancedPath 'tests/advanced_test.c')
)
$lintArguments = @('run', './cmd/cstylekit', 'lint', '--config', $configPath, '--scope', 'files')
foreach ($file in $generatedFiles) {
    $lintArguments += @('--file', $file)
}

Write-Output '[6/9] 黄金 demo lint'
Invoke-NativeCommand -FilePath $go -ArgumentList $lintArguments

$cSources = @(
    (Join-Path $generatedPath 'demo.c'),
    (Join-Path $advancedPath 'state_machine.c'),
    (Join-Path $advancedPath 'protocol.c'),
    (Join-Path $advancedPath 'fixed_pool.c'),
    (Join-Path $advancedPath 'tests/advanced_test.c')
)
$gccExecutable = Join-Path $buildPath 'demo_gcc.exe'
$gccArguments = @($gccFlags) + @(
    '-I',
    $generatedPath,
    '-I',
    $advancedPath,
    '-o',
    $gccExecutable
) + $cSources

Write-Output '[7/9] GCC 严格 C99 零告警并执行行为测试'
Invoke-NativeCommand -FilePath $gcc -ArgumentList $gccArguments
Invoke-NativeCommand -FilePath $gccExecutable

$clangVersion = (& $clang '--version' | Out-String)
if ($LASTEXITCODE -ne 0) {
    throw '无法读取 Clang 版本。'
}
$clangHostFlags = @()
if ($clangVersion -match 'Target:\s+riscv') {
    $mingwSysroot = 'C:/msys64/ucrt64'
    if (-not (Test-Path -LiteralPath $mingwSysroot)) {
        throw '当前 Clang 是 RISC-V 发行版，且未找到用于主机语法检查的 MinGW sysroot。'
    }
    $clangHostFlags = @('--target=x86_64-w64-windows-gnu', "--sysroot=$mingwSysroot")
}
$clangArguments = @($clangHostFlags) + @($clangFlags) + @(
    '-fsyntax-only',
    '-I',
    $generatedPath,
    '-I',
    $advancedPath
) + $cSources

Write-Output '[8/9] Clang 严格 C99 零告警'
Invoke-NativeCommand -FilePath $clang -ArgumentList $clangArguments

Write-Output '[9/9] 四个头文件分别通过 GCC/Clang C99 与 G++/Clang++ C++17'
$headers = @(
    @{Name = 'demo.h'; IncludePath = $generatedPath},
    @{Name = 'state_machine.h'; IncludePath = $advancedPath},
    @{Name = 'protocol.h'; IncludePath = $advancedPath},
    @{Name = 'fixed_pool.h'; IncludePath = $advancedPath}
)
foreach ($header in $headers) {
    $commonHeaderArguments = @(
        '-I',
        $header.IncludePath,
        '-fsyntax-only',
        '-include',
        $header.Name,
        'NUL'
    )
    Invoke-NativeCommand -FilePath $gcc -ArgumentList (
        @($headerCFlags) + @('-x', 'c') + $commonHeaderArguments
    )
    Invoke-NativeCommand -FilePath $gxx -ArgumentList (
        @($headerCxxFlags) + @('-x', 'c++') + $commonHeaderArguments
    )
    Invoke-NativeCommand -FilePath $clang -ArgumentList (
        @($clangHostFlags) + @($headerCFlags) + @('-x', 'c') + $commonHeaderArguments
    )
    Invoke-NativeCommand -FilePath $clangxx -ArgumentList (
        @($clangHostFlags) + @($headerCxxFlags) + @('-x', 'c++') + $commonHeaderArguments
    )
}

Write-Output '验证通过：PDF、139 条款、snippets、lint、GCC、Clang、四个头文件和行为测试。'
