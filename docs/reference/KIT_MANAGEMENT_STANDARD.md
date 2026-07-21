# Kit 管理标准

Status: Accepted

## 1. 定位与权威

本文是 registered Kit 快速入门、使用和维护信息的唯一管理侧权威。消费者侧字段与派生契约见
[Kit Plugin View](KIT_PLUGIN_VIEW.md)。Kit 不得新增手写 `QUICKSTART.md` 或逐 Kit README；
`kit describe` 必须从 detached registry/manifest、Skill 元数据和静态 adapter catalog 投影，
避免形成第二事实源。

模板与 schema 定义可接受形态，既有门禁给出二值反馈；Kit 内部实现仍由各 Kit 自己拥有。
本标准不引入评分、排名、运行时 Plugin Registry、第二生命周期或新的 manifest 字段。

## 2. 三面九问

| 面 | 必须回答的问题 | 机器权威与查询落点 |
|---|---|---|
| 快速入门 | 1. 解决什么问题 | manifest `description`；`PluginView.quickstart.purpose` |
| | 2. 五分钟先跑什么 | manifest `commands` 中按名字排序的首个 read operation；`PluginView.quickstart.command` |
| | 3. 有哪些 Skill | manifest `skills`；`PluginView.quickstart.skills`，允许显式空数组 |
| 快速使用 | 4. 哪些操作只读 | `PluginView.operations[]` 中 `effect: read` |
| | 5. 哪些操作写状态、如何恢复 | `operations[].effect: write`、manifest `state.root`、领域 lifecycle rollback |
| 维护 | 6. 谁拥有、来源何处 | manifest `trust.level` 与 `trust.source` |
| | 7. 如何升级 | manifest `trust.updatePolicy`：`manual`、`pinned` 或 `tracked` |
| | 8. 如何验证 | manifest `profiles` 与 `kit verify --profile Lifecycle` |
| | 9. 外部依赖边界是什么 | `trust.thirdParty: true` 时的 `docs/reference/kits/<id>-BOUNDARY.md` |

九问只引用既有事实。`quickstart` 是同一事实的只读便利投影，不是可独立编辑的配置。

## 3. Quickstart 派生契约

`PluginView.quickstart` 固定包含：

```json
{
  "purpose": "面向用户结果的 manifest description",
  "command": "aicoding lifecycle status --scope kit --kit <id> --json",
  "skills": [
    {"id": "skill-id", "description": "manifest skill description"}
  ]
}
```

- `purpose` 是去除首尾空白后的 manifest `description`。
- `command` 取按 command 名字排序后的首个 read operation，再投影为现有 typed CLI 调用；
  不执行该命令，也不写文件。
- `skills` 复用 `internal/kit.parseSkillEntries` 的稳定 ID 顺序，仅保留 ID 与描述；无 Skill 时
  输出 `[]`。
- JSON 由 `kit describe --json` 输出；人类可读形态由既有 `report.WriteText` 即时渲染。

## 4. 管理门禁

门禁只扩展 `internal/kit.VerifyCatalogStructure` 已有的 `plugin view projection` 检查：

1. enabled Kit 的 description 必须非空且面向用户结果；以 `internal`、`Go`、`package`、
   `internal/`、`cmd/`、`pkg/` 或 `go-` 开头视为实现先行。
2. enabled Kit 至少登记一个可解析为 read 的 command。
3. enabled Kit 的 `trust.updatePolicy` 只允许 `manual`、`pinned`、`tracked`。
4. enabled 且 `trust.thirdParty: true` 的 Kit 必须存在约定路径边界卡。

同一问题在 Smoke 只进入 warnings；Lifecycle 进入 errors。Full/Release 通过既有 Lifecycle
结构用例间接阻断，不新建 validator 或 profile。修复方式只能是补正 manifest 或边界卡，
不得降低检查集。

## 5. 维护流程

1. 修改既有 manifest、Skill 或 adapter 权威。
2. 运行 `kit describe --kit <id> --json`，确认 Quickstart 随权威输入变化。
3. 运行 `kit verify --all --profile Lifecycle --json`。
4. 运行 DocSync 与目标 Full/Release 门禁。

回滚仓库事实使用 `git revert`；已执行的 Kit 写状态使用领域 lifecycle rollback。两者不合并，
也不得靠删除未知文件模拟恢复。
