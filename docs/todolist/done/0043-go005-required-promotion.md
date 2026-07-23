# TODO 0043: GO-005 晋升 Required

Status: Done
Verify: go test ./internal/testengine/... -count=1 && go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...

## 范围

- GO-005 的 Severity 从 WarnOnly 晋升为 Required，并删除已兑现的 one-release Note。
- Staticcheck 继续固定为 `honnef.co/go/tools/cmd/staticcheck@v0.7.0`。
- GO-002 继续按 ADR 0013 保持 WarnOnly；不晋升其他 leaf。

## 冻结规格锚点

`internal/testengine/engine_test.go` 的
`TestRegistryPinsStaticcheckAndGovulncheckPolicy` 是 GO-005 severity、空 Note 与固定命令的
唯一规格锚点。本轮只把被有意改变的期望同步为 `Required`/空 Note，保持确定值精确比较。
第三种 `Severity("BROKEN")` 已真实使该测试失败，证明锚点没有放宽。今后同类有意变更必须
同步这一锚点。

## 实施证据

- 晋升前 Staticcheck：exit `0`，stdout/stderr 为空。
- 临时 SA4018：`internal/todolist/todolist.go:125:2`，直接 Staticcheck exit `1`。
- Full：GO-005 为 `FAIL / REQUIRED / exit 1`，
  `test-results/0043-negative-full/summary.json`。
- Release：GO-005 为 `FAIL / REQUIRED / exit 1`，
  `test-results/0043-negative-release/summary.json`。
- 探针文件前后 SHA-256 相同，`git diff --exit-code -- internal/todolist/todolist.go` 为 `0`；
  还原后 Staticcheck 再次 exit `0`、零 finding。
- 完整原始输出见
  [GO-005 Required 晋升原始证据](../../operations/evidence/go005-required-promotion.md)。

## 完成条件

- GO-005 正例、锚点鉴别力和 Full/Release 负例已留原始证据。
- 最终 Release、docsync、governance、plan、todolist 与 hooks 全绿。
- 最终 Release summary：`test-results/0046-final-release/summary.json`。
