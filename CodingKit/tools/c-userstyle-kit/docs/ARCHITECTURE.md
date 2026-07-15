# C UserStyle Kit 架构

## 单一证据链

```text
61 页 PDF
   │  Microsoft MarkItDown 0.1.6 + 规范化 + PDF 对照检查
   ▼
可检索 Markdown（0—16 章，139 条款）
   │
   ├── AGENTS.md：Agent 行为约束
   ├── COMMENTING_METHOD.md：复杂函数、逻辑段和分支注释方法
   ├── 规则目录 JSON/Markdown：每条绑定 demo/lint/compile/test/manual
   ├── c-kit.json + schema：规则和门禁机器合同
   └── c-snippets.json + schema：用户可编辑代码片段
             │
             ├── Go 生成器 ──► 简单 demo + 公开 advanced 样例
             ├── Go lint ─────► 正例零诊断 + 负例预期规则 ID
             ├── GCC/Clang ───► 严格 C99 零告警
             ├── C/C++ 探针 ─► 每个头文件独立 C99/C++17
             └── 行为测试 ───► 正常、边界、故障注入
```

## 简单入口与高级职责模块

| 模块 | 单一职责 | 主要规范证据 |
| --- | --- | --- |
| 根 `demo` | 有界采样平均值和等级判定 | 参数校验、静态函数原型、整数上界、case 注释和头文件保护宏 |
| `advanced/state_machine` | 状态机和中断共享快照 | 状态迁移、临界区、数组/整数边界和错误码 |
| `advanced/protocol` | 不可信数据边界 | 字节序、字符串结尾、格式串、符号/截断、二进制长度 |
| `advanced/fixed_pool` | 固定资源生命周期 | 无动态分配、固定资源库、清理、重复释放和旧句柄拒绝 |

测试只通过四个公开头文件调用接口，不访问静态函数。`advanced/README.md` 明确导航三个高级职责；
无法安全放入正例的禁止项由 lint 负例和规则目录证明。

## 规则执行边界

- `demo`：适合用安全正例表达的规则。
- `lint`：语法稳定、低误报且能给出明确修复方向的规则。
- `compile`：标准、类型、格式串、告警、头文件自包含和 C++ 兼容。
- `test`：运行时边界、错误返回、状态、资源和故障注入。
- `manual`：职责、命名语义、注释准确性、架构耦合等需要判断的规则。

注释规则采用分层证据：lint 只检查可稳定判断的形式、位置、空行和分支结构；复杂函数总览是否
覆盖真实职责、分支注释是否说明领域意图，由 `AGENTS.md`、`docs/COMMENTING_METHOD.md`、黄金样例
和人工评审共同保证。禁止仅凭注释关键词声称完成语义验证。

139 条编号条款和第 0/16 章等非编号内容都必须有证据，但不要求也不允许把所有语义规则伪装成
正则 lint。`tools/rules/build_rule_catalog.py --check` 保证分类和证据不丢失。

## 生成器

黄金文件作为 Go `embed.FS` 模板随二进制编译，输出顺序和路径固定。生成器会清理未修改的旧版顶层
`demo_protocol.*`、`demo_pool.*` 和测试文件，但保留用户修改过的同名文件；Go 单元测试直接对嵌入模板
执行完整 lint。

`c-snippets.json` 保持 VS Code snippet 根对象格式。CLI 只实现当前模板需要的文件名、日期、编号默认值
和 `$0` 等常用占位符子集；完整 choice/transform 语法交给 VS Code，避免复制编辑器全部语义。

## staged Hook

`lint --scope staged` 保留原有 Git index 快路径：

```text
git diff --cached --unified=0  -> 修改行范围
git show :path                -> 实际待提交文本
```

文件级契约对被修改文件检查一次，行级规则只检查修改范围。当前目录不是 Git 仓库时应使用
`--scope files`；这不影响完整验证脚本。

## AiCoding 集成边界

本 Kit 作为 AiCoding-owned 确定性工具保存在 `CodingKit/tools/c-userstyle-kit`，由 Kit registry 管理，
并通过既有 `aicoding skill c99-standard-c` 用户入口调用。它不修改 skills submodule、生成插件、
Codex 插件缓存或 Marketplace 路径；AiCoding 的 Smoke、Full、Release 与文档同步门禁负责集成验收。
