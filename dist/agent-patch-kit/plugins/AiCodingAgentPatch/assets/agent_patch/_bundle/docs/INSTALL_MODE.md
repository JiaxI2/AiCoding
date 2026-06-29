# Agent Patch Kit install mode

Agent Patch Kit v0.2.2 fixes the v0.2.1 editable-install problem.

## User install, default

Use the normal install script without `-Dev`:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1
```

This performs a non-editable install:

```powershell
python -m pip install --force-reinstall .
```

After validation, the original zip file and extracted source directory can be deleted.

## Developer install

Use this only when developing Agent Patch Kit itself:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -Dev
```

This performs an editable install and the source directory must not be deleted.

## Check install health

```powershell
apatch install doctor
apatch doctor
apatch brief --format md
apatch state status
```

A healthy user install reports:

```text
install_mode: non-editable / user mode
status: OK; original zip/extracted directory can be deleted
bundle_assets: OK
```

## Repair broken editable install

Run from the v0.2.2 extracted root:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\repair-agent-patch-kit.ps1
```

For project deployment:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\repair-agent-patch-kit.ps1 `
  -DeployScope project `
  -ProjectRoot <CURRENT_REPO_ROOT> `
  -Agent both `
  -WriteAgentsSnippet
```
