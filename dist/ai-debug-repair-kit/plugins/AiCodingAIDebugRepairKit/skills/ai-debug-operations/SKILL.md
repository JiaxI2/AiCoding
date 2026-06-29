---
name: ai-debug-operations
description: >
  Use this skill for generic debug operation lifecycle, including session
  setup, JSON CLI execution, evidence capture, capability/profile checks,
  and report generation. Do not use it for domain root-cause analysis or
  automatic code repair.
---

# AI Debug Operations Skill

This skill governs safe, auditable use of debug and repair CLI tools.

## Core rule

Use deterministic JSON CLI output:

```powershell
airepair doctor --output json
```

## Responsibilities

- Confirm workspace and profile.
- Run CLI commands in JSON mode.
- Capture stdout/stderr and exit code.
- Save evidence paths.
- Check explicit PASS/FAIL from a test runner.
- Record side effects.
- Stop on policy denial.

## Boundaries

This skill must not:

- Build a business hypothesis.
- Infer embedded root cause.
- Tune FOC/PID parameters.
- Run unbounded repair loops.
- Flash hardware without a dedicated HIL/flash policy.
- Treat AI text as a passing test.

If the user asks for automatic repair, hand off to `ai-debug-repair-loop`.
