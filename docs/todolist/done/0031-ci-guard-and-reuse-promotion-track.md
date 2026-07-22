# TODO 0031: CI 防回退 + 复用晋级轨道（doctor perf 进 schedule；release-gate 绿灯累计启动）

Status: Done
Verify: CI schedule 含 doctor perf 且一次远端运行绿；main 上 release-gate 首次正式计数运行完成并记录 run URL（1/3）

> 来源：FORWARD_PLAN O2 + O5。**O2 是"冷 Release 也快"的最后一步**：
> `--reuse auto` 晋级后日常 Release 走 Receipt 命中（实测 392ms）。
> 晋级条件已由 ADR 0007 §5 冻结：main 上 release-gate **连续 3 次**成功
> （每次 off seed + verify-reuse 审计），切换必须是独立评审提交并引用 3 个 run URL。
> **本项不做晋级本身**——只把轨道铺好、把第一盏绿灯点亮、把计数显性化。

## 实现计划

### A. doctor perf 进 CI（O5，防回退）

1. `aicoding-ci.yml` 的 schedule 任务（每周 cron 已存在）追加一步：
   `bin\aicoding.exe doctor perf --json`，超预算按既有 1.5×Warn/3×Fail 语义，
   Fail 使 job 红。artifact 上传 perf 输出。
2. **不加到 PR/push 路径**（CI 机器差异大，schedule 一周一次足够防回退；
   这是 0014 时已定的"不做每命令持续 benchmark CI"的延续）。

### B. release-gate 绿灯累计启动（O2 轨道）

1. 确认 main 上 release-gate job（workflow_dispatch + schedule 已接线）在
   **当前 main tip** 触发一次正式运行：off seed → `--verify-reuse` 全量审计 →
   test-results artifact 上传。
2. 运行绿后，在 `docs/operations/VALIDATION_EVIDENCE_BUDGET.md` 追加
   "晋级计数"小节：run URL、SHA、结论 —— **计数显性化，靠文档不靠记忆**。
   格式：`1/3: <run-url> @ <sha> PASS`。
3. **明确记录**：此前 feature 分支上的 dispatch 均不计入（ADR 0007 §5 原文要求
   main）；toolchain 变更不重置计数（ADR 0007 已注记）。
4. 后续两次由 schedule 自然产生或手动 dispatch；**凑满 3/3 后另开独立提交晋级默认值**
   （不在本项内做，本项 Verify 只要求 1/3 落账）。

## 明确不做

- **不在本项切换 `--reuse off → auto`**（那是证据凑齐后的独立评审提交）。
- 不为凑绿灯放宽审计（verify-reuse 全量重跑 + resultsDigest 比对不变）。
- 不加 PR 路径的 perf 门禁。

## 自测

```powershell
# A：
git diff .github/workflows/aicoding-ci.yml     # schedule 含 doctor perf 步骤
# 远端 dispatch 一次 schedule 等价物，贴 run URL 与 perf artifact
# B：
# 触发 main release-gate，贴 run URL；确认 off seed + verify-reuse 两段均绿
# BUDGET.md 出现 "1/3: <url> @ <sha> PASS"
bin\aicoding.exe docsync all --json ; bin\aicoding.exe test --profile Smoke --json
```

通过判据：doctor perf 远端一次绿 + artifact 在；release-gate 1/3 落账且 run URL 可访问；
默认值仍为 `--reuse off`（grep 确认，防止本项越权提前晋级）。

## 完成证据（2026-07-22）

- schedule-equivalent doctor perf：<https://github.com/JiaxI2/AiCoding/actions/runs/29896483333>
  在 feature tip `9890b667bfdc54ef5fafe49d27c736210ad13732` PASS，并上传
  `doctor-perf-evidence`。
- 正式 main 计数：<https://github.com/JiaxI2/AiCoding/actions/runs/29900035150>
  在同一 SHA 完成 `--reuse off` 冷种子与 `--verify-reuse` 全量审计，全部 job PASS，
  `release-gate-evidence` 已上传。
- `internal/cli/test.go` 与 `internal/testengine/engine.go` 的 flag/fallback 均仍以
  `ReuseOff` 为默认值；本项未做复用晋级。
