# AiCoding Marketplace Packaging

Run:

```powershell
apatch package aicoding-plugin --out dist/agent-patch-kit --zip
```

Generated layout:

```text
dist/agent-patch-kit/
├─ marketplace.agent-patch.json
└─ plugins/
   └─ AiCodingAgentPatch/
      ├─ .codex-plugin/plugin.json
      ├─ skills/aicoding-agent-patch-kit/SKILL.md
      └─ assets/
```

AiCoding default integration should use the sidecar marketplace file first. Merge into the main AiCoding marketplace only with explicit intent.
