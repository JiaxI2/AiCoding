# Acceptance Plan: Contract Freeze And Acquisition Boundary

验收人：独立于实现者的审查会话。全部通过才签收；任一 FAIL 打回并附证据。

## Phase 0：范围

- [ ] diff 半径不超出 IMPLEMENTATION_PLAN 文件清单；
- [ ] 零 CLI 命令变化、零 schema 必填项/语义变化、零已冻结文档契约改动
  （交叉引用行除外）；
- [ ] 两个新 check 在实现提交上零违规（冻结既成现实，无行为迁移）。

## Phase 1：静态审查

- [ ] 新架构文档 §4 两条门禁规格与 dependencies.go 实现一一对应；
- [ ] `acquisitionBoundary` 节与 schema 一致，allowlist 恰为 `.gitmodules` 与
  `config/skill-sources.json` 两项（不多不少）；
- [ ] T1 语料审计结果在交付说明中，badge/文档/schema-id URL 未被 pattern 误伤；
- [ ] 四项冻结声明（§2.1–2.4）表述完整：冻结面、规则、演进出口三要素齐备。

## Phase 2：门禁执行（全部退出码 0 且 ok=true）

```powershell
go build ./...
go test ./internal/governance/... ./internal/cli/...
bin\aicoding.exe governance dependencies --json    # 含两个新 check 且 OK
bin\aicoding.exe docsync all --json
bin\aicoding.exe test --profile Smoke --json
bin\aicoding.exe test --profile Full --json
```

## Phase 3：突变验证（注入违规 → 门禁失败 → 还原全绿）

| # | 注入 | 期望失败点 |
|---|---|---|
| M1 | 在 `config/mcp/components/visio-mcp.json` 的 `runtime.env` 加入含 `https://` 的值 | `activation manifests URL-free` error，定位到文件与 JSON path |
| M2 | 在 `config/codex-kit.json` 任一字符串值写入 `https://github.com/x/y.git` | 双杀：URL-free 与 `cloneable sources registry` 至少一条 error |
| M3 | 把一个可克隆 URL 放入 `config/kits/` 下新建 json | `cloneable sources registry` error |

- [ ] M1–M3 均按期望失败且信息可定位；还原后门禁恢复全绿、无残留文件。

## Phase 4：行为等价与抽查

- [ ] `lifecycle status --scope all --json` ok=true（登记/manifest 解码路径无回归）；
- [ ] 离线激活抽查（手工，非门禁）：断网状态下对已本地化组件执行
  `lifecycle plan --scope mcp --action update --component visio-mcp --json`
  成功；若 install 因 pip 下载失败，确认与 §3.4 已识别偏差描述一致并记录。

## Phase 5：文档收尾

- [ ] CHANGELOG 条目存在；identity 零版本号；
- [ ] 新文档链接可解析；handbook §8 文档地图含新行；docsync 通过即视为登记完成。

## 签收

通过后：新架构文档 Status → Accepted and Frozen；本计划 Plan Status → Approved；
勾满 TASKS.md；在下方追加签收记录（日期、基线/实现提交、Phase 2 摘要、M1–M3 结果）。

签收记录：

- （待填）
