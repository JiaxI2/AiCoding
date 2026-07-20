# TODO 0008: Kit 架构文档还债（c-userstyle-kit 全量 + ai-debug-kit 边界卡）

Status: Done
Verify: docs/architecture/C_USERSTYLE_KIT_ARCHITECTURE.md 与 docs/architecture/AI_DEBUG_KIT_BOUNDARY.md 存在、含 Status 头、过 docsync all

> 依赖 0007（模板骨架已入 docs/architecture/README.md）。
> Kit 全景结论：c-userstyle-kit（order 75）与 loop 同级别，裁决"什么代码算合格"，
> 却零架构文档；ai-debug-kit 是 Kit 级资产但未注册，有上游开源项目可借鉴，
> **只需一页边界卡防走歪，不写全量架构**。

## 实现计划

### A. `C_USERSTYLE_KIT_ARCHITECTURE.md`（全量，用 LOOP 模板骨架）

Status: Accepted and Frozen（它已在产线，行为已稳定）。章节：

1. **结论**：C99 风格裁决面。谁拥有规则（`config/skills/c99-standard-c/`）、
   谁执行（`CodingKit/tools/c-userstyle-kit` 的 cstylekit + clang-format）、
   谁消费（skill 命令 / pre-commit / C99-00x 测试用例）。
2. **系统位置图**：CLI `skill c99-standard-c {status,templates,fmt,check,verify}` →
   internal/cstyle → 外部 go run cstylekit → clang-format；标注哪段是
   external-command、哪段是 builtin。
3. **边界判据**：与 docsync（文档 vs 代码风格）、与 governance lint 的分工；
   scope 语义（changed/staged/all/paths）与 gitx 的关系。
4. **数据契约**：`config/skills/c99-standard-c/skill.json`、clang-format.yaml 投影
   （C99-005 校验的 source-of-truth 链）、report.Result 字段。
5. **可靠性与安全**：clang-format 版本指纹（toolchain digest 是否覆盖？如未覆盖
   在"演进边界"记录）、排除目录策略（C99-006）、外部进程失败时的行为。
6. **演进边界**：明确不做（不扩语言、不做自动重构、规则变更必须走什么流程）。

写法要求：**只描述已存在的行为，逐条给出代码/配置锚点**（文件:行 或命令），
不发明新机制。写完它就是 0007 FREEZE 断言的候选来源。

### B. `AI_DEBUG_KIT_BOUNDARY.md`（一页边界卡，Status: Accepted Direction）

只写四条，各一段：

1. **位置与形态**：`CodingKit/tests/ai-debug-kit/`（Python + uv），含嵌套子模块
   `_external/Mklink-AI-Probe`（pin 策略：跟随上游 stable tag，前移须评审）。
2. **不进 Go 控制面**：不注册 kit-registry、不加 CLI 命令、不被 testengine 引用，
   直到它有明确的平台消费者并走 ADR。
3. **不承担门禁**：任何 verify/test profile 不依赖它的输出。
4. **上游同步纪律**：借鉴的开源项目变更 → 先在上游/fork 验证 → 更新 pin →
   本仓库只动 gitlink。禁止把上游代码复制进本仓库改（防走歪的核心一条）。

## 明确不做

- 不给 ai-debug-kit 写全量架构（有上游可借鉴，边界卡足够）。
- 不在本项改任何代码/配置，纯文档 + 锚点核对。
- common-control-kit 的文档欠账**单独开 todo**（其职责需要先和 owner 确认，不猜）。

## 自测（可信任方式）

```powershell
# 锚点核对：文档中引用的每个文件路径必须真实存在
# （写个一次性脚本 grep 出所有 `path:line` 锚点逐个 Test-Path，贴输出）
bin\aicoding.exe docsync all --json          # 新文档接入索引后绿
bin\aicoding.exe governance layout --json
bin\aicoding.exe skill c99-standard-c status --json    # 文档描述与实际输出比对
bin\aicoding.exe skill c99-standard-c verify --profile fast --json
bin\aicoding.exe test --profile Smoke --json
git status --porcelain
```

通过判据：两份文档 Status 头齐全（0007 门禁自动覆盖）；C_USERSTYLE 文档中每个
命令示例可实际执行且输出与描述一致（抽 3 条贴输出）；边界卡不超过一页（~60 行）。
