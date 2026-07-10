# Implementation Plan: Plan Mode Overlay Integration

Plan Status: Approved

## Scope

Integrate `F:\Study\AI\aicoding-agent-dev-kit-plan-mode-overlay-v0.4` into AiCoding on `main` without creating a branch and without modifying `CodingKit/agents/skills` or plugin cache state.

## Steps

1. Copy the overlay repo payload into the AiCoding repository.
2. Adapt hook registration to the existing `config/hooks-registry.json` model.
3. Add a repository-level agent hook bridge and Agent Engineering Foundation compatibility verifier.
4. Update README, README_CN, README_EN, kit manifest, and CHANGELOG.
5. Run overlay, hook, kit, documentation, governance, and Git validation.

## Verification

- `pwsh tools\specialty\verify-agent-dev-kit-plan-mode.ps1 -Json`
- `pwsh tools\specialty\hooks\aef\plan-mode-gate.ps1 -Event manual -Mode warn -Json`
- `pwsh tools\specialty\hooks\aef\spec-artifact-gate.ps1 -Event manual -Mode warn -Json`
- `pwsh tools\specialty\verify-agent-engineering-foundation.ps1 -Json`
- `bin\aicoding.exe verify hooks --json`
- `git diff --check`
