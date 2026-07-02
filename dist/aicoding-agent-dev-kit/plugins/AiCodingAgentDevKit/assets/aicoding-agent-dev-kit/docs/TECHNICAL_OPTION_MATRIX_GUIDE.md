# Technical Option Matrix Guide

Use this guide when the implementation path is not obvious.

## Required Columns

| Column | Purpose |
|---|---|
| Option ID | Stable ID such as OPT-001 |
| Name | Short option name |
| Architecture | Structural description |
| Flow | Runtime or data flow |
| Pros | What this option optimizes |
| Cons | What it sacrifices |
| Risks | Failure modes |
| Validation | How to prove it works |
| Compatibility | Compatibility and migration cost |
| Effort | Rough implementation cost |
| Recommended When | Conditions where this option wins |

## Domain-neutral Rule

The reusable Kit must not include product-specific or application-specific examples.

Use generic placeholders in the Kit:

```text
Option A
Option B
Option C
```

Concrete examples should be created by the Agent in the target repository after reading that repository's context.
