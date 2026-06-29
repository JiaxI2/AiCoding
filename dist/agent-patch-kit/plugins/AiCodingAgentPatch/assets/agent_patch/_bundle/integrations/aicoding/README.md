# AiCoding Integration

Default safe mode:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File integrations/aicoding/install-to-aicoding.ps1 -AiCodingRoot C:\path	o\AiCoding -Mode repo-skill
```

Marketplace sidecar package:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File integrations/aicoding/package-marketplace.ps1 -AiCodingRoot C:\path	o\AiCoding
```

Merge into `.agents/plugins/marketplace.json` only when you want AiCoding to expose this as a local plugin entry:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File integrations/aicoding/package-marketplace.ps1 -AiCodingRoot C:\path	o\AiCoding -Merge
```

This integration does not edit `CodingKit/agents/skills/plugins/AiCoding` by default, because AiCoding treats `CodingKit/agents/skills` as an external submodule release dependency.
