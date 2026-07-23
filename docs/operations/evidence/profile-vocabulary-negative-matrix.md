# TODO 0036 profile 词汇与子命令 catalog 负例矩阵

日期：2026-07-23
仓库：`F:\\Study\\AI\\worktrees\\AiCoding`

## 1. 修复前 help/runtime 不一致

基线：clean `main`，commit `c901c05`，使用该提交已检入的 `bin\\aicoding.exe`。

命令：

```powershell
bin\aicoding.exe kit verify --help
bin\aicoding.exe kit verify --all --profile Full
```

原始输出：

```text
Usage: aicoding kit verify [options]

Options:
  -all
        all enabled kits
  -json
        json output
  -kit string
        kit id
  -profile string
        Smoke, Full or Release (default "Smoke")
  -repo-root string
        repository root
  -with-state
        include stable install state summary
HELP_EXIT=0
[FAIL] kit verify handles Smoke/Lifecycle only; use aicoding skill verify --all --profile Full (163 ms)
  - kit verify handles Smoke/Lifecycle only; use aicoding skill verify --all --profile Full
RUNTIME_EXIT=2
```

后续各项只在实现后的工作树执行，每次注入单一破坏后立即还原。

## 2. 修复后 help/runtime 一致性

先真跑 canonical help 与行为：

```powershell
bin\aicoding.exe kit verify --help
bin\aicoding.exe kit verify --all --level smoke
```

原始输出：

```text
Usage: aicoding kit verify [options]

Options:
  -all
        all enabled kits
  -json
        json output
  -kit string
        kit id
  -level string
        smoke or lifecycle (default "smoke")
  -repo-root string
        repository root
  -with-state
        include stable install state summary
HELP_EXIT=0
[OK] kit smoke verify (7 ms)
  [OK] aicoding-platform                      smoke
  [OK] docsync-plus                           smoke
  [OK] reuse-governance                       smoke
  [OK] common-control-kit                     smoke
  [OK] c-userstyle-kit                        smoke
  [OK] release-governance-overlay-kit         smoke
RUNTIME_EXIT=0
```

随后单点把 `runKit` 的 canonical flag 临时破坏为独立
`--profile` help `Smoke, Full or Release`，运行门禁：

```text
=== RUN   TestFreezeChecksCurrentRepository
=== RUN   TestFreezeChecksCurrentRepository/FREEZE-009
    freeze_test.go:29: runKit declares --profile help outside the product vocabulary catalog
--- FAIL: TestFreezeChecksCurrentRepository (0.05s)
    --- FAIL: TestFreezeChecksCurrentRepository/FREEZE-009 (0.05s)
FAIL
FAIL    github.com/JiaxI2/AiCoding/internal/testengine    1.861s
FAIL
GATE_EXIT=1
```

还原后 `FREEZE-008/009` 原始输出：

```text
=== RUN   TestFreezeChecksCurrentRepository
=== RUN   TestFreezeChecksCurrentRepository/FREEZE-008
=== RUN   TestFreezeChecksCurrentRepository/FREEZE-009
--- PASS: TestFreezeChecksCurrentRepository (0.04s)
    --- PASS: TestFreezeChecksCurrentRepository/FREEZE-008 (0.02s)
    --- PASS: TestFreezeChecksCurrentRepository/FREEZE-009 (0.02s)
PASS
ok      github.com/JiaxI2/AiCoding/internal/testengine    1.805s
RESTORED_EXIT=0
```

## 3. catalog 外可路由子命令

在 `Execute` 的 catalog guard 之前临时加入 `kit shadow` 直接成功分支。先证明它确实可路由：

```text
COMMAND=go run ./cmd/aicoding kit shadow
STDOUT=<empty>
ROUTE_RUN_EXIT=0
```

同一破坏下运行 FREEZE-008：

```text
=== RUN   TestFreezeChecksCurrentRepository
=== RUN   TestFreezeChecksCurrentRepository/FREEZE-008
    freeze_test.go:29: cli.Execute contains catalog-external argv route "shadow"
--- FAIL: TestFreezeChecksCurrentRepository (0.01s)
    --- FAIL: TestFreezeChecksCurrentRepository/FREEZE-008 (0.01s)
FAIL
FAIL    github.com/JiaxI2/AiCoding/internal/testengine    1.807s
FAIL
GATE_EXIT=1
```

该分支已立即还原；§2 的 restored 输出证明还原后门禁转绿。

## 4. 兼容窗口旧参数

三种旧形式逐条真跑；均成功且 warning 写入既有 `report.Result.warnings`：

```text
COMMAND=bin\aicoding.exe kit verify --all --profile Lifecycle
[OK] kit lifecycle structure verify (359 ms)
  ! deprecated: kit verify --profile is retired by ADR 0012; use --level smoke|lifecycle
EXIT=0

COMMAND=bin\aicoding.exe kit test --all --profile Smoke
[OK] kit smoke test (5 ms)
  ! deprecated: kit test --profile Smoke is retired by ADR 0012; omit --profile
  [OK] aicoding-platform                      smoke
  [OK] docsync-plus                           smoke
  [OK] reuse-governance                       smoke
  [OK] common-control-kit                     smoke
  [OK] c-userstyle-kit                        smoke
  [OK] release-governance-overlay-kit         smoke
EXIT=0

COMMAND=bin\aicoding.exe skill c99-standard-c verify --profile fast
[OK] C99 Standard C skill C Kit verification (18789 ms)
  ! deprecated: skill c99-standard-c verify --profile is retired by ADR 0012; use --depth fast|full
EXIT=0
```

## 5. 第四套 `--profile` 词汇

把唯一 `productProfileVocabulary` 临时注入 `Canary` 后运行 FREEZE-009：

```text
=== RUN   TestFreezeChecksCurrentRepository
=== RUN   TestFreezeChecksCurrentRepository/FREEZE-009
    freeze_test.go:29: product --profile vocabulary changed: got [Smoke Full Release Canary]; want [Smoke Full Release]
--- FAIL: TestFreezeChecksCurrentRepository (0.10s)
    --- FAIL: TestFreezeChecksCurrentRepository/FREEZE-009 (0.10s)
FAIL
FAIL    github.com/JiaxI2/AiCoding/internal/testengine    1.866s
FAIL
GATE_EXIT=1
```

词汇表已立即还原；§2 的 restored 输出证明还原后门禁转绿。

## 6. pluginview quickstart 真跑

`kit describe --all --json` 从 command catalog 投影出 6 条 quickstart，
`inputDigest=sha256:c7c7f871c162f4df3bf0bb7e01496ccac87acba51e2514416377f1339f66cfe4`。
逐条按输出原文执行并解析原始 JSON 的 `ok` 与 `planDigest`：

```text
COMMAND=aicoding lifecycle status --scope kit --kit aicoding-platform --json EXIT=0 OK=True PLAN=sha256:983d504456dfd8ab896496384e6af36b8fcf9ba4b0b2fd85fd1ed945f4343e29
COMMAND=aicoding lifecycle status --scope kit --kit docsync-plus --json EXIT=0 OK=True PLAN=sha256:ca5590ccca21f21e1cc1532c38681203839c659d97a1d09989db160d48e25d85
COMMAND=aicoding lifecycle status --scope kit --kit reuse-governance --json EXIT=0 OK=True PLAN=sha256:0c81e1fd0a4afe6cdd6236b3896ba602718b56f7348bf4921ea9d4ecab2e3c8f
COMMAND=aicoding lifecycle status --scope kit --kit common-control-kit --json EXIT=0 OK=True PLAN=sha256:c52fef7223e49c4875ebdc43e44fa321bced4f624bc80a95d1950790c805173e
COMMAND=aicoding lifecycle status --scope kit --kit c-userstyle-kit --json EXIT=0 OK=True PLAN=sha256:dc17935a1c1e1f6776d264747a9be29171f56b7419fd4a4cf9232804b856d9c2
COMMAND=aicoding lifecycle status --scope kit --kit release-governance-overlay-kit --json EXIT=0 OK=True PLAN=sha256:5b6be35f7e5be9235bb7b7e8897f4b05382d143db1c3169ecc94dc6abb0e2551
```

## 7. Full、Release 与传输基线

Full 修复后真跑结果：

```text
COMMAND=bin\aicoding.exe test --profile Full --reuse off --allow-dirty --out test-results\0036-final-full --json
CONCLUSION=PASS TOTAL=73 PASS=69 FAIL=0 WARN=0 SKIP=4
SUMMARY=test-results/0036-final-full/summary.json
EXIT=0
```

Release 首轮与 fresh-clone 后复跑均没有 REQUIRED 失败，但因 `Taskfile.yml` 仍未暂存而保留
同一个 advisory；不能把它误记成全绿：

```text
COMMAND=bin\aicoding.exe test --profile Release --reuse off --allow-dirty --out test-results\0036-final-release --json
CONCLUSION=PASS_WITH_WARNINGS TOTAL=73 PASS=72 FAIL=0 WARN=1 SKIP=0
WARNING=FRESH-004: transport-sensitive paths changed since last successful fresh-clone: Taskfile.yml
EXIT=0

COMMAND=bin\aicoding.exe fresh-clone --profile Release --json
SOURCE_MODE=cloned SOURCE_TREE_OID=5edd0b5af96f511e9aa84cf86ddf15b5e513ca9f
STEPS=temp,git.source-tree,git.clone,git.submodule,worktree.overlay,go.build,release.verify,transport.baseline,temp.release
ALL_STEPS_OK=true EXIT=0
```

dirty 运行时，基线文件与 Release subject Tree 均为
`5edd0b5af96f511e9aa84cf86ddf15b5e513ca9f`；`CheckFreshCloneTransportDrift` 仍会把
`git diff` 返回的未暂存路径追加到 Tree diff。全部正常暂存后又真跑一次 fresh-clone 和
Release，进一步得到：

```text
FRESH_CLONE_SOURCE_TREE=5edd0b5af96f511e9aa84cf86ddf15b5e513ca9f
RELEASE_SUBJECT_MODE=index
RELEASE_SUBJECT_TREE=fdf89ab9b3ce071cdcd31e8d08c7d091ddbb1d7b
CONCLUSION=PASS_WITH_WARNINGS TOTAL=73 PASS=72 FAIL=0 WARN=1 SKIP=0
WARNING=FRESH-004: transport-sensitive paths changed since last successful fresh-clone: Taskfile.yml
EXIT=0
```

因此“暂存即可消除”不成立：fresh-clone 的 `sourceTreeOID`/transport baseline 取当前 HEAD，
而测试 subject 取 index Tree。只有本批提交使 `Taskfile.yml` 进入 HEAD，再以新 HEAD 成功运行
fresh-clone 后，FRESH-004 才具备清零前提。提交后的 clean-tree Full/Release 继续覆盖写入本节
列出的固定 summary 路径。
