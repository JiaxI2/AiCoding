# Agent Patch Workflow

1. `apatch brief --format md` loads the short workflow when the full skill is too much context.
2. `apatch state status` confirms the kit is enabled.
3. `apatch status` confirms the working tree state.
4. `apatch scan` confirms the target count and file range.
5. List files before cross-file edits; keep one topic per patch.
6. `apatch replace --preview` or `apatch ast --preview` shows a non-writing preview.
7. Literal replacements must have one intended match. Stop on zero or multiple matches.
8. `apatch replace --apply` or `apatch ast --apply` creates a transaction snapshot, then writes changes.
9. Run `git diff --check` and the relevant Fast Path verification after patching.
10. `apatch verify` runs optional old/new counts, Taskfile, and link validation.
11. `apatch summary` prints changed files and diff stat.

Do not skip preview for multi-file edits.
Do not use patch workflows for release/tag remote operations, hardware actions, or mixed-theme rewrites.
Do not move or delete files without listing them and waiting for confirmation.

Rollback boundaries must be explicit: transaction rollback, `git restore <file>`, `git restore --staged <file>`, or `git reset --soft HEAD~1`.
