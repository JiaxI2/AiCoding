# TODO 0009: Kit 管理标准化（管理契约 + Quickstart 投影 + 门禁闭环）

Status: Done
Verify: bin/aicoding.exe kit describe --all --json 每个 enabled kit 的 quickstart 字段非空，且 kit verify --all --profile Lifecycle 含管理面检查全绿

> 治理模型一句话（RL+SFT 的工程化翻译，只借结构不借算法）：
> **模板与 Schema 是先验（SFT——教会"合规长什么样"），门禁是奖励信号（RL——绿灯即正反馈），
> 两层之间是自由探索区（kit 内部实现随便写）。** 先验收紧形态、奖励约束结果、中间不管过程。
> 对应开源先例：Homebrew 的 formula 约定（先验）+ `brew audit`（奖励）+ 配方内部自由。

## 背景

现状：6 个 enabled kit 的"怎么快速上手/怎么维护"完全靠口口相传。`kit describe`
（plugin view，已上线）解决了"有什么能力"，但没回答"5 分钟怎么用起来"和"坏了谁修、怎么修"。
危险的反方向：给每个 kit 手写一份 QUICKSTART.md —— 那是平行事实源，
PLUGIN_STANDARD 已明确"UI/README 只消费生成 JSON，不维护手工副本"。

## 实现计划

1. **新增 `docs/reference/KIT_MANAGEMENT_STANDARD.md`**（唯一权威，含 Status 头），定义
   每个 registered kit 必须回答的三面九问，以及每一问的**机器可查落点**：

   | 面 | 问题 | 落点（全部已有字段/命令，不新增事实源） |
   |---|---|---|
   | 快速入门 | 我是谁/解决什么 | manifest `description`（面向用户结果） |
   | | 5 分钟上手命令 | manifest `commands` 中至少一条 read 操作 + `kit describe` 投影 |
   | | 有哪些 Skill | manifest `skills`（可为空但须显式） |
   | 快速使用 | 只读操作 | plugin view `operations.read` |
   | | 写操作与恢复 | plugin view `operations.write` + `state.root` + lifecycle rollback |
   | 维护 | 谁拥有/怎么升级 | `trust.{level,source,updatePolicy}` |
   | | 怎么验证 | `profiles` + `kit verify --profile Lifecycle` |
   | | 外部依赖怎么跟 | 外部包装类 kit 必须有边界卡（见 0010） |

2. **Quickstart 是投影，不是文件**：扩展 `kit describe` 的 plugin view 增加
   `quickstart` 字段（从 manifest description + 首条 read command + skills 描述**派生**，
   零手工维护）。人类可读形态走 `report.WriteText` 渲染，不落盘。
3. **门禁接入 `VerifyCatalogStructure`**（老位置，不新建 validator）新增检查：
   - enabled kit 的 description 面向用户结果（非空且不以内部实现开头——启发式：
     不以 "internal"/"Go"/包名开头）；
   - 至少一条 read 类型 command；
   - `trust.updatePolicy` ∈ {manual, pinned, tracked}；
   - 外部包装类（`trust.thirdParty: true`）必须存在对应边界卡文件。
   分级沿用惯例：Smoke → warning，Lifecycle/Full/Release → error。
4. **现有 6 个 kit 补齐欠账**：逐个跑新门禁，修 manifest 缺口（预计 description
   和 updatePolicy 有历史欠账）。**修 manifest，不放宽门禁**（aicoding-platform 陈旧
   status 命令的处理先例）。
5. `docs/reference/KIT_PLUGIN_VIEW.md` 与本标准互链（消费者侧 vs 管理侧，边界写死）。

## 明确不做

- 不给每个 kit 手写 QUICKSTART/README（平行事实源）。
- 不新增 manifest 字段（schema 已冻结；三面九问全部映射到既有字段）。
- 不做 kit 评分/排名（奖励信号是二值门禁，不是连续分数——防"指标游戏"）。
- 不引入任何真实 RL/SFT 机制（那是隐喻，不是实现）。

## 自测（可信任方式）

```powershell
go test ./internal/kit/... ./internal/cli/...
# 负例：临时把某 kit description 改为 "internal helper" → Lifecycle profile 必须 error → 恢复
bin\aicoding.exe kit verify --all --profile Lifecycle --json   # 全绿（欠账已修）
bin\aicoding.exe kit verify --all --profile Smoke --json       # 新检查为 warning 级
bin\aicoding.exe kit describe --all --json                     # 每个 enabled kit quickstart 非空
# 确定性：连续两次 describe 剔除 elapsed 后字节一致
bin\aicoding.exe docsync all --json ; bin\aicoding.exe test --profile Full --json
```

通过判据：负例被抓（贴输出）；6 个 kit 全部过新门禁；quickstart 字段确实由 manifest
派生（改 description → describe 输出跟着变，贴前后对比）；全仓无新增 QUICKSTART.md 文件。
