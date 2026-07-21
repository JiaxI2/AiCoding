# TODO 0014: 延迟预算入宪（CLI ≤10s 分级承诺 + pull 暴露面收敛）

Status: Done
Verify: bin/aicoding.exe doctor perf --json 按延迟等级断言实测中位数；git pull 路径的子模块配置由 provision 写入并可验证

> 实测基线（2026-07-20，warm）：kit list 206ms / governance layout 273ms /
> lifecycle status --scope all 2151ms / doctor --all 2684ms / verify Smoke 3005ms /
> git pull --dry-run 本机 1.7s。
> 结论：**当前没有任何非工具链命令超 10s**——用户体感的"pull 几十秒"来自
> 网络到 GitHub + 18MB skills 子模块的 fetch（环境因素），但架构能收敛暴露面。
> 缺的不是优化，是**预算承诺 + 防回退门禁**：今天 3s 的命令没人盯着，两年后就是 15s。

## 实现计划

### A. 延迟等级入宪（性能预算是接口契约的一部分）

1. `PRIMITIVE_CONSTITUTION.md` §3 追加一段（只收紧不放宽，符合其修订规则）：
   CLI 命令按延迟等级分类，等级是**接口承诺**，升级（变慢一档）视同破坏性变更：

   | 等级 | 承诺（warm 中位数） | 覆盖 |
   |---|---|---|
   | fast | ≤ 300ms | 查询/投影类：list、describe、check、status（单域） |
   | standard | ≤ 3s | 聚合诊断类：doctor --all、verify Smoke、lifecycle status --scope all |
   | heavy | ≤ 10s | 显式重活：export、fresh-clone prerequisites、聚合 verify Full |
   | toolchain-bound | 豁免但须声明 | test --profile（go test/race 主导）、fresh-clone（clone+build）、release |

   豁免不是免检：toolchain-bound 命令必须在 help/COMMANDS.md 标注
   "耗时由 Go 工具链/网络决定"，且其**自身开销**（引擎调度、报告写盘）计入 standard 预算。
2. **等级登记处 = typed command catalog**：`CommandDescriptor` 增加
   `LatencyClass` 字段（catalog 是命令唯一权威，等级跟命令走，不另建表）。
   `newCommandCatalog` 校验：每个命令必须声明等级（缺失即启动 panic——沿用 HelpForm 纪律）。

### B. 防回退门禁（复用 doctor perf，不新建机制）

3. `doctor perf` 扩展：对每个 fast/standard 等级命令实测 3 次取中位数
   （进程内调用 handler，避免进程启动噪音），超预算 1.5 倍 → Warn，超 3 倍 → Fail。
   Warn 阈值防机器差异误报（性能门禁变成不稳定测试比没有更糟——Full 优化战役的既有结论）。
4. 已知超支项立案而不放宽：`lifecycle status --scope all` 2.15s 与 `doctor --all` 2.68s
   在 standard 内但接近上限——用 `--json` 输出各 adapter 的 `elapsedMs`（已有字段）定位
   最慢 adapter，若单 adapter >1s 则在其域内开 Fast Path（先测后改，测量数据入 todo 再动手）。

### C. pull 暴露面收敛（环境因素的架构应对）

5. **provision 写入 git 传输优化配置**（git-native：配置进 .git/config，随 0011 一起做）：

   ```text
   fetch.parallel = 0            # git 自动并行度
   submodule.fetchJobs = 4       # 子模块并行 fetch
   core.fscache = true           # Windows 文件系统缓存（Git for Windows）
   ```

6. **子模块按需而非默认递归**：文档明确 `git pull`（不带 --recurse-submodules）是
   日常路径——skills 子模块只在 lifecycle install/update 时同步（其入口已存在）。
   `doctor --all` 增加一条检查：submodule.recurse 若被全局设为 true 则 Warn 提示
   （它让每次 pull 都拖 18MB 子模块的 fetch 协商）。
7. **不做**：浅克隆子模块（shallow submodule 与 pin 前移工作流冲突，历史上坑多）、
   partial clone（对 81MB .git 收益不值复杂度）、镜像/代理建议（属用户网络环境，
   文档一行带过即可）。

## 明确不做

- 不为达标砍验证内容（Full 优化战役红线延续）。
- 不做每命令的持续 benchmark CI（doctor perf 本地门禁足够；CI 机器差异大）。
- 不承诺冷启动/网络路径的数字（环境因素，声明豁免）。

## 自测（可信任方式）

```powershell
go test ./internal/cli/... ./internal/repohealth/...
# 等级完整性：catalog 每个命令都有 LatencyClass（缺失应 panic，负例验证后撤销）
bin\aicoding.exe doctor perf --json          # 输出各命令实测 vs 预算，全绿
# 防回退负例：临时在某 fast 命令里 sleep 500ms → doctor perf 必须 Warn/Fail → 撤销
# pull 配置：
bin\aicoding.exe provision --json
git config --get fetch.parallel ; git config --get submodule.fetchJobs
# 实测对比（网络路径，记录但不作为通过判据）：
Measure-Command { git pull } ; Measure-Command { git pull --recurse-submodules }
bin\aicoding.exe test --profile Full --json
```

通过判据：catalog 等级全覆盖且启动校验生效（负例）；doctor perf 对注入的慢命令报警
（负例贴输出）；provision 后三项 git config 就位；COMMANDS.md 标注 toolchain-bound 豁免；
两次 pull 实测数字记入本 todo 作为基线（网络路径只记录不承诺）。

## 执行证据（2026-07-21，与 0022 刀 6 合并完成）

- A：typed command catalog 的 25 个顶层命令全部登记 `fast` / `standard` / `work`；构造缺失
  `LatencyClass` 的 route 时，`TestCommandCatalogRejectsMissingLatencyClass` 通过，启动期
  catalog 校验会拒绝该 route。
- B：`doctor perf --json` 对 14 个 fast/standard 代表路径各跑 3 次并全部 PASS。本轮实测
  `doctor --all` 中位数 351.807ms、`lifecycle status --scope all` 38.750ms、
  `governance dependencies` 157.952ms、`verify Smoke` 219.803ms。注入 fast 慢命令的真实
  负例输出为 `median=601ms budget=400ms status=warn` 与
  `median=1202ms budget=400ms status=fail`。
- C：本 worktree 的三项 local config 初始均为 `<missing>`；执行 `provision --json` 后为
  `fetch.parallel=0`、`submodule.fetchJobs=4`、`core.fscache=true`，第二次执行三个 action
  均为 `kept`。`TestInitRepairsGitTransportConfiguration` 还实际验证了错误值
  `1 / 1 / false` 会被纠偏为 `0 / 4 / true`。

网络耗时不作为本项硬门禁：日常 `git pull` 不递归同步 skills 子模块；需要更新依赖时仍由
lifecycle install/update 显式执行。没有引入浅克隆、partial clone、镜像或第二套性能预算。
