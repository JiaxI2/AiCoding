# PS7-first Rule

Use `pwsh` by default.

Required runtime check:

```powershell
$PSVersionTable.PSVersion
$PSVersionTable.PSEdition
```

Reject assumptions based on Windows PowerShell 5.1 unless explicitly detected.
