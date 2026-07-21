# TODO 0017: 节点级 Receipt（四大节点分域失效 —— 测试架构 docx 核心采纳项）

Status: Done
Verify: 只改 docs 后 test --profile Full 复用 go 域节点 Receipt；--verify-reuse 逐节点审计绿

> 来源：docx §7/§8/§10 + §22.4。**这是曾被裁决"不实施"的 Full 优化阶段六的复活**——
> 当时拒绝理由是"结果缓存的静默假 PASS 风险不值得承担"。现在前提变了：
> validationevidence 已提供内容寻址 Receipt、fail-closed store、`--verify-reuse` 审计
> 三件套，node receipt 是**同一机制在更细粒度的第二次应用**，不是新发明。
> **前置硬条件：0016 的时间分解与 explain 先落地；docx §22.8 的路径级增量本项仍不做。**

## 设计裁决（对 docx 的修正）

| docx 原文 | 本项裁决 | 理由 |
|---|---|---|
| §7.2 十七个节点 | **先四个域节点**：`go`（unit+race+vet）/ `docsync` / `governance` / `lifecycle-readonly` | docx 自己也说"初期不应拆得过细"；四域对应 0016 实测的耗时大头 |
| §23 新建 internal/{target,validation,receipt,…} 八包 | **零新包**：节点 = testengine 内的用例分组；Receipt 复用 validationevidence store，以 `node + nodeInputDigest` 扩展 fingerprint，profile 仍为既有三档 | 已有实现就是 docx 要的东西，照 §23 重建即平行事实源 |
| §16.1 `aicoding validate …` 新命令面 | **不改命令**：仍是 `test --profile`，复用行为对用户透明 | CLI 面已冻结；第二命令结构违反一主题一权威 |
| §9.5 失败 Receipt 短 TTL 缓存 | **仍不缓存失败**（延续既有裁决） | 失败重跑成本 = 诊断价值；TTL 机制复杂度不值 |
| §10.1 路径级输入范围 | 节点输入范围**粗粒度声明**（go 域 = `**/*.go + go.mod + go.sum + testdata/**`），digest 用 `git ls-tree` 对声明路径的子树哈希，**不逐文件 SHA-256** | git 已有子树哈希；逐文件扫描违反"不采用全仓逐文件哈希"非目标 |

## 实现计划

1. **节点划分登记在 Registry**（用例结构加 `Node string` 字段，约定：未标注 = `repo` 域，
   任何仓库变化都失效——保守默认，fail-closed）：
   - `go`：GO-001/002/003/004/005/006
   - `docsync`：DOC-001/002/004
   - `governance`：GIT-002…009、GOV 类静态
   - `lifecycle-readonly`：LIFE-*、RC-*、KIT 结构类
2. **节点输入 digest**：每个节点声明路径集合；单次
   `git ls-tree -r -z --full-tree <tree>` 批量取得 tracked mode/type/OID/path，按域过滤、排序后
   摘要，不逐文件读内容或计算 SHA-256。工作区 dirty 时节点复用整体禁用（沿用 subject 可复用
   判定，不放宽）。
3. **节点 Receipt**：identity = 现有 fingerprint 组成 + `nodeName + nodeInputDigest`
   替换 subjectTreeOID 位（profile 内节点身份与整树解耦——这就是"README 改动不失效
   go 节点"的机制）。存储路径 `receipts/<profile>/nodes/<node>/<identity>.json`，
   复用原子写与完整性校验。
4. **执行流程**：test 运行时逐节点查 Receipt → hit 的节点标记 `reused-from-node`
   （Result.Reason 注明），miss 的节点正常执行 → 全部节点结论聚合后，整树 Receipt
   照旧生成（**节点 Receipt 是加速层，整树 Receipt 仍是对外唯一凭证**——push gate、
   alias、plan gates 只认整树，节点层不对外）。
5. **`--verify-reuse` 扩展到节点级**：审计模式逐节点重跑并比对节点 resultsDigest；
   CI 的 release-gate 审计路径自动覆盖。
6. **默认开关**：节点复用跟随 `--reuse` 总开关（off 时完全不查节点），
   晋级纪律沿用 ADR 0007 §5（三次远端绿灯）。

## 明确不做

- 不做路径级/文件级增量（docx §22.8 自己也放最后）。
- 不做节点间 DAG 调度（当前串行执行顺序不变；调度属 Full 优化阶段三，仍按当时裁决挂起）。
- 不新增任何对外命令/不改变 push gate 的整树语义。
- 失败节点不入 Receipt。

## 自测（可信任方式）

```powershell
go test ./internal/testengine/... ./internal/validationevidence/...
bin\aicoding.exe test --profile Full --reuse auto --json          # 种子
echo x >> README.md ; git add -A ; git commit -m "docs: touch"
bin\aicoding.exe test --profile Full --reuse auto --json
# 断言：go/lifecycle 节点 reused-from-node，docsync/governance 正常执行（README 在其输入域）
# 反向：改一个 .go 文件 → go 节点 miss、docsync 节点 hit
bin\aicoding.exe test --profile Full --verify-reuse --json        # 节点级审计绿
# 污染负例：手工篡改一份节点 Receipt 的 resultsDigest → verify-reuse 必须 FAIL → 清理
# 整树语义不变：
bin\aicoding.exe validation check --profile Full --target HEAD --json   # 仍以整树为准
git log --oneline -1 ; git revert --no-edit HEAD                  # 清理 README 提交
```

通过判据：docs-only 改动后 Full 耗时显著下降且 go 节点 reused（贴前后耗时对比，
预期 ~18s → <8s，实测回填）；.go 改动正确反向失效；节点审计抓污染负例；
push gate / alias 行为与本项前完全一致（回归断言）。

## 实现与探针验收（2026-07-21）

- 未新增包、runner、报告 schema、Registry 或公共命令；`validationevidence` 只增加
  `DeriveNodeFingerprint`、`CheckNode`、`PutNode` 三个操作，公共 API 保持 13 个操作加
  `Error()` 的上限。
- Full `--reuse off` 种子为 63 PASS / 0 FAIL / 0 WARN / 3 profile SKIP，engine
  `243663 ms`、外部 `244897 ms`。
- README-only clean Tree 以 `--reuse auto` 命中 `go` 6 个和 `lifecycle-readonly` 9 个用例，
  `cache_hit_ratio=15/63`；engine `11654 ms`、外部 `12853 ms`，相对种子下降 94.8%。
- Go-only clean Tree 仅命中 `docsync` 的 DOC-001/002/004，`cache_hit_ratio=3/63`；GO-001…006
  全部真实执行并通过，证明反向失效正确。该轮 engine `438047 ms`，因系统负载不同不作性能基线。
- 正向 `--verify-reuse` 为 63 PASS、`VALIDATION_RECEIPT_HIT`；把当前 Go 节点 Receipt 的
  `resultsDigest` 手工改为全零后，审计以 `EVIDENCE-NODE-GO` 和
  `VALIDATION_REUSE_AUDIT_MISMATCH` 失败。随后逐字节恢复，SHA-256 与备份同为
  `C6D392656CCDBCB4E1B54658E0932E9D5DF4794F32086658111C3A37FDF67344`。
- 恢复后 `validation check --profile Full --target HEAD --json` 在 230 ms 内命中整体 Receipt
  `sha256:88d9f9c948a9949fd3125bf21a8b2b1fa82ea1be394c9525b1b63048eb116078`；节点层没有进入
  alias/push gate。原始报告保存在 ignored `test-results/0017-node-receipts-20260721/`。
- Status 切换前的主工作树 staged INDEX 最终执行 `test --profile Full --reuse off --json`：Tree
  `59b2b9017b1321318b32c8ab2ea79c6e2b7e9c8b`，63 PASS / 0 FAIL / 0 WARN / 3 profile SKIP，
  engine `307208 ms`、CLI `308385 ms`，整树 Receipt
  `sha256:313259689407323a785bf98eacd47b7af40d4fb6b3c60c95d8df672a49bb4402`。
