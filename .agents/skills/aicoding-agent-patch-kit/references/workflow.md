# Agent Patch Workflow

1. `apatch status` confirms the working tree state.
2. `apatch scan` confirms the target count and file range.
3. `apatch replace --preview` or `apatch ast --preview` shows a non-writing preview.
4. `apatch replace --apply` or `apatch ast --apply` creates a transaction snapshot, then writes changes.
5. `apatch verify` runs `git diff --check` plus optional old/new counts, Taskfile, and link validation.
6. `apatch summary` prints changed files and diff stat.

Do not skip preview for multi-file edits.
