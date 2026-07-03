# AiCoding Fast Path V1 Kit Layer Constraints

## 1. V1 分层原则

Fast Path V1 只接管高频、轻量、确定性的本地检查。它不替换旧 Kit 生命周期，也不执行外部重工具。

```text
L0 Go Fast Path：hook / governance / Smoke manifest check / perf
L1 PowerShell Legacy：install / Full verify / Release verify / Windows-specific glue
L2 Python Domain：ai-debug-repair / aicoding-agent-kit / pytest / repair loop
L3 Markdown Skill：SKILL.md / references / templates / rules
L4 CI Release：Full gate / Release gate / fresh clone / artifact validation
```

## 2. 每个 Kit 的 V1 约束

| Kit | Fast Path V1 允许检查 | Fast Path V1 禁止执行 |
|---|---|---|
| aicoding-platform | manifest、路径、builtin requiredPaths | marketplace install/update、Codex plugin 注册 |
| agent-patch-kit | manifest、apatch 资产路径 | `apatch scan/patch/doctor` 真执行 |
| ai-debug-repair-kit | Python 入口路径、配置路径 | repair loop、pytest、DSS/XDS/J-Link 操作 |
| codex-agent-powershell-skill-kit | PowerShell skill 路径 | PSScriptAnalyzer、AST 深度 gate |
| docsync-plus | staged-only 文档同步轻量检查 | semantic drift、link drift、release docs 全量检查 |
| aicoding-agent-dev-kit | manifest、CLI 路径 | Plan Mode 完整门禁、Spec/TDD 完整流程 |
| common-control-kit | C99 模板/模块路径 | 控制算法验证、单元测试、报告生成 |

## 3. Smoke / Full / Release 边界

### Smoke

由 `aicoding.exe` 优先执行：

```powershell
bin\aicoding.exe kit verify --all --profile Smoke --json
```

只检查结构和路径。

### Full

继续由 PowerShell/Python 执行：

```powershell
pwsh scripts/aicoding-kit.ps1 test -All -Profile Full -Json
```

### Release

继续由旧流程和 CI 执行：

```powershell
pwsh scripts/test-kit-fresh-clone.ps1 -Profile Release -Json
```

## 4. 禁止事项

Fast Path V1 中禁止：

```text
- 自动执行烧录、写 Flash、reset、halt、run、写寄存器、写内存
- 自动执行 curl | sh 或远程脚本
- 自动删除 .git、用户目录、外部工作目录
- 自动安装全局依赖
- 自动执行 Full/Release gate
```
