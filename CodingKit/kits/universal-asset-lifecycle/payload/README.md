# Universal Asset Lifecycle Kit

This payload is a reusable template. Copy `asset-template.json` into any Kit, Skill, MCP, template, ruleset or profile package, then place distributable files under `payload/`.

Installation modes:
- `managed`: immutable upstream-owned payload with safe replacement and rollback.
- `editable`: user-owned copy; updates should be imported explicitly and reviewed.

Configuration precedence:
`defaults < config/assets < UserCfg/assets < .aicoding/local/assets < CLI`.
