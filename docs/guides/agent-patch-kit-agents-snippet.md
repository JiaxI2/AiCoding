## Agent Patch Kit Safety Rule

Before any text/code modification, use Agent Patch Kit.

Developer-only fast-read:

```powershell
apatch brief --format md
apatch state status
```

If effective state is disabled, do not apply edits unless the user explicitly asks to enable the kit or authorizes override.

Mandatory flow:

1. `apatch status`
2. `apatch scan "<target>" --fixed` unless regex is required
3. `apatch replace ... --preview` or `apatch ast ... --preview`
4. `apatch replace ... --apply` or `apatch ast ... --apply`
5. `apatch verify`
6. `apatch summary`

For Markdown changes, run:

```powershell
apatch links --mode offline --include-fragments full --input README.md --input README_CN.md --input README_EN.md --input CHANGELOG.md --input "docs/*.md" --input ".github/workflows/*.yml"
```

This is the default maintained-docs link check. Run `apatch links --mode offline --include-fragments full` for the full AiCoding-owned repository audit. The root `lychee.toml` excludes the read-only `CodingKit/agents/skills` source dependency and `CodingKit/tests/ai-debug-kit/_external` fixture repository; validate those Git submodules in their own source repositories. It also excludes the exact GBK-encoded `CodingKit/modules/common/vofa+/README.md` path because lychee accepts UTF-8 input only. First-party UTF-8 templates, generated assets, and fixtures remain in scope.

Do not use broad PowerShell regex replacement as the first choice.
