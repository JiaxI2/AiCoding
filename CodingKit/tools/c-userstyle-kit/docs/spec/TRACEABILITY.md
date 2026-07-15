# 可追溯性：华为 C 规范全覆盖黄金示例

| 用户要求 | 计划产物 | 验证证据 |
|---|---|---|
| PDF 可检索且可读 | `references/*.md` | 61 页、0—16 章、139 条款、240 个 C 代码块，缺失/额外均为 0 |
| demo 易于理解且覆盖 PDF 全范围 | 简单入口 + 公开 `advanced/` | 根目录只有 `demo.c/.h`；139 条编号条款及 3 组非编号内容全部有证据 |
| PDF 规则供 Agent 使用 | `AGENTS.md` | Agent 规则链接与内容扫描 |
| PDO 风格的分层注释方法 | `docs/COMMENTING_METHOD.md`、高级黄金样例 | 复杂函数编号总览；逻辑段空行和意图注释；分支与 case 说明 |
| 工号和修改历史项目策略 | `AGENTS.md`、snippets、黄金模板 | 未知工号省略；代码内历史默认禁用；原始 PDF/Markdown 不修改 |
| JSON 表达规则 | `examples/c-kit.json`、schema | schema 验证与 Go 配置测试 |
| 用户自定义模板 | `examples/c-snippets.json`、schema、`snippet` 命令 | 9 个片段可列举；常用变量和编号占位符可渲染 |
| lint 阻塞可机器规则 | `internal/cuserstyle/lint.go`、fixture | 9 个 C/H 黄金文件零诊断；负例命中指定规则 ID |
| 可读性与职责评审 | `readability.go`、`lint --summary --json` | 输出复杂度、调用扇出和单调用静态 helper 人工评审项；不把语义判断伪装成 lint |
| 外部候选文件验证 | `cstylekit verify`、target schema、`docs/EXTERNAL_VERIFICATION.md` | fast 秒级反馈；full 增加双编译器、C++17 头文件和 baseline/candidate 行为等价；不调用固件工具链 |
| 编译门禁 | `scripts/verify.ps1`、`scripts/verify.sh` | GCC 与 Clang 严格 C99；四个头分别通过两套 C99/C++17 探针 |
| 测试闭环 | Go 测试和 `advanced/tests/advanced_test.c` | 生成器/lint 测试与 GCC 行为、边界、故障注入测试 |
| 注释语义评审边界 | `docs/COMMENTING_METHOD.md`、规则目录 | 格式由 lint；领域意图和重构判断由 demo + manual |
| 集成 AiCoding | `CodingKit/tools/c-userstyle-kit`、Kit registry、既有 C99 Skill 路由 | C Kit fast verify、AiCoding Smoke/Full/Release；skills submodule 保持只读且 clean |
