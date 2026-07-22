# TODO 0030: 冻结面提升（已实测验证的架构进 Frozen + 可执行断言）

Status: Done
Verify: FREEZE_AND_ACQUISITION_BOUNDARY 收录新增冻结条目；对应 FREEZE-00x 静态用例在 Full 全绿且负例可抓

> 来源：FORWARD_PLAN C2。LOOP_ENGINEERING / PLAN_MODE 架构与 ADR 0007/0010
> 均已经过完整实测 + 合入 main + 多轮回归，仍停在 Accepted。
> **冻结不是仪式，是给未来改动上门禁**：改冻结面必须走 ADR + 三条件 + plan approve。
> 这也是四象限"已知的未知 → 地基"晋升的第一次机器化实践（呼应 COMPOUNDING_KNOWLEDGE 草稿）。

## 晋升清单（本项裁决，Codex 不自行扩减）

| 对象 | 现状 | 晋升后 |
|---|---|---|
| `LOOP_ENGINEERING_ARCHITECTURE.md` | Accepted | **Accepted and Frozen** |
| `PLAN_MODE_ARCHITECTURE.md`（若存在；实际文件名以仓库为准） | Accepted | Frozen |
| ADR 0007 validation-evidence 的核心契约（Receipt 身份组成 / fail-closed / 不缓存失败） | ADR Accepted | 契约条目进 FREEZE 边界文档 |
| ADR 0010 pinned reference 的 source 字段语义 | ADR Accepted | 同上 |
| **暂不晋升**：COMPOUNDING_KNOWLEDGE（Draft，方向未落地）、capability registry schema（刚建，未满一个 release） | — | — |

## 实现计划

1. `FREEZE_AND_ACQUISITION_BOUNDARY.md` 增补上述条目（该文件在 docs/architecture/**，
   **走 plan approve**——本项自己过自己要求的门禁）。
2. 每条新冻结面配一条**可执行断言**（testengine 静态用例，沿用 FREEZE-00x 模式）：
   - FREEZE-004：`work run|prepare|step` 不在 typed catalog（loop 永不实现清单）；
   - FREEZE-005：`Decide` 签名四参数注入（AST 或文本断言函数签名未变）；
   - FREEZE-006：Receipt 身份组成字段集不变（对 fingerprint struct 的字段清单断言）；
   - FREEZE-007：kit-manifest source 字段仍为可选（required 集合不含 source）。
3. 两份架构文档头部 Status 改为 `Accepted and Frozen`，正文附一行
   "解冻走 ADR + 三条件"。
4. `docs/architecture/README.md` 阅读路径同步（冻结区条目更新）。

## 明确不做

- 不冻结 Draft / 未满一个 release 的东西（见清单"暂不晋升"）。
- 不因冻结而重写文档内容（只改 Status + 加断言，正文零改动）。
- 断言只做存在性/签名/字段集检查，不做语义解释（机器可判的最小面）。

## 自测

```powershell
# plan 门禁自证：本项动 docs/architecture/** → plan check 命中 → approve 后才动
bin\aicoding.exe plan check --staged --json
go test ./internal/testengine/...
bin\aicoding.exe test --profile Full --json     # FREEZE-004..007 出现且绿
# 负例（各验一条后撤销）：
#   往 catalog 加 "work run" → FREEZE-004 红
#   改 Decide 签名 → FREEZE-005 红
bin\aicoding.exe docsync all --json
```

通过判据：四条 FREEZE 断言在 Full 出现且绿；两条负例被抓；
plan approve 记录在案（approvedTree 非空）；文档 Status 已提升且正文零语义改动。
