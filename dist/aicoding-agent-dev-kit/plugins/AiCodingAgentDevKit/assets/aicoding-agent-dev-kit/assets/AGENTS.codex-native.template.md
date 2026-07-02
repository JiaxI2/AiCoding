# AGENTS.md

## Agent Dev Kit Entry

1. Do not read the whole repository by default.
2. Run or request:
   `aicoding-agent-kit load --repo . --auto`
3. Read:
   `.agent-dev-kit/context/context-pack.md`
   `.agent-dev-kit/context/context-manifest.json`
4. Escalate context stage only with a reason.
5. Record only important human decisions or accepted/rejected Agent proposals in `.agent-memory/DECISIONS.md`.
6. Keep `.agent-memory/CURRENT.md` short and local.
7. Use TDD: Red -> Green -> Refactor -> Gate.
8. Before handoff or commit, run:
   `scripts/invoke-agent-quality-gate.ps1 -Mode pre-commit -Json`
