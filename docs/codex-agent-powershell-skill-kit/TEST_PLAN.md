# Codex Agent PowerShell Skill Kit Test Plan

## Purpose

Validate that the kit prevents high-frequency PowerShell agent failures:

- wrong PowerShell runtime assumptions;
- missing AST syntax validation;
- Bash/Linux command leakage;
- unsafe destructive operations;
- complex one-line commands that are hard to approve/debug;
- missing PSScriptAnalyzer gate;
- unsafe config overwrite behavior;
- AiCoding repo integration drift.

## Prerequisites

Run from AiCoding repo root.

```powershell
pwsh -NoProfile -Command '$PSVersionTable.PSVersion; $PSVersionTable.PSEdition'
```

Expected:

- Major version is `7` or newer.
- Edition is `Core`.

## Install / Update

Use the current package path and installation script for the checked-out kit. Do not install from a version-stamped temporary directory.

Expected after install:

- `.agents/skills/codex-agent-powershell-skill-kit/SKILL.md` exists when the runtime mirror is materialized.
- `config/codex-agent-powershell-skill-kit.json` exists.
- `scripts/verify-codex-agent-powershell-skill-kit.ps1` exists.
- `dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/.codex-plugin/plugin.json` exists when the package is present.
- `.codex-agent-powershell-skill-kit/install-state.json` exists after installation.

## Verification Gate

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-codex-agent-powershell-skill-kit.ps1 -InstallMissingTools
```

Expected:

- Runtime gate passes.
- AST gate passes on kit scripts.
- Safety gate passes on kit scripts.
- PSScriptAnalyzer is installed or installed automatically with `-InstallMissingTools`.
- PSScriptAnalyzer gate passes or reports actionable diagnostics.

## Test Cases

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-codex-agent-powershell-skill-kit.ps1 -InstallMissingTools
```

Expected:

- `good/Valid-*.ps1` passes all gates.
- `bad/Syntax-MissingBrace.ps1` fails AST gate.
- `bad/Linux-Aliases.ps1` fails safety gate.
- `bad/Unsafe-RemoveItem.ps1` fails safety gate.
- Rewrite examples produce a plan and do not execute destructive actions.

## Manual Smoke Tests

### AST Syntax Failure

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\.agents\skills\codex-agent-powershell-skill-kit\tools\Invoke-PowerShellAstGate.ps1 -ScriptDefinition 'if ($true) { Write-Output "missing"'
```

Expected: fails with parse error.

### Bash Leakage Rewrite

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\.agents\skills\codex-agent-powershell-skill-kit\tools\Invoke-SafeRewritePlan.ps1 -Command 'ls -la | grep test && rm -rf temp' -Format Markdown
```

Expected: outputs rewrite plan. Does not execute command.

### PSScriptAnalyzer Gate

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\.agents\skills\codex-agent-powershell-skill-kit\tools\Invoke-PSScriptAnalyzerGate.ps1 -Path .\.agents\skills\codex-agent-powershell-skill-kit\tests\cases\good -Recurse
```

Expected: passes or only emits allowed warnings depending on config.

## AiCoding Integration Checks

```powershell
Test-Path .\.agents\skills\codex-agent-powershell-skill-kit\SKILL.md
Test-Path .\config\codex-agent-powershell-skill-kit.json
Test-Path .\dist\codex-agent-powershell-skill-kit\plugins\AiCodingPowerShellSkillKit\.codex-plugin\plugin.json
Test-Path .\.agents\plugins\marketplace.json
```

Expected: all required current paths are present for the selected install mode.

## Existing AiCoding Verification

```powershell
bin\aicoding.exe smoke --json
bin\aicoding.exe doctor pwsh-budget --json
```

Expected: AiCoding default Go gate still verifies, and PowerShell remains inside specialty boundaries.

## Rollback

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\scripts\uninstall-codex-agent-powershell-skill-kit.ps1 -RepoRoot .
```

Expected:

- Skill directory is removed.
- Dist directory is removed.
- Config sidecar is removed.
- Marketplace entry is removed or disabled.
- `.codex-agent-powershell-skill-kit/install-state.json` is preserved only if `-KeepState` is used.

## Source Ownership Verification

After install, verify that AiCoding did not become canonical owner of the skill source:

```powershell
Test-Path .\dist\codex-agent-powershell-skill-kit\plugins\AiCodingPowerShellSkillKit\skills\codex-agent-powershell-skill-kit\SKILL.md
Test-Path .\.agents\skills\codex-agent-powershell-skill-kit\RUNTIME_MIRROR_NOTICE.md
Get-Content .\.agents\skills\codex-agent-powershell-skill-kit\.runtime-mirror.json -Raw | ConvertFrom-Json
```

Expected:

- packaged skill path exists under `dist/`;
- repo-scoped `.agents/skills/...` exists only as runtime mirror;
- `.runtime-mirror.json` contains `canonicalOwnedByAiCoding: false`.