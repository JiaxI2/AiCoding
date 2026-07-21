# TODO 0016: 验证观测与解释（时间分解 + explain —— 测试架构 docx 阶段一）

Status: Done
Verify: bin/aicoding.exe test --profile Release --json 输出各用例 setup/execute/persist 分解；validation explain 给出 Receipt miss 的具体失效字段

> 来源：《AiCoding 测试架构设计.docx》评审采纳项之一。docx 的核心裁决见
> staging README「测试架构 docx 总裁决」。本项对应 docx §19（可观测性）+
> §16.4（explain）+ §22.1（阶段一：先建时间分解，不改行为）——
> **它是 0017 节点级 Receipt 的前置：没有时间分解证据，不许拆节点。**

## 背景

docx §19.2 说得对：`Release completed in 76s` 是不可行动的输出。当前 Result 已有
`duration_ms`（单用例总耗时），缺的是**用例内分解**（排队/准备/执行/落盘）与
**miss 原因的字段级定位**（现在 `validation check` 只说 "no reusable Receipt exists"，
不说**哪个 digest 变了**）。

## 实现计划

1. **用例级时间分解**（testengine，加字段不改语义，全部 omitempty 向后兼容）：
   `Result` 增 `queue_ms / setup_ms / execute_ms / persist_ms`；
   `summary` 增 `slowest_cases`（Top5 id+ms，省得每次手写 jq）。
2. **Release 阶段分解**：FRESH-001（Release 内最重）的 `FreshCloneStep` 已有
   逐步结构，把每步 `elapsed_ms` 补上（clone / submodule / overlay / build / verify
   各占多少——0018 物化改造的立项证据就从这来）。
3. **`validation explain --profile P --target HEAD|INDEX --json`**（新子命令）：
   - Receipt hit：输出命中的 identity 与各 digest；
   - Receipt miss：**逐字段对比**当前 fingerprint 与最近一份同 profile Receipt
     （按 mtime 取最近，明确标注"仅诊断参考"）：

     ```json
     { "decision": "miss",
       "changed": [
         { "field": "subjectTreeOID", "old": "a1b2…", "new": "c3d4…" },
         { "field": "toolchainDigest", "old": "…", "new": "…" } ],
       "unchanged": ["validationPlanDigest", "engineSemanticDigest", "…"] }
     ```

   - 复用既有 store 读取路径与完整性校验，**explain 是只读诊断，不影响 check 的
     O(1) 快路径**（check 不做对比扫描，explain 才做）。
   - 同步 HelpForm + COMMANDS.md。
4. **缓存指标进 summary**：`cache_hit_ratio`（用例级，0017 前恒为 0/1 整体值）、
   `receipt_invalid_reason`（有 miss 时）。
5. **基线场景固化**（docx §19.4 采纳）：`docs/operations/VALIDATION_EVIDENCE_BUDGET.md`
   追加固定八场景表（cold-full / warm-full / one-go-file / docs-only / lifecycle-change /
   cold-release / warm-release / fresh-clone-release），本项跑第一轮填入；
   此后架构级修改必须重跑同表（0014 的 doctor perf 管命令级，此表管 profile 级，互补不重复）。

## 明确不做

- 不改任何执行行为/顺序（纯观测，docx §22.1 的纪律）。
- 不做节点拆分（0017）、不做物化（0018）。
- explain 不做跨 profile 对比、不做历史趋势（单次诊断够用；趋势属 CI 报表域）。

## 自测（可信任方式）

```powershell
go test ./internal/testengine/... ./internal/validationevidence/... ./internal/cli/...
bin\aicoding.exe test --profile Release --reuse off --json
# 断言：每个 command 用例四段分解之和 ≈ duration_ms（±5%）；FRESH-001 各步有 elapsed_ms
bin\aicoding.exe validation explain --profile Release --target HEAD --json    # hit 路径
Add-Content README.md "`n0016 docs-only explain probe"
git add README.md
bin\aicoding.exe validation explain --profile Release --target INDEX --json
# 断言：changed 只列 subjectTreeOID（docs 改动不动 toolchain/plan digest）——这就是
# 0017 要的立项证据：README 改动使整个 Release Receipt 失效，但其余 fingerprint 输入没变
git restore --staged --worktree README.md
# 确定性：explain 连续两次输出字节一致（剔除 elapsed）
bin\aicoding.exe test --profile Full --json
```

通过判据：分解之和校验通过；explain 对 docs-only 改动精确指认 treeOID 单字段变化
（贴输出）；八场景基线表填入实测数；explain 不改变 check 的耗时（前后各测 5 次中位数比对）。
