param(
  [string]$Out = 'dist/agent-patch-kit',
  [switch]$Zip
)
$ErrorActionPreference = 'Stop'
$args = @('package','aicoding-plugin','--out',$Out)
if ($Zip) { $args += '--zip' }
apatch @args
