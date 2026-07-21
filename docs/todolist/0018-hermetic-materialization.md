# TODO 0018: Hermetic 源码物化（Release 摆脱全量 clone —— docx §12/§14 采纳）

Status: Done
Verify: Release 的 hermetic 验证不再执行 git clone；FRESH-001 真 clone 保留在 CI schedule；两者证据字段可区分

> 来源：docx §3.3（Hermetic ≠ Cold）、§12（Materialization）、§14（Fresh Clone 定位）。
> docx 这一刀切得准：**FRESH-001 把两个不同的问题绑在一起付费**——
> "干净源码树上能否构建+验证"（每次 Release 都要答）与
> "远程传输+bootstrap 是否完好"（只在传输面变化时要答）。
> 前者不需要 clone；后者不需要每次。0016 的 FRESH-001 分步计时是本项立项证据。

## 实现计划

1. **`internal/kit` 内新增物化函数**（不新建包——它是 freshclone.go 的兄弟路径，
   放同包共享 overlay/校验逻辑）：
   - 输入：当前验证主体的 HEAD/INDEX Tree + submodule gitlink 清单（来自 `git ls-tree`，零网络；
     staged 验证必须测试 staged 内容，不静默退回旧 HEAD）；
   - 实现：`git archive --format=tar <tree>` 流入 Go 标准库 `archive/tar` reader，子模块逐个
     `git -C <submodule> archive`（本地对象库已有内容，**零网络**、不依赖平台 tar 解码）；
   - 输出：干净源码目录 + 位于源码树外的 `source-manifest.json`（tree OID、各子模块
     path+commit+tree、文件计数）——物化结果身份 = superproject tree + 所有递归 gitlink 的
     复合 digest；manifest 不污染“源码文件集 == Git blob 集”断言；
   - 保证（docx §12.2 全单照收）：无未跟踪文件、无 .git 状态、模式正确、目录独立。
2. **Release 流程切换**：FRESH-001 由唯一 testengine 的私有 `materialized` leaf 在
   **物化目录**执行 `freshCloneChecks` 的 Release 分支
   （build + release verify 不变，省掉 clone+submodule 网络往返）。
   报告字段 `sourceMode: "materialized"` 与旧 `"cloned"` 区分，证据不混淆。
3. **真 fresh-clone 降频不降级**（docx §14.3）：
   - 原 FRESH-001 的真 clone 命令从 Release profile 移出，ID 复用为 REQUIRED 的物化验证；
     真 clone 保留在 CI 的 schedule 任务
     （每周 cron 已存在）与 `fresh-clone` 显式命令；
   - 新增静态门禁 `FRESH-004`：成功的显式真 clone 在 Git common-dir 写单行 Tree advisory
     baseline；传输敏感路径（`.gitmodules`/`.gitattributes`/`.githooks/**`/bootstrap 代码）
     自该 baseline 以来有变化时，
     Release 内 **Warn 提示**"传输面已变化，建议显式跑 fresh-clone"——
     提示而非强制，避免把网络成本绑回本地路径。
4. **物化缓存暂不做**（docx §12.4 推迟）：`git archive` 对本仓库 ~2s 级，
   缓存复杂度（只读保护、reflink、清理）超过收益；0016 实测若证明 >5s 再立项。
5. 文档：LOOP/验证架构文档不动；`docs/operations/` 记录 Release 证据链变化
   （hermetic 定义从"clone 隔离"改为"物化隔离 + 周期性真 clone"）；CHANGELOG 明确
   这是证据语义变化，Release Receipt 的 plan digest 会因此失效（正确失效）。

## 明确不做

- 不删除真 fresh-clone 能力（降频，不降级）。
- 不做跨仓库共享物化缓存 / %LOCALAPPDATA% 全局缓存（docx §17.2 推迟）。
- 不改 Full profile（它已无 clone 类用例）。
- 不引入打包依赖（产物来自既有 Git，解包只用 Go 标准库）。

## 已实现的正确性负例（2026-07-22）

- 递归 superproject → submodule → nested submodule 夹具物化为 6 个 tracked 文件、2 个
  submodule manifest entry；根/子模块未跟踪文件、未暂存 tracked 内容、`.git` 与位于源码树外
  的 manifest 均未混入源码文件集；其中中文目录/文件名用于覆盖 Windows 路径解码，`../` tar
  entry 被 fail-closed 拒绝。身份为
  `sha256:e9ec4d12f41def6ced5696b001696e78ff3a4a4fbefa013fe7568eeaee8df407`。
- FRESH-004 实际负例：未暂存 `.gitmodules` 与 staged `.githooks/pre-commit` 分别返回
  `transport-sensitive paths changed since last successful fresh-clone`；同 baseline 之后只改 docs
  不提示；无 baseline 明确提示先运行显式 fresh-clone。
- 显式 `fresh-clone --profile Release --json` 仍为 `sourceMode=cloned` 并完整通过，墙钟
  `69871 ms`，其中 `git.clone=52429 ms`；baseline 写入与 `temp.release` 均成功。
- 真实仓库负例先后捕获 Windows bsdtar 中文路径解码失败，以及无 `.git` 物化树未显式传
  `--repo-root` 的失败；前者改由 Go 标准库读取 tar 流并增加中文/越界夹具，后者在保留现场
  重放通过后固化参数。两次均 fail-closed，未伪造成功。
- 最终 staged Tree `35fcc02fd045557aaf889b51d902021fd96f7fa0` 的
  `test --profile Release --reuse off` 为 67/67 PASS、0 WARN、0 SKIP。FRESH-001
  `sourceMode=materialized`，物化 1809 个 tracked 文件与 3 个递归 gitlink，耗时 `16184 ms`
  （materialize/build/verify/temp release = 4610/6697/3831/953 ms），命令/步骤无 clone，
  `keptTemp=false`；相对旧 Release 内 FRESH-001 `66956 ms` 下降 **75.8%**。

## 自测（可信任方式）

```powershell
go test ./internal/kit/...
# 物化正确性：
bin\aicoding.exe test --profile Release --reuse off --json
# 断言：FRESH 步骤 sourceMode=materialized、无 git clone 子进程（进程列表/步骤名核对）、
#       物化目录文件集 == git ls-tree -r 文件集（抽查对比脚本贴输出）
# 隔离性负例：工作区放一个未跟踪文件 → 物化目录必须不含它
# 性能对比（0016 基线表回填 cold-release/warm-release 两行）：
1..3 | % { (Measure-Command { bin\aicoding.exe test --profile Release --reuse off --json }).TotalSeconds }
# 真 clone 保留验证：
bin\aicoding.exe fresh-clone --profile Release --json      # 显式命令仍可用
grep -n "fresh-clone" .github/workflows/aicoding-ci.yml    # schedule 任务仍在
# 传输面变化提示：改 .gitmodules（临时）→ Release 出现 FRESH-004 Warn → 撤销
bin\aicoding.exe test --profile Full --json
```

通过结果：Release 无 clone 步骤；物化文件集与 tree 一致（含子模块）；未跟踪文件、中文路径
和路径越界负例通过；同 leaf 实测 `66.956s → 16.184s`；FRESH-004 提示负例通过；每周 CI
真 clone 未被移除。
