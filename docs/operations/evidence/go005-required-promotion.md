# GO-005 Required 晋升原始证据

日期：2026-07-23
仓库：`F:\Study\AI\worktrees\AiCoding`

## 1. 晋升前零 finding

命令：

```powershell
go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...
```

完整原始输出与退出码：

```text
=== STATICCHECK_STDOUT_STDERR_BEGIN ===
=== STATICCHECK_STDOUT_STDERR_END ===
STATICCHECK_EXIT=0
```

随后只把 `internal/testengine/engine.go` 中 GO-005 的 `Severity` 从 `WarnOnly` 改为
`Required`，并删除已兑现的
`Note: "WarnOnly for one release before promotion to Required"`。固定版本
`honnef.co/go/tools/cmd/staticcheck@v0.7.0` 未改，GO-002 继续保持 WarnOnly。

## 2. 唯一规格锚点与鉴别力

`internal/testengine/engine_test.go` 的
`TestRegistryPinsStaticcheckAndGovulncheckPolicy` 是 GO-005 severity、空 Note 与固定命令的
唯一规格锚点。最终值真跑通过；临时把实现改为第三种 `Severity("BROKEN")` 后，原始输出为：

```text
=== RUN   TestRegistryPinsStaticcheckAndGovulncheckPolicy
=== PAUSE TestRegistryPinsStaticcheckAndGovulncheckPolicy
=== CONT  TestRegistryPinsStaticcheckAndGovulncheckPolicy
    engine_test.go:246: GO-005 policy mismatch: testengine.TestCase{ID:"GO-005", Category:"GO", Title:"Staticcheck 静态分析", Node:"go", Severity:"BROKEN", Profiles:[]string{"full", "release"}, Kind:"command", Command:[]string{"go", "run", "honnef.co/go/tools/cmd/staticcheck@v0.7.0", "./..."}, TimeoutKind:"long", ExpectJSON:false, OptionalPath:"", NetworkFailureWarn:false, Note:""}
--- FAIL: TestRegistryPinsStaticcheckAndGovulncheckPolicy (0.02s)
FAIL
FAIL    github.com/JiaxI2/AiCoding/internal/testengine    2.141s
FAIL
GO005_ANCHOR_WRONG_VALUE_EXIT=1
```

还原前后 `internal/testengine/engine.go` 的 SHA-256 均为
`8c038955b80bb02766dc3b09257a523c28ce61c2d8235b5318268c7a4b4dc1d9`，还原后同一测试
`PASS`、退出码 `0`。断言没有删除或放宽；今后若有意改变 GO-005 severity、Note 或命令，
必须同步这一确定值锚点。

## 3. Staticcheck 真实负例

在原本无正式变更的 `internal/todolist/todolist.go` 中临时加入自赋值，直接运行固定
Staticcheck 的完整原始输出为：

```text
=== NEGATIVE_STATICCHECK_STDOUT_STDERR_BEGIN ===
internal\todolist\todolist.go:125:2: self-assignment of staticcheckProbe to staticcheckProbe (SA4018)
exit status 1
=== NEGATIVE_STATICCHECK_STDOUT_STDERR_END ===
NEGATIVE_STATICCHECK_EXIT=1
```

同一单点破坏下，Full 的结构化原始结果摘要与 GO-005 输出为：

```text
{"profile":"full","conclusion":"FAIL","total":73,"pass":64,"fail":4,"warn":1,"skip":4,"go005_status":"FAIL","go005_severity":"REQUIRED","go005_exit":1,"go005_command":"go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...","summary":"F:\\Study\\AI\\worktrees\\AiCoding\\test-results\\0043-negative-full\\summary.json"}
--- GO-005 stdout ---
internal\todolist\todolist.go:125:2: self-assignment of staticcheckProbe to staticcheckProbe (SA4018)
--- GO-005 stderr ---
exit status 1
```

Release 的结构化原始结果摘要与 GO-005 输出为：

```text
{"profile":"release","conclusion":"FAIL","total":73,"pass":68,"fail":4,"warn":1,"skip":0,"go005_status":"FAIL","go005_severity":"REQUIRED","go005_exit":1,"go005_command":"go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...","summary":"F:\\Study\\AI\\worktrees\\AiCoding\\test-results\\0043-negative-release\\summary.json"}
--- GO-005 stdout ---
internal\todolist\todolist.go:125:2: self-assignment of staticcheckProbe to staticcheckProbe (SA4018)
--- GO-005 stderr ---
exit status 1
```

这两次负例发生在同步文档与 TODO 尚未写入时，所以 DOC-001、GIT-008、HEALTH-001 也处于
Required FAIL，GO-003 为 WARN；上述汇总没有把它们隐藏成“只有 GO-005 失败”。GO-005 的
独立 leaf 证据仍明确为 `status=FAIL / severity=REQUIRED / exit_code=1`，且 stdout 精确指出
文件、行列与 SA4018。

## 4. 按字节还原

`internal/todolist/todolist.go` 在探针前后的 SHA-256 均为
`eb5045d8d6ecc5d133db0a6e740f1654ff405954d106e48810cf9f048250e8de`。还原后的原始检查为：

```text
=== RESTORED_STATICCHECK_STDOUT_STDERR_BEGIN ===
=== RESTORED_STATICCHECK_STDOUT_STDERR_END ===
STATICCHECK_PROBE_PATH_DIFF_EXIT=0
RESTORED_STATICCHECK_EXIT=0
```
