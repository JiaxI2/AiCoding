# PRD 选项：华为 C 规范全覆盖黄金示例

Decision Status: Selected

Selected option: Option D - 简单入口 + 公开高级样例目录。

## 目标

让初次使用者先看到一对有简单功能的 ISO C99 文件，同时让高级规则覆盖样例保持完整、公开可见，
并按以下闭环交付：

```text
黄金 demo -> 规则目录 -> JSON/schema -> lint/编译门禁 -> 行为与负向测试
```

完整 PDF 作为只读参考副本保存在 `references/`。根目录使用标准文件名 `AGENTS.md`，将 PDF 规则改写成 Agent 可执行约束，不逐字复制原文。

## 不可变约束

- 业务示例模块不超过 4 个，优先控制为 3 个。
- 每个 `.c` 有同名 `.h`；公共头文件自包含并具有唯一保护宏。
- 静态函数前置区只放函数原型；完整函数注释放在定义上方。
- 复杂函数先给编号控制流总览，主要逻辑段使用空行和段前意图注释。
- 非显然分支和每个有实际处理的 `switch case` 均说明领域意图；连续空标签例外，fallthrough 必须明确说明。
- 未知工号省略，代码内修改历史默认交给 Git/CHANGELOG；启用源文件历史必须由用户显式确认。
- 禁止为了“展示规则”引入真实的不安全实现。禁止项通过安全替代实现和负向 lint fixture 证明。
- PDF 每条原则、规则和建议都必须在规则目录中归类为 `demo`、`lint`、`compile`、`test` 或 `manual`，不允许出现未分类项。
- 用户明确要求和当前 C99 Skill 高于纯格式建议；冲突必须记录理由。

## Option D：简单入口 + 公开高级样例目录（当前选择）

```text
generated-demo/
  demo.c / demo.h                    简单采样平均值与等级判定
  advanced/
    README.md                        阅读导航和职责说明
    state_machine.c / .h             状态机、ISR/DMA 和临界区
    protocol.c / .h                  不可信输入和协议边界
    fixed_pool.c / .h                固定资源生命周期
    tests/advanced_test.c            公开行为测试
```

优点：根目录认知负担最低，高级规则证据仍对最终用户可见并保持职责分离。风险是生成文件总数没有减少，
但目录层次和文件名明确区分了“入门功能”和“完整覆盖”。

## Option A：三个职责模块（历史选择，已被用户反馈取代）

### 文件结构

```text
generated-demo/
  demo.c / demo.h                    核心状态机、ISR、DMA、边界和数值安全
  demo_protocol.c / demo_protocol.h  字节序、用户输入、字符串与二进制边界、安全格式化
  demo_pool.c / demo_pool.h          固定资源池、生命周期、释放后防护和资源复用
  tests/demo_test.c                  行为测试、边界测试和故障注入
```

安全宏、`static inline`、断言和行优先数组访问分布在上述模块中，不再增加专门的“规则展示模块”。

### 优点

- 三个模块分别对应状态、输入和资源三类真实变化点，职责清晰。
- 能自然覆盖动态内存、字节序、字符串、效率和并发等条件规则。
- 文件数量可控，适合作为 Agent 黄金示例。

### 风险

- 单个 demo 比当前版本明显增大。
- 需要扩展生成器，使其确定性生成 7 个文件。

### 验证

- GCC 和 Clang 严格 C99、`-Werror`、`-Wvla`、`-Wconversion`、`-Wshadow`、`-Wmissing-prototypes`。
- 三个公共头文件分别通过 C99 和 C++17 独立编译。
- 运行行为测试与 lint 负向 fixture。

## Option B：两个合并模块

### 文件结构

保留 `demo.c/.h`，把协议安全、固定资源池和诊断能力合并进 `demo_support.c/.h`，另加一个测试入口。

### 优点

- 文件最少，生成器改动相对小。

### 风险

- `demo_support` 会承担输入、资源、宏和诊断等多种职责，弱化 PDF 的单一职责与结构单一原则。
- 注释和规则展示容易集中成“规范大杂烩”。

## Option C：四个专用模块

在 Option A 基础上增加 `demo_diagnostics.c/.h`，单独承载日志、断言、格式化和故障注入接口。

### 优点

- PDF 规则到代码的映射最直观，lint fixture 与测试隔离最好。

### 风险

- 文件偏多，超出“不能太多”的优先方向。
- 诊断模块对于一个黄金示例可能显得过度设计。

## 推荐

采用 **Option D**。它保留 Option A 的三个高级职责模块，但解决顶层多个 `demo_*` 文件难以理解的问题，
并新增 VS Code 风格 snippets 作为用户自定义入口。

## 决策状态

已解决：用户明确要求高级规则覆盖样例对最终用户可见，因此不采用内部隐藏夹具；使用公开 `advanced/`。
