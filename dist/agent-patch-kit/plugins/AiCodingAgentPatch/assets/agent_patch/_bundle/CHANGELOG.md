# Changelog

## v0.2.2

### Fixed

- Changed default CLI install mode from editable/developer install to non-editable user install.
- Fixed the issue where deleting the extracted kit source directory breaks `apatch` with `ModuleNotFoundError: No module named 'agent_patch'`.
- Bundled Skill, docs, scripts, config and plugin templates into the Python package so `apatch deploy` and `apatch package` still work after source deletion.

### Added

- `apatch install doctor` to verify whether the CLI install is self-contained and non-editable.
- `apatch install repair` to print safe repair commands.
- `scripts/repair-agent-patch-kit.ps1` to uninstall broken editable installs and reinstall v0.2.2 in user mode.
- `docs/INSTALL_MODE.md` and `docs/CODEX_REINSTALL_PROMPT.md`.

### Changed

- `scripts/install-agent-patch-kit.ps1` now uses non-editable install by default.
- `scripts/install-agent-patch-kit.ps1 -Dev` is now the explicit developer editable install mode.
- `scripts/update-agent-patch-kit.ps1` uses non-editable install by default.

## v0.2.1

- Added developer brief and system/user/project enable-disable state control.

## v0.2.0

- Added ast-grep wrapper, transaction rollback, Markdown link validation and AiCoding marketplace packaging.
