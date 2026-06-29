# Agent Patch Kit v0.2.2（自包含安装修复版）

## v0.2.2 安装行为

默认安装已经改成非 editable、自包含用户安装。`apatch install doctor` 显示 `install_mode: non-editable / user mode` 且 `bundle_assets: OK` 后，原始 zip 和解压目录可以删除。只有开发这个 Kit 本身时才使用 `-Dev`。

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1
apatch install doctor
```

修复 v0.2.1 的 broken editable install：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\repair-agent-patch-kit.ps1
```

详见 `docs/INSTALL_MODE.md` 和 `docs/CODEX_REINSTALL_PROMPT.md`。

Agent Patch Kit 是面向 AI Agent 的安全修改 Kit，用 CLI 固化以下流程：

```text
git status -> rg/apatch scan -> preview -> apply + transaction -> verify -> diff summary
```

目标是减少 Agent 在 Windows PowerShell 中直接写复杂正则、直接大范围替换、修改后不验证的问题。

## v0.2.1 新增

- `apatch brief`：开发者 / Agent 专用快速理解入口，用户文档不用读这个。
- `apatch state`：支持系统、个人、项目级启用 / 禁用 Agent Patch Kit。
- `apatch deploy --scope system|user|project`：支持 CLI 部署到系统级、个人 Agent 或指定项目。

## Windows 安装

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1
```

如果希望自动安装缺失工具：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -InstallMissing
```

## 让 Agent 快速读懂这个 Kit

这是给开发者和 Agent 使用的，不是普通用户入口：

```powershell
apatch brief --format md
apatch brief --format json
```

Codex / Agent 在执行修改前，应先运行：

```powershell
apatch state status
```

如果显示 disabled，不应该继续 apply，除非用户明确要求重新启用或授权覆盖。

## 打开 / 关闭 Kit

查看状态：

```powershell
apatch state status
apatch state where
```

个人级关闭 / 打开：

```powershell
apatch state disable --scope user --reason "temporary opt out"
apatch state enable --scope user --reason "restore"
```

项目级关闭 / 打开：

```powershell
apatch state disable --scope project --path C:\path\to\repo --reason "project opt out"
apatch state enable --scope project --path C:\path\to\repo --reason "project opt in"
```

系统级关闭 / 打开，Windows 下可能需要管理员权限：

```powershell
apatch state disable --scope system --reason "machine policy"
apatch state enable --scope system --reason "machine policy"
```

等效规则：

```text
system enabled AND user enabled AND project enabled
```

任意一级 disabled，Agent 都应停止 apply/edit 操作。

## 部署到本地系统 / 个人 Agent

个人 Agent：

```powershell
apatch deploy --scope user --agent both
```

系统级托管目录：

```powershell
apatch deploy --scope system --agent both
```

系统级部署会放到 `%ProgramData%\AgentPatchKit` 下，适合企业或本机统一 Agent wrapper 接入。Codex 最通用的默认方式仍是 user/project 部署。

## 部署到指定项目

```powershell
apatch deploy --scope project --agent both --project C:\path\to\repo --write-agents-snippet
```

或者用安装脚本：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -DeployScope project -ProjectRoot C:\path\to\repo -Agent both -WriteAgentsSnippet
```

## 常用命令

```powershell
apatch status
apatch scan "old text" --fixed
apatch replace --old "old text" --new "new text" --fixed --preview
apatch replace --old "old text" --new "new text" --fixed --apply
apatch verify --old "old text" --new "new text" --fixed
apatch summary
```

## ast-grep 结构化修改

```powershell
apatch ast --lang c --pattern "if ($A)" --rewrite "if ($A != NULL)" --preview
apatch ast --lang c --pattern "if ($A)" --rewrite "if ($A != NULL)" --apply
```

## 事务快照 / 回滚

`replace --apply` 和 `ast --apply` 默认会创建事务快照。

```powershell
apatch tx list
apatch tx rollback <transaction-id> --preview
apatch tx rollback <transaction-id> --apply --force
```

只有明确授权删除事务开始后新增的未跟踪文件时，才使用：

```powershell
apatch tx rollback <transaction-id> --apply --force --clean-created
```

## Markdown link validator

```powershell
apatch links --mode offline --include-fragments full
```

## AiCoding Marketplace 打包

```powershell
apatch package aicoding-plugin --out dist/agent-patch-kit --zip
```

或者：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File integrations\aicoding\package-marketplace.ps1 -AiCodingRoot C:\path\to\AiCoding
```
