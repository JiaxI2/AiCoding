# Release Governance Overlay

这个 overlay 为 AiCoding 仓库加入三个能力：

1. Tag / Release 命名空间治理。
2. `Taskfile.yml` 统一人和 Agent 的高效入口。
3. 后续缓存、增量验证和性能历史的接口规划。

## 1. 应用后新增入口

```powershell
task setup
task smoke
task perf
task full
task release
task skills
task rollback
task tag:audit
task tag:plan
task tag:verify
```

Taskfile 只做命令路由，不承载复杂业务逻辑。

## 2. Tag 治理脚本

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Audit -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Plan
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Verify
```

`Plan` 只生成纠偏命令，不会自动创建或 push tag。动态审计报告默认写入 `.aicoding/reports/release-governance/`，不作为长期文档提交。

## 3. 性能闭环后续方向

预留目录：

```text
.aicoding/cache/
```

计划文件：

```text
kit-index.json
manifest-hash.json
staged-files.json
governance-hash.json
perf-history.jsonl
```

计划命令：

```powershell
bin\aicoding.exe impact --staged --json
bin\aicoding.exe verify smart --staged --json
bin\aicoding.exe doctor perf --save --json
```

缓存只用于 Smoke / Hook；Full / Release 不使用缓存。

## 4. 安全边界

本 overlay 不执行：

- tag 删除
- force push
- remote tag 覆盖
- release 删除
- DSS / XDS / reset / halt / run / flash / erase / write-memory
