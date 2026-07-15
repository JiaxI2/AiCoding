# 验证报告（2026-07-15）

## 结论

Option D 的独立 Kit 已完成：61 页 PDF 参考、可检索 Markdown、139 条规则目录、简单入口、
公开高级样例、VS Code 风格 snippets、第 8 章注释方法、可读性摘要、Go lint 正负例、
GCC/Clang 严格 C99、独立头文件、行为测试和外部 fast/full 验证全部通过。

## 需求逐项证据

| 要求 | 结果 | 直接证据 |
| --- | --- | --- |
| 尝试官方 Docker 转换 | 有执行证据；下载阶段阻塞后安全终止 | `tools/pdf-reference/Dockerfile.markitdown`、`references/CONVERSION_REPORT.md` |
| 使用官方转换器得到 Markdown | 通过 | Microsoft MarkItDown 0.1.6 的 `*.raw.md`，再由规范化脚本生成最终 `*.md` |
| Markdown 完整、可读 | 通过 | 61 页、0—16 章、139/139 条款、240 个 C 代码块；缺失 0、额外 0、重复页眉页脚 0 |
| PDF 整体作为正式参考 | 通过 | `references/*.pdf`、`*.raw.md`、`*.md`、SHA-256 和公开分发状态 |
| PDF 规则作为 Agent 约束 | 通过 | 根 `AGENTS.md` 覆盖头文件、函数、命名、变量、宏、安全、注释、格式、测试和可移植性 |
| demo 覆盖规范全范围 | 通过 | `docs/RULE_CATALOG.md`：139 条及 3 组非编号内容全部 `covered`，未分类 0 |
| demo 易于理解 | 通过 | 顶层只有简单 `demo.c/.h`；高级状态机、协议和固定池集中在公开 `advanced/` |
| 高级样例最终用户可见 | 通过 | `advanced/README.md` 导航 3 对 C/H 与公开行为测试，不使用内部隐藏夹具 |
| 静态前置声明只注册函数 | 通过 | 三个 `.c` 前置区无 Doxygen；lint 规则 `documentation.private-prototype` |
| 定义处完整函数注释 | 通过 | lint 规则 `documentation.definition-details` 及所有函数定义的 `@details` |
| 第 8 章文件头字段 | 通过 | 9 个 C/H 文件均含版权、版本、日期、作者、内容、功能和关系；未知工号省略，源码修改历史由 Git/`CHANGELOG.md` 管理 |
| 复杂函数注释层级 | 通过 | `docs/COMMENTING_METHOD.md`、`documentation.function-flow`；定义处编号总览，函数体按空行、逻辑段和领域意图组织 |
| 非显然分支与 `case` | 通过 | `comment.case-intent`、`comment.case-fallthrough`、黄金 Demo；连续空标签例外，语义准确性保留人工评审 |
| 简单对象宏注释 | 通过 | `simpleObjectCommentStyle=block`；对象式宏使用独占一行的普通 `/* ... */` |
| 公开声明性能和重入约束 | 通过 | lint `documentation.performance`、`documentation.reentrancy` |
| 全局数据详细注释 | 通过 | `s_protocol_version` 说明取值范围、只读所有权和访问限制；lint `documentation.global-variable` |
| 每个 case/default 有注释 | 通过 | lint `comment.case-intent`；核心状态机全部分支均有中文意图注释 |
| 头文件保护宏 | 通过 | `DEMO_H` 与三个 `ADVANCED_*_H` 保护宏均唯一 |
| snippets 自定义 | 通过 | 9 个 VS Code 兼容片段可列举/渲染；`init` 安装且默认不覆盖用户修改 |
| JSON 和 schema | 通过 | `c-kit`、snippets、139 条规则目录和 external verify target 分别通过 checked-in schema |
| lint 正例、负例与可读性 | 通过 | 9 个 C/H 黄金文件零诊断；负例命中预期规则 ID；JSON summary 输出复杂函数、扇出和人工评审项 |
| 指针返回函数识别 | 通过 | 覆盖 `TYPE *Function`、`TYPE **Function`、星号分隔写法和表达式误判负例；PDO 的 19 个函数全部进入调用图 |
| GCC 严格 C99 | 通过 | GCC 14.2，`-Werror -Wvla -Wconversion -Wsign-conversion -Wshadow -Wmissing-prototypes -Wstrict-prototypes -Wformat=2` |
| Clang 严格 C99 | 通过 | Clang 17 前端使用相同严格告警；本机 RISC-V 发行版通过 MinGW host target 语法编译 |
| 头文件独立 C99/C++17 | 通过 | 4 个头分别通过 GCC/Clang C99 和 G++/Clang++ C++17，共 16 个探针 |
| 行为、边界、故障注入 | 通过 | GCC 可执行测试覆盖状态、字节序、校验、字符串、格式、容量、陈旧句柄和重复释放 |
| 外部 fast/full 验证 | 通过 | fast 6 步约 1.25 秒；full 11 步约 2.75 秒并比较 baseline/candidate stdout；输入使用只读快照，未调用 TI/CCS、`gmake` 或固件构建 |
| PDO 候选验收 | 通过 | full 11 步约 2.11 秒，0 条 lint 诊断；GCC/Clang C99、C/C++17 头探针、host test 和 baseline/candidate 行为等价通过 |
| PowerShell 质量 | 通过 | Runtime、AST、安全和 PSScriptAnalyzer（warning 也失败）全部通过 |
| 集成 AiCoding | 通过 | 受控快照位于 `CodingKit/tools/c-userstyle-kit`；Kit registry 与既有 C99 Skill 路由已接入；skills submodule 保持 `4fd28b47...` 且 clean |

## 已执行主门禁

```powershell
./scripts/verify.ps1
```

主门禁九个阶段全部返回 0。离线 Markdown 链接检查为 0 个错误。
`scripts/verify.sh` 已通过 MSYS2 `sh -n` 语法检查，但没有在真实 Linux/macOS 主机执行。

Patch Kit 的定向检查也证明旧版顶层高级文件均不存在、冗余静态原型注释计数为 0、Markdown
离线检查为 0 个错误；其总状态只因本目录不是 Git 仓库而无法执行 `git diff --check`，
不是内容门禁失败。

## 明确未做

- 没有声称 Docker 镜像构建成功；两次 Docker 构建都在下载阶段无进展，转换改由同版本官方包完成。
- 没有修改或刷新 AiCoding 插件、Marketplace、插件缓存或 skills submodule；C Kit 按仓库架构作为 `CodingKit/tools` 资产接入。
- 没有把 PDO 候选文件接入 TI/CCS 固件工程；外部验证仅使用 C Kit 的主机 lint、编译和 harness。
- 当前通用 `verify` 不把“候选头与基线头 ABI 类型等价”作为独立门禁；本次四个公开函数原型已人工逐项核对，并通过 GCC 严格 C99 双头同译单元兼容探针。候选头本身另已通过四种 C/C++ 探针。
- PDO host test 尚未覆盖 Tx 非字节对齐、非零 OD 位偏移，以及 Rx 编译成功但 Tx 编译失败时保留旧 Active 计划的组合；现有测试和代码审计未发现对应实现错误。
- 原始独立 Kit 目录不是 Git 仓库；集成后的受控快照由 AiCoding 父仓库统一提供 Git diff、commit、Tag 与 Release 证据。

## 回滚

AiCoding 集成的回滚边界是删除 `CodingKit/tools/c-userstyle-kit` 受控快照、Kit manifest/registry
登记以及既有 C99 Skill 的 verify 适配，并恢复对应文档与测试。原始独立 Kit、PDO 原文件和
`pdo_dynamic_refactored.c/.h` 候选文件均位于本仓库之外，不属于本次发布提交。
