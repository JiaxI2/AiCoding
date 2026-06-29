# GitHub Deployment

## As a standalone repository

1. Commit this kit to a GitHub repository.
2. Clone it on a Windows machine.
3. Run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/install-agent-patch-kit.ps1 -InstallMissing -DeployScope user -Agent both
```

## As an AiCoding project integration

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File integrations/aicoding/install-to-aicoding.ps1 -AiCodingRoot C:\path	o\AiCoding -Mode repo-skill
```

## As an AiCoding marketplace sidecar

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File integrations/aicoding/package-marketplace.ps1 -AiCodingRoot C:\path	o\AiCoding
```

This creates:

```text
AiCoding/dist/agent-patch-kit/plugins/AiCodingAgentPatch
AiCoding/.agents/plugins/agent-patch-marketplace.json
```

Use `-Merge` only when you intentionally want to merge the plugin entry into `.agents/plugins/marketplace.json`.
