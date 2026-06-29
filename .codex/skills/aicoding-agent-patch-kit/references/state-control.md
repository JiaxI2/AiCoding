# State control

```powershell
apatch state status
apatch state where
apatch state enable --scope system|user|project
apatch state disable --scope system|user|project
```

Effective enablement requires system, user, and project scopes to all be enabled. Missing state files default to enabled.
