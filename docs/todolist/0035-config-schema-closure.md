# TODO 0035: config schema 闭合完备化

Status: In-Progress
Verify: bin/aicoding.exe governance dependencies --json && bin/aicoding.exe docsync all --json && bin/aicoding.exe plan verify --json && bin/aicoding.exe todolist --json

## 架构裁决

配置权威继续是 Git + JSON + checked-in schema，Validation Receipt store 继续是内容寻址文件存储。
本项不引入数据库、第三方 JSON Schema 库、集中式配置加载框架或新的治理领域。只有在
Receipt List / `--verify-reuse` 的实测延迟显著退化，或出现跨仓库证据聚合需求时，才重新
评估数据库；量化阈值与测量口径写入 Validation Evidence BUDGET。

## 开工审计基线

2026-07-22 以 `rg --files config -g '*.json'` 实测 53 个 JSON：34 个非 schema 配置与
19 个 `config/schemas/*.json`。当前 `internal/docsync/policy_schema.go` 的闭合表为 6/6。

### a) 已绑定且由 policy schema closure 强制执行（6）

- `config/docs-sync.policy.json` → `config/schemas/docs-sync-policy.schema.json`
- `config/docs-sync.semantic.json` → `config/schemas/docs-sync-semantic.schema.json`
- `config/impact-policy.json` → `config/schemas/impact-policy.schema.json`
- `config/plan-policy.json` → `config/schemas/plan-policy.schema.json`
- `config/tagging-policy.json` → `config/schemas/tagging-policy.schema.json`
- `config/validation-policy.json` → `config/schemas/validation-policy.schema.json`

### b) schema 已存在但未进入 policy closure（13）

下列 schema 有的只被专门消费方、自定义解码或 freeze/存在性断言覆盖；这些校验继续保留，
但不替代本项的 schema 语义闭合。能对应 checked-in 配置的逐项加入绑定表；输出/工件 schema
进入显式 standalone 反向登记，使新增或删除幽灵 schema 都 fail-closed。

- 可绑定配置：`agent-dev-kit-plan-mode.registry.schema.json`、
  `dependency-governance.schema.json`、`internal-capabilities.schema.json`、
  `kit-manifest.schema.json`、`kit-registry.schema.json`、`loop-work-spec.schema.json`、
  `mcp-component.schema.json`、`mcp-registry.schema.json`、`pwsh-budget.schema.json`。
- standalone schema：`cli-report.schema.json`、`loop-attempt.schema.json`、
  `loop-profile.schema.json`、`plan-spec.schema.json`。
- 已确认的现有执行位置包括 `internal/governance`、`internal/capability`、`internal/kit`、
  `internal/loopkit`、`internal/mcpcontrol`、`internal/repohealth`、`internal/report`、
  `internal/plan` 与 `internal/testengine/freeze.go`；Plan Mode registry 的 PowerShell 检查
  当前只确认 schema 可解析，尚未用 schema 验证实例，因此仍按幽灵 surface 收口。

### c) 完全无 schema 的配置（9）

- `config/codex-kit.json`
- `config/common-registry.json`
- `config/hooks-registry.json`
- `config/repository-layout.json`
- `config/repository-navigation.json`
- `config/reuse-governance.json`
- `config/skill-sources.json`
- `config/skills/c99-standard-c/skill.json`
- `config/skills/c99-standard-c/templates/comment-templates.json`

提示清单与实测有两处差异：Phase 2 后 `config/codex-kit.json` 仍存在；提示清单未列出上述
两个 `config/skills/c99-standard-c/**` JSON。以本次枚举为验收基线，不移动、合并或重命名。

内置校验器只读取其支持的关键字，不会拒绝未知 schema 元数据，已确认 `$comment` 会被忽略；
因此开放扩展 map 可直接用 `$comment` 解释，不需要扩展校验器。

## 实施范围

1. 为 c 类九项与新增的 schema closure exclusion 配置补 strict schema，默认
   `additionalProperties: false`；动态键 map 必须用 `$comment` 说明理由。
2. 把全部非 schema 配置加入唯一 binding table；registry 的专门 coverage 校验继续并存。
3. `governance dependencies` 复用既有 inventory，要求 `config/` 每个 JSON 都被 binding 或
   显式 exclusion 覆盖；唯一通配形式是目录后缀 `/**`。
4. exclusion 配置自身有 schema；不存在的排除、模糊通配、未登记配置和未登记 schema
   均非零并指出路径。`config/schemas/**` 的理由是 schema 由 binding/standalone 反向约束。
5. 审计后目标为新增 exclusion 配置在内的 35/35 配置绑定；若实施时枚举变化，以实测 N/N
   为准并在本条目留下差异。

## 真跑负例

- 新绑定配置注入 `"illegal": true`：schema closure 非零并指出 `$` 路径。
- 新增 `config/rogue.json`：dependencies 完备性非零并指出文件。
- exclusion 写入不存在路径：非零并指出幽灵排除。
- exclusion 自身注入非法字段：非零并指出 `$` 路径。
- 删除任一 b 类 schema：闭合检查非零并指出 schema。
- 全部还原后 Full 全绿；随后独立 Release 全绿。

## 完成定义

- Plan Mode 覆盖 `config/schemas/**` 与全部实际修改路径并绑定 approved tree。
- Full、Release 各真跑一次；固定 summary 路径写回本条目。
- `docsync all`、`governance dependencies`、`plan verify`、`todolist` 全绿。
- 工作树干净，`git diff --exit-code` 通过；main 经正常 pre-push Receipt 门禁推送。
- 不翻转 reuse 默认值，不触碰 `CodingKit/agents/skills` 或 TODO 0019。
