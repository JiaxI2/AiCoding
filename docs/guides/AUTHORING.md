# Skill / Kit / MCP / Hook 创作指引

本页只说明创作入口和权威边界，不建立第二套源码、registry、lifecycle 或验证体系。
生成器提供可审查的起点；是否登记、启用和发布仍由现有门禁决定。

## 改什么，去哪改

| 资产 | 权威位置 | 起点 | 完成路径 |
|---|---|---|---|
| 既有或新 Skill | AiCoding 仓库外的可写 Codex-Skills checkout/worktree | `skill init` 或直接修改上游权威源 | 上游校验与 PR → 合并 → AiCoding gitlink pin 前移 |
| Kit | AiCoding 的 `config/kits/`、`config/kit-registry.json` 与对应 capability | `kit init` | 评审生成物 → 实现 capability → verify → 显式 enable |
| MCP component | AiCoding 的 `CodingKit/tools/` capability、`config/mcp/components/` manifest 与 `config/mcp-registry.json` | `mcp init` | 实现 capability → 评审并登记 disabled entry → doctor/verify → 显式 enable |
| Hook | 仓库 Hook 在 `.githooks/`；插件 Hook 在其上游权威仓库 | 没有 `hook init` | 修改既有权威实现 → `verify hooks` 与匹配 profile |

`CodingKit/agents/skills` 是 Codex-Skills 的只读挂载和 pin，不是本地创作目录。
AiCoding 不拥有 Skill 源码，也不接受仓库内的第二份 `SKILL.md`。

## Skill：先预览，再写到仓库外

不传 `--out` 时，命令不写文件；JSON 报告在 `data.content` 返回完整草稿：

```powershell
bin\aicoding.exe skill init demo-skill --dry-run --json
```

确认后，把目标目录显式指向 AiCoding 之外的可写 Codex-Skills worktree：

```powershell
bin\aicoding.exe skill init demo-skill `
  --out F:\path\to\writable-Codex-Skills\path\to\demo-skill --json
```

输出只有一个 `SKILL.md`。它包含 `Skill Type`、`Workflow Contract`、`Gate Rules` 和
`Human Confirmation` 等稳定章节，且模板本身通过 Codex-Skills 的 `quick_validate.py` 与
`skill_gate.py validate` 结构门禁。它仍是明确标注为未批准的草稿：作者必须替换通用流程、
选择真实 CLI/Hook/CI gate，并由 owner 确认人工复核范围。

以下情况全部 fail-closed：ID 非小写连字符格式、目标已存在、目标位于 AiCoding 仓库内，
或经符号链接解析后落入 `CodingKit/agents/skills`。命令不会覆盖既有文件。

## Skill 跨仓升级三步

1. 在 AiCoding 之外的可写 Codex-Skills checkout/worktree 修改权威 Skill，运行上游校验并提交 PR。
2. 等 PR 合并后，在一次独立且获授权的 AiCoding 维护变更中，把
   `CodingKit/agents/skills` 检出到已合并 commit；只更新 gitlink，不在挂载目录创作源码。
3. 暂存 gitlink，运行 `skill verify --all --profile Smoke`、依赖治理和匹配的完整 profile，
   再提交 pin 前移。

示意命令如下；`<merged-commit>` 必须来自已经评审合并的上游 commit：

```powershell
git -C CodingKit/agents/skills fetch origin
git -C CodingKit/agents/skills checkout <merged-commit>
git add CodingKit/agents/skills
bin\aicoding.exe skill verify --all --profile Smoke --json
```

## Kit：复用既有合规脚手架

```powershell
bin\aicoding.exe kit init demo-kit --dry-run --json
bin\aicoding.exe kit init demo-kit --json
```

`kit init` 写 schema v2 manifest、disabled registry entry、WorkSpec 示例与保守 dependency
binding；`--external` 还写外部边界卡。它不自动 enable、不创建 Skill，也不虚构真实依赖。
正式启用前要按实现修正 binding，并执行 Kit/Lifecycle 门禁。

## MCP：只生成 manifest 和登记建议

不传 `--out` 时，JSON 报告在 `data.componentContent` 返回 component manifest，并在
`data.registryEntry` 返回 `enabled:false` 的登记建议；命令不修改 registry：

```powershell
bin\aicoding.exe mcp init demo-mcp --dry-run --json
bin\aicoding.exe mcp init demo-mcp --out config\mcp\components --json
```

骨架严格匹配冻结的 MCP component 结构，预置 Smoke/Full/Release 入口，并显式声明
`allowArbitraryShell:false`、`ownsWorkflowPrompts:false`。它不会生成 Python package 或业务
tools，因此不是可运行 component。作者必须实现 `CodingKit/tools/` 下的通用 capability，
修正 runtime、doctor、verify、安全与输出声明，人工把建议合并进 registry，再运行
`mcp doctor` 和对应 `mcp verify`。ID 已登记、使用保留的 `aicoding-` 前缀或目标已存在时拒绝。

## Hook：没有通用 init

现有 Git 触发点已经由 `.githooks/` 覆盖，新增通用 Hook 脚手架会制造第二套策略入口，
因此没有 `hook init` 或 `config/templates/hook/`。修改 Hook 时只改实际权威实现，并运行：

```powershell
bin\aicoding.exe verify hooks --json
bin\aicoding.exe test --profile Full --json
```
