# TODO 0023: 能力可发现性（单一 Registry + 生成式 README 索引 + 孤儿门禁）

Status: Done
Verify: bin/aicoding.exe capability list --json 覆盖 28/28 个 internal 一级目录且无 unregistered；四条负例均被抓；Full 65/62/0/0/3 PASS

> 来源：《internal 能力产品化与可发现性架构优化方案》评审。
> **问题真实、方案超配 5 倍。** 本项取其诊断，砍掉 4/5 的产物义务。

## 一、问题确认：真实且严重

`internal/` 有 28 个一级目录，用户（和 Agent）**无法知道有什么、怎么用、从哪进**。
文档的核心论断我完全同意：

> **不可发现的能力，等价于不可用的能力。**

而且这条对 Agent 比对人更致命——Agent 不会翻源码猜能力边界。

## 二、评审：砍掉什么，为什么

原方案要求**每个能力 4 份文档**（product / quickstart / architecture / reference）
+ 2 份新注册表 + 6 条新命令 + 脚手架。按 28 个目录算 = **112 份文档**。
单人仓库不可能维护，半年后必然全部过期——**过期文档比没文档更有害**。

| 原方案 | 裁决 | 理由 |
|---|---|---|
| `config/internal-capabilities.json` | ✅ **采纳，唯一新增** | 机器可读能力注册表确实缺 |
| `config/product-catalog.json` | 🔴 **拒绝** | 与上者 + `kit-registry.json` 构成三个注册表；产品视图是 registry 的**投影**，不是第二份数据 |
| 每能力 4 份文档 | ⚠️ **砍成按需** | 见下"文档义务分级" |
| `docs/PRODUCTS.md` + `docs/INTERNAL_CAPABILITIES.md` | ⚠️ **合成一份** | 两份索引即两处漂移面；`docs/CAPABILITIES.md` 一份，且**生成**不手写 |
| `aicoding capability list/describe` | ✅ 采纳 | 与 `kit describe` 同构（0009 已建的投影模式） |
| `aicoding capability graph/doctor/new` | 🔴 拒绝 | graph 无消费者；doctor 并入 `governance capabilities`；new 归 0020 模板家族 |
| `aicoding docs generate-capability-index` | ⚠️ 改为 `capability index --write` | 不新增 `docs` 命令域；生成职责归 capability 自己 |
| `aicoding governance capabilities` | ✅ 采纳 | 孤儿治理是关键门禁 |
| `aicoding loop plan/run/status/resume/stop/report` | 🔴 **强烈拒绝** | 文档凭空发明了 6 条命令，其中 **`loop run` 正是 LOOP_ENGINEERING_ARCHITECTURE §1 明令永不实现的**（让 AiCoding 自己转循环 = 第二控制面）。实际实现是 `work validate/next/status/record` 四条 |
| 仓库根 `kit/` 目录 | 🔴 拒绝 | 与既有 `config/kits/*.json` + CodingKit 冲突 |
| `internal/loopengineering` | ⚠️ 事实错误 | 实际包名是 `internal/loopkit` |

> **注意**：该文档是对着一个想象中的仓库写的（包名、命令、目录三处都对不上）。
> 采纳它的**诊断**，不采纳它的**清单**。

### 文档义务分级（本项的核心裁决）

**按是否有公共入口分三档，不是一刀切要求 4 份：**

```text
有公共 CLI 入口的能力（约 8 个）
  必须：registry 记录 + 一段 summary + 已有架构文档链接
  可选：quickstart（只在 CLI 参数非平凡时写）
  ——不强制 product/reference 两份

无公共入口的实现域（约 15 个）
  只需：registry 记录 + 一句 summary + type: internal-only
  ——零新增文档

Primitive（gitx/report/platform/registry/runner 等，约 5 个）
  只需：registry 记录 + type: primitive
  ——零新增文档，它们本来就不该面向用户
```

**判据不是"写了几份文档"，是"从 README 三跳内能否找到答案"**——
这正是原文档 §14.1 自己定的验收标准，本项只保留这一条。

## 三、实现计划

1. **`config/internal-capabilities.json`**（唯一新增注册表）+ 配套 schema。
   字段砍到必要集合：

   ```json
   { "id": "validation-evidence", "package": "internal/validationevidence",
     "name": "Validation Evidence", "type": "domain-capability",
     "status": "stable", "summary": "把测试结论绑定到 Git 内容身份，同一内容零成本复用。",
     "publicEntries": ["aicoding validation check", "aicoding validation explain"],
     "architectureDoc": "docs/decisions/0007-validation-evidence.md",
     "verification": ["go test ./internal/validationevidence/..."] }
   ```

   `type ∈ {primitive, domain-capability, product-workflow, internal-only}`；
   `status ∈ {experimental, beta, stable, deprecated}`。
   **不要 owners（单人仓库）、不要 dependencies（`governance dependencies` 已是权威）。**

2. **`internal/capability` 包**（只读投影，≤6 个公开函数）：
   `Load` / `List` / `Describe` / `Verify`（孤儿检查）/ `RenderIndex`。
   只读 registry + `os.Stat` 校验文档路径存在，**不扫描 internal/ 源码**
   （目录清单来自 registry 与 `os.ReadDir("internal")` 单次对比）。

3. **CLI 三条**（同步 HelpForm + COMMANDS.md）：

   ```text
   aicoding capability list [--type T] [--status S] --json      只读
   aicoding capability describe --id X --json                   只读
   aicoding capability index --write                            仅写 README 生成区
   ```

4. **`governance capabilities`** 并入既有 governance 域，检查：
   - `internal/` 一级目录 ⊆ registry（**未注册即孤儿 → 红**）；
   - registry 声明的文档路径存在；
   - `publicEntries` 的命令在 typed command catalog 中存在；
   - `status: stable` 的能力必须有 `verification`；
   - README 生成区与 registry 一致（digest 比对）。

5. **README 生成区**（0012 的能力橱窗升级为生成式）：

   ```md
   <!-- BEGIN GENERATED: CAPABILITIES -->
   <!-- END GENERATED: CAPABILITIES -->
   ```

   `docsync` 增加一条：生成区内容与 `capability index` 输出不一致即 error
   （与 0007 的 architecture Status 门禁同一机制）。

6. **`docs/CAPABILITIES.md`** 一份索引，同样生成、不手写。

## 四、明确不做

- 不建 `product-catalog.json`（产品视图是 registry 投影）。
- 不为每个能力强制 4 份文档（分级见上）。
- 不新增 `capability graph/doctor/new`、不新增 `docs` 命令域。
- **不实现 `loop run`**（原文档提议，但架构文档明令禁止）。
- 不建仓库根 `kit/` 目录。
- 不做能力成熟度评分/排名（二值门禁原则）。

## 五、自测（可信任方式）

```powershell
go test ./internal/capability/... ./internal/governance/... ./internal/docsync/...

bin\aicoding.exe capability list --json          # 覆盖 internal/ 全部一级目录
bin\aicoding.exe capability describe --id validation-evidence --json
bin\aicoding.exe capability index --write ; git diff README.md docs/CAPABILITIES.md

# 孤儿负例：临时新建 internal/orphan/ → governance capabilities 必须红 → 删除后转绿
# 文档负例：registry 指向不存在的 architectureDoc → 必须红
# 生成区负例：手改 README 生成区 → docsync all 必须红
# 命令负例：publicEntries 写一个不存在的命令 → 必须红

1..5 | % { (Measure-Command { bin\aicoding.exe capability list --json }).TotalMilliseconds }
#   中位数 < 300ms（0022 的 fast 档）；禁止全仓扫描

bin\aicoding.exe docsync all --json ; bin\aicoding.exe test --profile Full --json
```

通过判据：
1. 28 个 internal 一级目录全部有 registry 记录，无孤儿。
2. 四条负例全部被抓（逐条贴输出）。
3. `capability list` <300ms 且不做全仓扫描（调用计数断言）。
4. README 与 `docs/CAPABILITIES.md` 生成区手改即报错。
5. **全仓只有一个能力注册表**（`grep -rn "product-catalog" .` 为空）。
6. 无新增 `loop run` 类命令（`grep -rn '"loop"' internal/cli/catalog.go` 为空）。

## 六、完成证据（2026-07-21）

- `governance capabilities --json`：`registeredCount=28`、`internalDirectoryCount=28`、
  `unregistered=[]`，README 与 `docs/CAPABILITIES.md` 均为最新。
- 四条真实注入逐条转红并回滚：

  | 注入 | 明确错误 |
  |---|---|
  | 新建 `internal/orphan/orphan.go` | `unregistered internal package: internal/orphan` |
  | `architectureDoc` 指向 `docs/architecture/DOES_NOT_EXIST.md` | `architecture document is missing: validation-evidence: ...` |
  | 把 README 生成区的 28 手改为 29 | `capability generated index: README.md is stale` |
  | 写入 `aicoding validation nonexistent` | `public entry is absent from typed command catalog` |

- `capability list` 五次进程级墙钟为 `3911.442 / 47.958 / 41.970 / 49.999 /
  45.286 ms`，中位数 `47.958 ms < 300 ms`；包级调用计数回归证明 `List` 对
  `internal/` 目录读取为 0，`Verify` 恰好单次 `ReadDir`。
- 禁止项断言：仓库没有 `product-catalog` 文件或 TODO 自述之外的实现引用；
  `internal/cli/catalog.go` 没有 `"loop"` command；唯一能力数据文件为
  `config/internal-capabilities.json`（另有配套 schema）。
- 最终 Full 报告：`65 total / 62 pass / 0 fail / 0 warn / 3 skip`，引擎耗时
  `219388 ms`；新增 `CAP-001` 为 REQUIRED/PASS，耗时 `58 ms`。首次 Full 曾抓到
  README 投影测试误把 domain capability 当成 product workflow；修正断言并以
  `go test -count=1` 回归后，第二次 Full 完整转绿。
