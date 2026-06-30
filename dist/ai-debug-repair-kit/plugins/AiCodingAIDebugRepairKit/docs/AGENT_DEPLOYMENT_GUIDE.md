# Agent 部署与集成指导：AI Debug Repair Kit

> **适用对象**：Codex、Claude Code、Gemini CLI、具备终端执行能力的 Agent
> **适用仓库**：`https://github.com/JiaxI2/AiCoding`
> **Kit 名称**：`aicoding-ai-debug-repair-kit`
> **CLI 名称**：`airepair`
> **版本**：v0.4.1
> **目标**：指导 Agent 完成本地单独部署、本地 AiCoding 集成、远程 AiCoding 仓库集成、验证、状态检查和卸载。

---

## 0. Agent 总规则

Agent 执行本 Kit 部署前必须遵守：

1. 先确认当前目录和目标目录，不允许在未知目录直接复制文件。
2. 优先使用 PowerShell 脚本，不手工散拷文件。
3. 所有安装、验证、卸载命令使用 `-Json`，便于结构化判断。
4. 不自动提交 Git。
5. 不自动 push。
6. 不自动 flash/reset/halt。
7. 不删除用户源码。
8. 不修改用户已有业务代码。
9. 失败时保留日志和状态文件，报告失败点。
10. 安装后必须运行 status 和 verify。

Agent 必须区分三种模式：

| 模式 | 目的 | 是否修改 AiCoding |
|---|---|---:|
| Standalone Local | 只安装 `airepair` CLI | 否 |
| AiCoding Local Integration | 集成到本地 AiCoding 工作区 | 是 |
| AiCoding Remote Integration | 将插件正式提交到远程 AiCoding 仓库 | 是，需要 Git commit/PR，但不得自动执行 |

---

# 1. Standalone Local：单独本地部署

## 1.1 适用场景

用户只想使用 `airepair` CLI，不希望修改 AiCoding 仓库，也不希望更新 `.agents/plugins/marketplace.json`。

适合：

- 临时试用；
- 在任意工程中运行 build/test repair profile；
- 不需要 Codex plugin marketplace；
- 不需要把 Kit 写入 AiCoding。

## 1.2 Agent 前置检查

Agent 应执行：

```powershell
python --version
python -m pip --version
```

如果 Python 不存在，停止并提示用户安装 Python 3.10+。

## 1.3 安装命令

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File "<package-root>\scripts\install-airepair-standalone.ps1" `
  -PackageRoot "<package-root>" `
  -Json
```

其中 `<package-root>` 是解压后的 Kit 根目录，例如：

```powershell
F:\Downloads\aicoding-ai-debug-repair-kit-v0.4.1
```

## 1.4 验证命令

```powershell
airepair version --output json
airepair doctor --output json
```

初始化当前工作区 profile：

```powershell
airepair init --workspace . --output json
airepair profile validate --profile .ai-debug-repair\profiles\loop.safe.json --output json
```

## 1.5 卸载命令

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File "<package-root>\scripts\uninstall-airepair-standalone.ps1" `
  -Json
```

## 1.6 Standalone 模式不会修改

```text
.agents/plugins/marketplace.json
dist/ai-debug-repair-kit/
scripts/install-ai-debug-repair-kit.ps1
scripts/verify-ai-debug-repair-kit.ps1
```

---

# 2. AiCoding Local Integration：集成本地 AiCoding

## 2.1 适用场景

用户已经有本地 AiCoding 仓库，希望 Codex 能通过本地 plugin/skills 使用 Repair Kit。

目标结果：

```text
AiCoding/
├─ .agents/plugins/marketplace.json
├─ dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit/
├─ scripts/install-ai-debug-repair-kit.ps1
├─ scripts/uninstall-ai-debug-repair-kit.ps1
├─ scripts/status-ai-debug-repair-kit.ps1
├─ scripts/verify-ai-debug-repair-kit.ps1
└─ .ai-debug-repair/install-state.json
```

## 2.2 Agent 判断当前目录是否为 AiCoding 根目录

Agent 应检查当前目录是否存在以下任意组合：

```text
.agents/
CodingKit/
dist/
README.md
```

建议 PowerShell：

```powershell
Get-ChildItem -Force
Test-Path ".agents"
Test-Path "CodingKit"
Test-Path "dist"
```

如果当前目录不像 AiCoding 根目录，Agent 应停止并要求用户进入 AiCoding 根目录。

## 2.3 安装命令

在 AiCoding 根目录执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File "<package-root>\scripts\install-ai-debug-repair-kit.ps1" `
  -PackageRoot "<package-root>" `
  -Json
```

脚本执行内容：

1. 复制插件包到：
   ```text
   dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit
   ```
2. 更新：
   ```text
   .agents/plugins/marketplace.json
   ```
3. 安装 `airepair` CLI。
4. 复制管理脚本到：
   ```text
   scripts/
   ```
5. 写入：
   ```text
   .ai-debug-repair/install-state.json
   ```

## 2.4 安装后状态检查

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\status-ai-debug-repair-kit.ps1" `
  -Json
```

必须检查：

- `pluginExists = true`
- `manifestExists = true`
- `marketplaceExists = true`
- `airepair` 可找到

## 2.5 完整验证

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\verify-ai-debug-repair-kit.ps1" `
  -Json
```

并运行：

```powershell
airepair doctor --output json
airepair version --output json
```

## 2.6 初始化 Repair Loop Profile

```powershell
airepair init --workspace . --output json
```

生成：

```text
.ai-debug-repair/profiles/build.json
.ai-debug-repair/profiles/test.json
.ai-debug-repair/profiles/loop.safe.json
```

Agent 后续使用 `ai-debug-repair-loop` Skill 前，必须先读取并验证：

```powershell
airepair profile validate --profile .ai-debug-repair\profiles\loop.safe.json --output json
```

## 2.7 本地卸载

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\uninstall-ai-debug-repair-kit.ps1" `
  -Json
```

如需同时卸载 CLI：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\uninstall-ai-debug-repair-kit.ps1" `
  -Json `
  -UninstallPip
```

卸载会删除：

```text
dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit
.ai-debug-repair/install-state.json
marketplace 中的 aicoding-ai-debug-repair-kit 入口
```

卸载不会删除：

```text
用户源码
用户 Git 历史
用户手工创建的 profile
运行日志，除非用户明确要求
```

---

# 3. AiCoding Remote Integration：集成远程仓库

## 3.1 适用场景

用户希望把 Repair Kit 正式加入：

```text
https://github.com/JiaxI2/AiCoding
```

使得新电脑 clone AiCoding 后，可以通过仓库自带脚本恢复 Kit。

## 3.2 推荐分支流程

Agent 不得直接提交到 `main`，应创建 feature 分支：

```powershell
git clone https://github.com/JiaxI2/AiCoding.git AiCoding-ai-debug-repair
cd AiCoding-ai-debug-repair
git checkout -b feature/ai-debug-repair-kit-v0.4.1
```

安装插件快照，但不安装本机 pip 包：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File "<package-root>\scripts\install-ai-debug-repair-kit.ps1" `
  -PackageRoot "<package-root>" `
  -Json `
  -SkipPipInstall
```

验证文件结构：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\verify-ai-debug-repair-kit.ps1" `
  -Json
```

注意：远程集成阶段使用 `-SkipPipInstall` 是为了避免把本机 Python 环境状态作为仓库集成的必要条件。

## 3.3 远程集成应提交的文件

应提交：

```text
dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit/.codex-plugin/plugin.json
dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit/skills/**
dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit/assets/ai-debug-repair-kit/**
dist/ai-debug-repair-kit/marketplace.ai-debug-repair.json
scripts/install-ai-debug-repair-kit.ps1
scripts/uninstall-ai-debug-repair-kit.ps1
scripts/status-ai-debug-repair-kit.ps1
scripts/verify-ai-debug-repair-kit.ps1
.agents/plugins/marketplace.json
```

建议同时提交：

```text
docs/AI_DEBUG_REPAIR_KIT_TECHNICAL_DESIGN.md
docs/AGENT_DEPLOYMENT_GUIDE.md
docs/REMOTE_INTEGRATION_CHECKLIST.md
assets/gitignore.ai-debug-repair.fragment
```

如果 AiCoding 仓库有统一 docs 目录，可以把这些文档移动到：

```text
docs/ai-debug-repair-kit/
```

但插件包内部仍建议保留一份 README。

## 3.4 远程集成不应提交的文件

不得提交：

```text
.venv/
__pycache__/
.pytest_cache/
.ai-debug-repair/install-state.json
.ai-debug-repair/runs/
.ai-debug-repair/attempts.jsonl
.ai-debug-repair/repair-context.json
.ai-debug-repair/repair-report.md
真实硬件串口日志
真实硬件 TCP 日志
用户私有路径和设备序列号
```

建议将以下片段加入 `.gitignore`：

```gitignore
# AI Debug Repair Kit runtime state
.ai-debug-repair/install-state.json
.ai-debug-repair/runs/
.ai-debug-repair/attempts.jsonl
.ai-debug-repair/repair-context.json
.ai-debug-repair/repair-report.md
```

## 3.5 Git 检查

```powershell
git status --short
git diff --check
```

Agent 应输出变更摘要，但不得自动 commit，除非用户明确要求。

用户要求提交时，建议：

```powershell
git add dist/ai-debug-repair-kit scripts docs assets .agents/plugins/marketplace.json
git commit -m "feat(ai-debug): add repair loop kit plugin"
git push -u origin feature/ai-debug-repair-kit-v0.4.1
```

## 3.6 PR 描述模板

```markdown
## Summary

Add AiCoding AI Debug Repair Kit plugin package.

## Includes

- `ai-debug-kit-deploy` Skill
- `ai-debug-operations` Skill
- `ai-debug-repair-loop` Skill
- `airepair` CLI assets
- install/status/verify/uninstall scripts
- local marketplace entry
- Agent deployment documentation

## Safety

- No flash/reset/halt by default
- No auto commit/push
- PASS only from configured test runner
- Bounded `max_iterations`
- Forbidden paths enforced by repair profile

## Verification

- `verify-ai-debug-repair-kit.ps1` passed
- `airepair doctor --output json` passed
- `airepair profile validate` passed
```

---

# 4. 新电脑从远程 AiCoding 恢复

## 4.1 Clone

```powershell
git clone https://github.com/JiaxI2/AiCoding.git
cd AiCoding
```

## 4.2 安装本地 CLI 和激活插件

如果 Repair Kit 已经随 AiCoding 提交：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\install-ai-debug-repair-kit.ps1" `
  -PackageRoot "." `
  -Json
```

这里 `PackageRoot="."` 的前提是 AiCoding 仓库内已经存在：

```text
dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit
```

## 4.3 验证

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\status-ai-debug-repair-kit.ps1" `
  -Json

powershell -NoProfile -ExecutionPolicy Bypass `
  -File ".\scripts\verify-ai-debug-repair-kit.ps1" `
  -Json

airepair doctor --output json
```

## 4.4 初始化项目 profile

```powershell
airepair init --workspace . --output json
```

用户应根据项目实际修改：

```text
.ai-debug-repair/profiles/build.json
.ai-debug-repair/profiles/test.json
.ai-debug-repair/profiles/loop.safe.json
```

---

# 5. Agent 使用 Repair Loop 的标准提示词

## 5.1 安全部署检查

```text
使用 ai-debug-kit-deploy 检查当前 AiCoding 仓库中的 AI Debug Repair Kit 是否可用。
请运行 status、verify、airepair doctor，并检查 marketplace 是否包含 aicoding-ai-debug-repair-kit。
不要修改源码，不要提交。
```

## 5.2 初始化本地 profile

```text
使用 ai-debug-kit-deploy 初始化当前仓库的 airepair profile。
执行 airepair init，然后验证 loop.safe.json。
只输出需要用户修改的 build/test 命令位置，不要启动修复循环。
```

## 5.3 执行受控修复循环

```text
使用 ai-debug-repair-loop。
基于 .ai-debug-repair/profiles/loop.safe.json 做最多 3 轮修复。
每轮只能修改 allowed_paths 内文件，不得修改 forbidden_paths。
先运行 airepair loop export-context。
每轮运行 airepair build run 和 airepair test run。
只有 test runner 返回 ok:true 才能记录 pass。
不要 flash/reset/halt，不要 commit，不要 push。
结束后生成 repair report。
```

## 5.4 远程集成

```text
使用 ai-debug-kit-deploy 将 AI Debug Repair Kit 集成到当前 AiCoding 仓库。
只做本地文件集成和验证，不要提交。
完成后输出 git status、应提交文件列表、PR 描述建议。
```

---

# 6. 常见失败处理

## 6.1 `airepair` 找不到

执行：

```powershell
python -m pip install --user --force-reinstall ".\dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit\assets\ai-debug-repair-kit"
```

然后重新验证：

```powershell
airepair doctor --output json
```

## 6.2 marketplace 没有入口

检查：

```powershell
Get-Content .agents\plugins\marketplace.json
```

缺失时重新安装：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\install-ai-debug-repair-kit.ps1" -PackageRoot "." -Json -SkipPipInstall
```

## 6.3 profile validate 失败

检查：

```text
max_iterations > 0
build_profile 存在
test_profile 存在
allowed_paths 非空
forbidden_paths 不与 allowed_paths 重叠
allow_flash = false
auto_commit = false
```

## 6.4 build 通过但 test 失败

Agent 不得记录 PASS。应记录：

```powershell
airepair loop record-attempt --profile .ai-debug-repair\profiles\loop.safe.json --result fail --notes "build passed, test failed" --output json
```

然后让 Codex 读取测试日志后做下一轮最小 patch。

---

# 7. 最终验收标准

Standalone 验收：

```text
[ ] airepair version --output json 成功
[ ] airepair doctor --output json 成功
[ ] airepair init 成功
[ ] profile validate 成功
```

AiCoding Local 验收：

```text
[ ] 插件目录存在
[ ] plugin.json 存在
[ ] 三个 Skill 存在
[ ] marketplace 入口存在
[ ] airepair CLI 可用
[ ] status 成功
[ ] verify 成功
[ ] 可卸载
```

Remote Integration 验收：

```text
[ ] git diff --check 通过
[ ] 无运行态日志
[ ] 无 .venv
[ ] 无真实硬件私有记录
[ ] PR 描述包含 Safety 和 Verification
[ ] 新电脑 clone 后可运行 install 脚本恢复
```
