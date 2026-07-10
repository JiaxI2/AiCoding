# Release Governance Overlay

这个 overlay 为 AiCoding 仓库加入三个能力：

1. Tag / Release 命名空间治理。
2. `Taskfile.yml` 统一人和 Agent 的高效入口。
3. 性能测量入口和动态报告边界。

## 1. 应用后新增入口

```powershell
task setup
task smoke
task ci
task full
task release
task test:latest
task style:c:status
task style:c:templates
task fmt-check:c
task fmt-check-staged:c
```

Taskfile 只做命令路由，不承载复杂业务逻辑。

## 2. Tag 治理脚本

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Audit -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Plan
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Verify
```

`Plan` 只生成纠偏命令，不会自动创建或 push tag。动态审计报告默认写入 `.aicoding/reports/release-governance/`，不作为长期文档提交。

## 3. 性能闭环边界

当前默认性能入口是：

```powershell
bin\aicoding.exe doctor perf --json
```

动态报告写入 `.aicoding/reports/` 时不作为长期文档提交。cache 和 smart verify 不属于本轮收口范围；Full / Release 不使用本地热路径优化。

## 4. 安全边界

本 overlay 不执行：

- tag 删除
- force push
- remote tag 覆盖
- release 删除
- DSS / XDS / reset / halt / run / flash / erase / write-memory
