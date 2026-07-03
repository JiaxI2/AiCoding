# Kit Install State

AiCoding v2.0 separates desired declarations from local runtime facts.

## Declared State

Kit manifests may declare state paths:

```json
{
  "state": {
    "root": ".aicoding/state/kits/aicoding-agent-dev-kit",
    "installState": ".aicoding/state/kits/aicoding-agent-dev-kit/install-state.json"
  }
}
```

These paths describe where runtime facts belong. They do not replace the existing install, update, or uninstall scripts in v2.0.

## Runtime State

Generated state lives under `.aicoding/state/` and is ignored by Git. Third-party and user-created skill installs write `.aicoding/state/skills/<skill-id>/install-state.json` with source, url, commit, license, installedAt, installedBy, files, trust, and enabled state.

## Rules

- State records are local facts, not source declarations.
- State files must not be committed.
- Install and uninstall ownership remains with the existing lifecycle scripts until a later state-based phase.
- Future safe uninstall work must use state ownership and must not delete unknown user-managed files.
