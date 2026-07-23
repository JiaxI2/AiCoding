# GitHub Actions Node 24 迁移原始证据

日期：2026-07-23
仓库：`F:\Study\AI\worktrees\AiCoding`

## 1. 官方来源验证方法

每个 action 均从其官方 GitHub 仓库执行以下等价步骤，不采信 prompt 或外部文本中的 SHA：

1. `gh api repos/<owner>/<repo>/releases/latest`，拒绝 draft/prerelease，并取得当前稳定 tag；
2. `gh api repos/<owner>/<repo>/git/ref/tags/<tag>` 解析精确 tag；
3. 若 ref 的 `object.type=tag`，继续调用
   `gh api repos/<owner>/<repo>/git/tags/<tag-object-sha>`，直到 peel 到 commit；
4. 对 `vN` major tag 做同样解析，要求它与精确 release tag 指向同一 commit；
5. 用 GitHub contents API 读取该 tag 的 `action.yml`，要求 `runs.using=node24`。

原始核验结果：

```text
ACTION=actions/checkout LATEST=v7.0.1 MAJOR=v7 COMMIT=3d3c42e5aac5ba805825da76410c181273ba90b1 USING=node24 URL=https://github.com/actions/checkout/releases/tag/v7.0.1
EXACT_CHAIN=ref:commit:3d3c42e5aac5ba805825da76410c181273ba90b1
MAJOR_CHAIN=ref:commit:3d3c42e5aac5ba805825da76410c181273ba90b1
ACTION=actions/setup-go LATEST=v7.0.0 MAJOR=v7 COMMIT=b7ad1dad31e06c5925ef5d2fc7ad053ef454303e USING=node24 URL=https://github.com/actions/setup-go/releases/tag/v7.0.0
EXACT_CHAIN=ref:commit:b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
MAJOR_CHAIN=ref:commit:b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
ACTION=actions/upload-artifact LATEST=v7.0.1 MAJOR=v7 COMMIT=043fb46d1a93c77aae656e7c1c64a875d1fc6a0a USING=node24 URL=https://github.com/actions/upload-artifact/releases/tag/v7.0.1
EXACT_CHAIN=ref:commit:043fb46d1a93c77aae656e7c1c64a875d1fc6a0a
MAJOR_CHAIN=ref:commit:043fb46d1a93c77aae656e7c1c64a875d1fc6a0a
ACTION=actions/github-script LATEST=v9.0.0 MAJOR=v9 COMMIT=3a2844b7e9c422d3c10d287c895573f7108da1b3 USING=node24 URL=https://github.com/actions/github-script/releases/tag/v9.0.0
EXACT_CHAIN=ref:tag:d746ffe35508b1917358783b479e04febd2b8f71 -> object:commit:3a2844b7e9c422d3c10d287c895573f7108da1b3
MAJOR_CHAIN=ref:tag:373c709c69115d41ff229c7e5df9f8788daa9553 -> object:commit:3a2844b7e9c422d3c10d287c895573f7108da1b3
ACTION=go-task/setup-task LATEST=v2.1.0 MAJOR=v2 COMMIT=01a4adf9db2d14c1de7a560f09170b6e0df736aa USING=node24 URL=https://github.com/go-task/setup-task/releases/tag/v2.1.0
EXACT_CHAIN=ref:commit:01a4adf9db2d14c1de7a560f09170b6e0df736aa
MAJOR_CHAIN=ref:commit:01a4adf9db2d14c1de7a560f09170b6e0df736aa
```

## 2. release-gate 不变量

迁移前后的两行原文及 UTF-8 SHA-256：

```text
SEED_EXACT=<        run: .\bin\aicoding.exe test --profile Release --reuse off --json>
SEED_SHA256=32a261f8da3dded57d44d53a2a490d5a2398d0124bd5280ccd9eab664ac55f05
AUDIT_EXACT=<        run: .\bin\aicoding.exe test --profile Release --verify-reuse --json>
AUDIT_SHA256=18b75214cc2f8b16569534110bf0391d961b2bc3dc80c2d84a60163cd786352b
```

## 3. setup-task 唯一规格锚点

`internal/testengine/engine_test.go` 的
`TestScheduledCISeedsAndAuditsReleaseBeforeDefaultPromotion` 是 release-gate 中 setup-task
完整 commit SHA 与 `# v2` 注释的唯一规格锚点。临时把 workflow 改成 40 个零的错误 SHA，
原始输出为：

```text
=== RUN   TestScheduledCISeedsAndAuditsReleaseBeforeDefaultPromotion
    engine_test.go:348: release-gate is missing pinned Task setup contract "uses: go-task/setup-task@01a4adf9db2d14c1de7a560f09170b6e0df736aa # v2"
--- FAIL: TestScheduledCISeedsAndAuditsReleaseBeforeDefaultPromotion (0.03s)
FAIL
FAIL    github.com/JiaxI2/AiCoding/internal/testengine    1.988s
FAIL
SETUP_TASK_ANCHOR_WRONG_VALUE_EXIT=1
```

还原前后 `.github/workflows/aicoding-ci.yml` 的 SHA-256 均为
`b2375d3fe0c315db3dfdd2be38b5812c4f5afcc7f87e1ebf32fad1e10497b477`；还原后同一测试
`PASS`、退出码 `0`。断言仍是确定字符串精确匹配，未删除、未放宽，也没有运行时绕过。
今后有意改变 setup-task pin 时必须同步这一锚点。

## 4. 远端运行

包含本迁移的 `main@21522e92225471450cf492c87a0c16fec95afd55` 经正常 push 和 pre-push
Receipt 门禁后，以 `workflow_dispatch` 真跑：

- AiCoding CI：[run 30001965694](https://github.com/JiaxI2/AiCoding/actions/runs/30001965694)
  结论 `success`；`smoke`、`release-gate`、`clean-clone-full`、
  `scheduled-doctor-perf` 四个 job 全部 `success`。
- Issue governance：
  [run 30001969328](https://github.com/JiaxI2/AiCoding/actions/runs/30001969328)
  结论 `success`；适用于 dispatch 的 `sync-labels` 为 `success`，仅适用于 issue 事件的
  `normalize-lifecycle` 按既有条件正常 `skipped`。

完整日志逐行检查的原始计数与实际 action 启动行：

```text
RUN=30001965694 LOG_LINES=6567 NODE20_DEPRECATION_MATCHES=0
##[group]Run actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1
##[group]Run actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
##[group]Run actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a
##[group]Run go-task/setup-task@01a4adf9db2d14c1de7a560f09170b6e0df736aa
RUN=30001969328 LOG_LINES=78 NODE20_DEPRECATION_MATCHES=0
##[group]Run actions/github-script@3a2844b7e9c422d3c10d287c895573f7108da1b3
```

匹配式覆盖 `Node 20`、`Node.js 20`、`node20` 以及同一行内的 Node/deprecated 组合；
两条完整日志均为零命中。
