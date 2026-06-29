---
name: ai-debug-repair-loop
description: >
  Use this skill when the user asks Codex to run a bounded AI repair loop:
  inspect build/test failure evidence, make minimal source patches, run
  configured build/test commands, record attempts, and stop on PASS or the
  iteration limit. Requires profiles and must not auto-flash or auto-commit.
---

# AI Debug Repair Loop Skill

This skill provides bounded AI automatic repair.

It implements:

```text
AI reads failure evidence
→ proposes minimal patch
→ runs build
→ runs test
→ records attempt
→ repeats until PASS or max_iterations
→ outputs report for human review
```

## Mandatory first commands

```powershell
airepair doctor --output json
airepair profile validate --profile .ai-debug-repair\profiles\loop.safe.json --output json
airepair loop status --profile .ai-debug-repair\profiles\loop.safe.json --output json
```

If profiles are missing:

```powershell
airepair init --workspace . --output json
```

Then ask the user to confirm build/test commands before modifying code.

## Safety gates

1. Confirm `max_iterations` is finite.
2. Confirm `allowed_paths` and `forbidden_paths`.
3. Record git status before edits.
4. Do not edit forbidden paths.
5. Do not modify tests unless explicitly requested.
6. Do not weaken tests.
7. Do not run flash/reset/halt.
8. Do not commit or push automatically.
9. Do not claim PASS unless `airepair test run` returns `ok: true`.
10. Stop on ambiguous profiles, policy denial, or unsafe command.

## Standard workflow

Export context:

```powershell
airepair loop export-context --profile .ai-debug-repair\profiles\loop.safe.json --output json
```

For each iteration:

```powershell
airepair build run --profile .ai-debug-repair\profiles\build.json --output json
airepair test run --profile .ai-debug-repair\profiles\test.json --output json
```

Record result:

```powershell
airepair loop record-attempt --profile .ai-debug-repair\profiles\loop.safe.json --result fail --notes "<short reason>" --output json
```

Use `--result pass` only after the configured test runner returns PASS.

## Patch rules

- Prefer one logical change per attempt.
- Keep patches small.
- Stay inside `allowed_paths`.
- Avoid broad rewrites.
- Preserve bootloader, linker, flash, safety, security, and startup code unless explicitly authorized.
- Do not hardcode expected output just to pass a verifier.

## Completion report

End with:

- Iterations used.
- Files changed.
- Build result.
- Test result.
- Evidence paths.
- Whether human review is required.
- No automatic commit unless explicitly requested after review.
