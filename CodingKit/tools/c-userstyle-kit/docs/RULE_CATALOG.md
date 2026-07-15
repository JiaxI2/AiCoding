# 华为 C 语言编程规范完整规则目录

> 本文件由 `tools/rules/build_rule_catalog.py` 从规范 Markdown 的条款标题机械生成。
> 条款原文及解释以本地 PDF/Markdown 参考副本为准；本目录只提供检索、分类和验收证据。

## 覆盖结论

- PDF：61 页；章节：0—16。
- 可编号条款：139 条，全部已分类，未分类 0 条。
- 非编号内容：封面/范围/简介、第 0 章和第 16 章均有独立证据。
- 证据类型：`demo`、`lint`、`compile`、`test`、`manual`。
- `covered` 表示存在明确证据路径，不表示所有规范都适合用正则表达式机器判断。

## 证据方法

| 方法 | 含义 |
| --- | --- |
| `demo` | 黄金 C/H 代码以安全正例体现规则。 |
| `lint` | Go lint 对稳定、低误报的语法规则实施门禁。 |
| `compile` | GCC、Clang、C99 与 C++17 头文件严格编译。 |
| `test` | 公开行为、边界和故障注入测试。 |
| `manual` | 语义、架构、命名清晰度等必须保留人工评审。 |

## 非编号内容覆盖

| 范围 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- |
| 封面、修订声明、目录、范围和简介 | `manual` | `references/huawei-c-language-programming-standard-dkba-2826-2011-5.md:文档元数据与目录`<br>`tools/pdf-reference/verify_reference.py:页数、章节与噪声检查` | `covered` |
| 0 规范制订说明 | `manual` | `AGENTS.md:适用范围与权威来源`<br>`AGENTS.md:修改工作流`<br>`docs/spec/SELECTED_SOLUTION.md:规则优先级与证据策略` | `covered` |
| 16 业界编程规范 | `manual` | `references/huawei-c-language-programming-standard-dkba-2826-2011-5.md:第16章`<br>`AGENTS.md:编译、测试与安全输入`<br>`scripts/verify.ps1:严格工具链门禁` | `covered` |

## 编号条款覆盖

## 1 头文件

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-01-P-01 | 原则 1.1 头文件中适合放置接口的声明，不适合放置实现。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-P-02 | 原则 1.2 头文件应当职责单一。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-P-03 | 原则 1.3 头文件应向稳定的方向包含。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-R-01 | 规则 1.1 每一个.c文件应有一个同名.h文件，用于声明需要对外公开的接口。 | `compile` | `generated-demo/demo.c+h`<br>`generated-demo/advanced/state_machine.c+h`<br>`generated-demo/advanced/protocol.c+h`<br>`generated-demo/advanced/fixed_pool.c+h` | `covered` |
| HW-C99-01-R-02 | 规则 1.2 禁止头文件循环依赖。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-R-03 | 规则 1.3 .c/.h文件禁止包含用不到的头文件。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-R-04 | 规则 1.4 头文件应当自包含。 | `compile` | `scripts/verify.ps1:每个头文件独立 C99/C++17 编译` | `covered` |
| HW-C99-01-R-05 | 规则 1.5 总是编写内部#include保护符（#define 保护）。 | `compile` | `lint:file.include-guard`<br>`generated-demo/demo.h:#ifndef DEMO_H` | `covered` |
| HW-C99-01-R-06 | 规则 1.6 禁止在头文件中定义变量。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-R-07 | 规则 1.7 只能通过包含头文件的方式使用其他.c提供的接口，禁止在.c中通过extern的方式使用外部函数接口、变量。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-R-08 | 规则 1.8 禁止在extern "C"中包含头文件。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-S-01 | 建议 1.1 一个模块通常包含多个.c文件，建议放在同一个目录下，目录名即为模块名。为方便外部使用者，建议每一个模块提供一个.h，文件名为目录名。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-S-02 | 建议 1.2 如果一个模块包含多个子模块，则建议每一个子模块提供一个对外的.h，文件名为子模块名。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-S-03 | 建议 1.3 头文件不要使用非习惯用法的扩展名，如.inc。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |
| HW-C99-01-S-04 | 建议 1.4 同一产品统一包含头文件排列方式。 | `compile` | `generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`scripts/verify.ps1:header-gates`<br>`lint:file.*` | `covered` |

## 2 函数

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-02-P-01 | 原则 2.1 一个函数仅完成一件功能。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-P-02 | 原则 2.2 重复代码应该尽可能提炼成函数。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-R-01 | 规则 2.1 避免函数过长，新增函数不超过50行（非空非注释行）。 | `demo` | `lint:complexity.function-lines`<br>`AGENTS.md:新增函数有效代码不超过 50 行` | `covered` |
| HW-C99-02-R-02 | 规则 2.2 避免函数的代码块嵌套过深，新增函数的代码块嵌套不超过4层。 | `demo` | `lint:complexity.nesting`<br>`AGENTS.md:嵌套不超过 4 层` | `covered` |
| HW-C99-02-R-03 | 规则 2.3 可重入函数应避免使用共享变量；若需要使用，则应通过互斥手段（关中断、信号量）对其加以保护。 | `demo` | `generated-demo/advanced/state_machine.c:临界区快照和中断单写者协议` | `covered` |
| HW-C99-02-R-04 | 规则 2.4 对参数的合法性检查，由调用者负责还是由接口函数负责，应在项目组/模块内应统一规定。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-R-05 | 规则 2.5 对函数的错误返回码要全面处理。 | `demo` | `generated-demo/advanced/tests/advanced_test.c:全部错误返回码断言` | `covered` |
| HW-C99-02-R-06 | 规则 2.6 设计高扇入，合理扇出（小于7）的函数。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-R-07 | 规则 2.7 废弃代码（没有被调用的函数和变量)要及时清除。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-S-01 | 建议 2.1 函数不变参数使用const。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-S-02 | 建议 2.2 函数应避免使用全局变量、静态局部变量和I/O操作，不可避免的地方应集中使用。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-S-03 | 建议 2.3 检查函数所有非参数输入的有效性，如数据文件、公共变量等。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-S-04 | 建议 2.4 函数的参数个数不超过5个。 | `demo` | `lint:complexity.parameters`<br>`examples/c-kit.json:safety.maxParameters=5` | `covered` |
| HW-C99-02-S-05 | 建议 2.5 除打印类函数外，不要使用可变长参函数。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.function`<br>`AGENTS.md:函数设计` | `covered` |
| HW-C99-02-S-06 | 建议 2.6 在源文件范围内声明和定义的所有函数，除非外部可见，否则应该增加static关键字。 | `demo` | `lint:naming.private-function`<br>`generated-demo/:static prototypes` | `covered` |

## 3 标识符命名与定义

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-03-P-01 | 原则 3.1 标识符的命名要清晰、明了，有明确含义，同时使用完整的单词或大家基本可以理解的缩写，避免使人产生误解。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-P-02 | 原则 3.2 除了常见的通用缩写以外，不使用单词缩写，不得使用汉语拼音。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-R-01 | 规则 3.1 产品/项目组内部应保持统一的命名风格。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-01 | 建议 3.1 用正确的反义词组命名具有互斥意义的变量或相反动作的函数等。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-02 | 建议 3.2 尽量避免名字中出现数字编号，除非逻辑上的确需要编号。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-03 | 建议 3.3 标识符前不应添加模块、项目、产品、部门的名称作为前缀。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-04 | 建议 3.4 平台/驱动等适配代码的标识符命名风格保持和平台/驱动一致。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-05 | 建议 3.5 重构/修改部分代码时，应保持和原有代码的命名风格一致。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-06 | 建议 3.6 文件命名统一采用小写字符。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-R-02 | 规则 3.2 全局变量应增加“g_”前缀。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-R-03 | 规则 3.3 静态变量应增加“s_”前缀。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-R-04 | 规则 3.4 禁止使用单字节命名变量，但运行定义i、j、k作为局部循环变量。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-07 | 建议 3.7 不建议使用匈牙利命名法。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-08 | 建议 3.8 使用名词或者形容词＋名词方式命名变量。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-09 | 建议 3.9 函数命名应以函数要执行的动作命名，一般采用动词或者动词＋名词的结构。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-S-10 | 建议 3.10 函数指针除了前缀，其他按照函数的命名规则命名。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-R-05 | 规则 3.5 对于数值或者字符串等等常量的定义，建议采用全大写字母，单词之间加下划线„_‟的方式命名（枚举同样建议使用此方式定义）。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |
| HW-C99-03-R-06 | 规则 3.6 除了头文件或编译开关等特殊标识定义，宏定义不能使用下划线„_‟开头和结尾。 | `lint` | `lint:naming.*`<br>`generated-demo/`<br>`AGENTS.md:命名` | `covered` |

## 4 变量

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-04-P-01 | 原则 4.1 一个变量只有一个功能，不能把一个变量用作多种用途。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-P-02 | 原则 4.2 结构功能单一；不要设计面面俱到的数据结构。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-P-03 | 原则 4.3 不用或者少用全局变量。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-R-01 | 规则 4.1 防止局部变量与全局变量同名。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-R-02 | 规则 4.2 通讯过程中使用的结构，必须注意字节序。 | `demo` | `generated-demo/advanced/protocol.c:DEMO_ReadU16BigEndian/DEMO_ReadU32BigEndian` | `covered` |
| HW-C99-04-R-03 | 规则 4.3 严禁使用未经初始化的变量作为右值。 | `demo` | `compiler:-Wuninitialized`<br>`generated-demo/:定义时初始化` | `covered` |
| HW-C99-04-S-01 | 建议 4.1 构造仅有一个模块或函数可以修改、创建，而其余有关模块或函数只访问的全局变量，防止多个不同模块或函数都可以修改、创建同一全局变量的现象。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-S-02 | 建议 4.2 使用面向接口编程思想，通过API访问数据：如果本模块的数据需要对外部模块开放，应提供接口函数来设置、获取，同时注意全局数据的访问互斥。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-S-03 | 建议 4.3 在首次使用前初始化变量，初始化的地方离使用的地方越近越好。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-S-04 | 建议 4.4 明确全局变量的初始化顺序，避免跨模块的初始化依赖。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |
| HW-C99-04-S-05 | 建议 4.5 尽量减少没有必要的数据类型默认转换与强制转换。 | `demo` | `generated-demo/demo.c`<br>`generated-demo/advanced/protocol.c`<br>`compiler:-Wshadow,-Wconversion` | `covered` |

## 5 宏、常量

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-05-R-01 | 规则 5.1 用宏定义表达式时，要使用完备的括号。 | `lint` | `lint:macro.*`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`AGENTS.md:宏与常量` | `covered` |
| HW-C99-05-R-02 | 规则 5.2 将宏所定义的多条表达式放在大括号中。 | `lint` | `lint:macro.*`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`AGENTS.md:宏与常量` | `covered` |
| HW-C99-05-R-03 | 规则 5.3 使用宏时，不允许参数发生变化。 | `lint` | `lint:macro.*`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`AGENTS.md:宏与常量` | `covered` |
| HW-C99-05-R-04 | 规则 5.4 不允许直接使用魔鬼数字。 | `lint` | `lint:macro.*`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`AGENTS.md:宏与常量` | `covered` |
| HW-C99-05-S-01 | 建议 5.1 除非必要，应尽可能使用函数代替宏。 | `lint` | `lint:macro.*`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`AGENTS.md:宏与常量` | `covered` |
| HW-C99-05-S-02 | 建议 5.2 常量建议使用const定义代替宏。 | `lint` | `lint:macro.*`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`AGENTS.md:宏与常量` | `covered` |
| HW-C99-05-S-03 | 建议 5.3 宏定义中尽量不使用return、goto、continue、break等改变程序流程的语句。 | `lint` | `lint:macro.*`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h`<br>`AGENTS.md:宏与常量` | `covered` |

## 6 质量保证

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-06-P-01 | 原则 6.1 代码质量保证优先原则（1）正确性，指程序要实现设计要求的功能。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |
| HW-C99-06-P-02 | 原则 6.2 要时刻注意易混淆的操作符。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |
| HW-C99-06-P-03 | 原则 6.3 必须了解编译系统的内存分配方式，特别是编译系统对不同类型的变量的内存分配规则，如局部变量在何处分配、静态变量在何处分配等。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |
| HW-C99-06-P-04 | 原则 6.4 不仅关注接口，同样要关注实现。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |
| HW-C99-06-R-01 | 规则 6.1 禁止内存操作越界。 | `test` | `generated-demo/advanced/protocol.c:长度先验`<br>`generated-demo/advanced/fixed_pool.c:容量先验` | `covered` |
| HW-C99-06-R-02 | 规则 6.2 禁止内存泄漏。 | `test` | `generated-demo/advanced/fixed_pool.c:固定资源池，无堆分配` | `covered` |
| HW-C99-06-R-03 | 规则 6.3 禁止引用已经释放的内存空间。 | `test` | `generated-demo/advanced/fixed_pool.c:代际句柄`<br>`advanced_test.c:陈旧句柄测试` | `covered` |
| HW-C99-06-R-04 | 规则 6.4 编程时，要防止差1错误。 | `test` | `generated-demo/advanced/protocol.c:先检查索引再访问`<br>`advanced_test.c:容量边界` | `covered` |
| HW-C99-06-R-05 | 规则 6.5 所有的if ... else if结构应该由else子句结束 ；switch语句必须有default分支。 | `test` | `lint:control.switch-default`<br>`lint:comment.case-intent` | `covered` |
| HW-C99-06-S-01 | 建议 6.1 函数中分配的内存，在函数退出之前要释放。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |
| HW-C99-06-S-02 | 建议 6.2 if语句尽量加上else分支，对没有else分支的语句要小心对待。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |
| HW-C99-06-S-03 | 建议 6.3 不要滥用goto语句。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |
| HW-C99-06-S-04 | 建议 6.4 时刻注意表达式是否会上溢、下溢。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`lint:control.*,embedded.*`<br>`generated-demo/advanced/fixed_pool.c` | `covered` |

## 7 程序效率

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-07-P-01 | 原则 7.1 在保证软件系统的正确性、简洁、可维护性、可靠性及可测性的前提下，提高代码效率。 | `demo` | `generated-demo/advanced/fixed_pool.c`<br>`generated-demo/advanced/protocol.c`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-07-P-02 | 原则 7.2 通过对数据结构、程序算法的优化来提高效率。 | `demo` | `generated-demo/advanced/fixed_pool.c`<br>`generated-demo/advanced/protocol.c`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-07-S-01 | 建议 7.1 将不变条件的计算移到循环体外。 | `demo` | `generated-demo/advanced/fixed_pool.c`<br>`generated-demo/advanced/protocol.c`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-07-S-02 | 建议 7.2 对于多维大数组，避免来回跳跃式访问数组成员。 | `demo` | `generated-demo/advanced/fixed_pool.c`<br>`generated-demo/advanced/protocol.c`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-07-S-03 | 建议 7.3 创建资源库，以减少分配对象的开销。 | `demo` | `generated-demo/advanced/fixed_pool.c`<br>`generated-demo/advanced/protocol.c`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-07-S-04 | 建议 7.4 将多次被调用的 “小函数”改为inline函数或者宏实现。 | `demo` | `generated-demo/advanced/fixed_pool.c`<br>`generated-demo/advanced/protocol.c`<br>`docs/spec/TRACEABILITY.md` | `covered` |

## 8 注释

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-08-P-01 | 原则 8.1 优秀的代码可以自我解释，不通过注释即可轻易读懂。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-P-02 | 原则 8.2 注释的内容要清楚、明了，含义准确，防止注释二义性。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-P-03 | 原则 8.3 在代码的功能、意图层次上进行注释，即注释解释代码难以直接表达的意图，而不是重复描述代码。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-R-01 | 规则 8.1 修改代码时，维护代码周边的所有注释，以保证注释与代码的一致性。不再有用的注释要删除。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-R-02 | 规则 8.2 文件头部应进行注释，注释必须列出：版权说明、版本号、生成日期、作者姓名、工号、内容、功能说明、与其它文件的关系、修改日志等，头文件的注释中还应有函数功能简要说明。 | `manual` | `lint:documentation.file-metadata,employee-id.*,modification-history.*`<br>`examples/c-kit.json:documentation 元数据覆盖策略`<br>`docs/COMMENTING_METHOD.md:文件头与版本历史` | `covered` |
| HW-C99-08-R-03 | 规则 8.3 函数声明处注释描述函数功能、性能及用法，包括输入和输出参数、函数返回值、可重入的要求等；定义处详细描述函数功能和实现要点，如实现的简要步骤、实现的理由、设计约束等。 | `manual` | `lint:documentation.performance,reentrancy,definition-details,function-flow`<br>`lint:documentation.private-prototype`<br>`generated-demo/advanced/state_machine.c:DEMO_RunCycle 编号流程` | `covered` |
| HW-C99-08-R-04 | 规则 8.4 全局变量要有较详细的注释，包括对其功能、取值范围以及存取时注意事项等的说明。 | `manual` | `lint:documentation.global-variable`<br>`generated-demo/advanced/protocol.c:s_protocol_version 取值范围和只读访问说明` | `covered` |
| HW-C99-08-R-05 | 规则 8.5 注释应放在其代码上方相邻位置或右方，不可放在下面。如放于上方则需与其上面的代码用空行隔开，且与下方代码缩进相同。 | `manual` | `lint:comment.numbered-intent-placement`<br>`readability.manualReview:review.comment.logical-blocks`<br>`docs/COMMENTING_METHOD.md:普通逻辑段语义由黄金样例和人工评审确认` | `covered` |
| HW-C99-08-R-06 | 规则 8.6 对于switch语句下的case语句，如果因为特殊情况需要处理完一个case后进入下一个case处理，必须在该case语句处理完、下一个case语句前加上明确的注释。 | `manual` | `lint:comment.case-fallthrough,comment.case-intent`<br>`generated-demo/demo.c:等级名称 switch`<br>`generated-demo/advanced/state_machine.c:状态 switch` | `covered` |
| HW-C99-08-R-07 | 规则 8.7 避免在注释中使用缩写，除非是业界通用或子系统内标准化的缩写。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-R-08 | 规则 8.8 同一产品或项目组统一注释风格。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-S-01 | 建议 8.1 避免在一行代码或表达式的中间插入注释。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-S-02 | 建议 8.2 注释应考虑程序易读及外观排版的因素，使用的语言若是中、英兼有的，建议多使用中文，除非能用非常流利准确的英文表达。对于有外籍员工的，由产品确定注释语言。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |
| HW-C99-08-S-03 | 建议 8.3 文件头、函数头、全局常量变量、类型定义的注释格式采用工具可识别的格式。 | `manual` | `docs/COMMENTING_METHOD.md`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c`<br>`lint:documentation.*,comment.*` | `covered` |

## 9 排版与格式

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-09-R-01 | 规则 9.1 程序块采用缩进风格编写，每级缩进为4个空格。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |
| HW-C99-09-R-02 | 规则 9.2 相对独立的程序块之间、变量说明之后必须加空行。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |
| HW-C99-09-R-03 | 规则 9.3 一条语句不能过长，如不能拆分需要分行写。一行到底多少字符换行比较合适，产品可以自行确定。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |
| HW-C99-09-R-04 | 规则 9.4 多个短语句（包括赋值语句）不允许写在同一行内，即一行只写一条语句。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |
| HW-C99-09-R-05 | 规则 9.5 if、for、do、while、case、switch、default等语句独占一行。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |
| HW-C99-09-R-06 | 规则 9.6 在两个以上的关键字、变量、常量进行对等操作时，它们之间的操作符之前、之后或者前后要加空格；进行非对等操作时，如果是关系密切的立即操作符（如－>），后不应加空格。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |
| HW-C99-09-S-01 | 建议 9.1 注释符（包括„/*‟„//‟„*/‟）与注释内容之间要用一个空格进行分隔。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |
| HW-C99-09-S-02 | 建议 9.2 源程序中关系较为紧密的代码应尽可能相邻。 | `lint` | `lint:format.*,control.compound-braces`<br>`.clang-format`<br>`generated-demo/` | `covered` |

## 10 表达式

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-10-R-01 | 规则 10.1 表达式的值在标准所允许的任何运算次序下都应该是相同的。 | `compile` | `compiler:-Wall,-Wextra,-Wconversion`<br>`lint:boolean.*,control.*`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c` | `covered` |
| HW-C99-10-S-01 | 建议 10.1 函数调用不要作为另一个函数的参数使用，否则对于代码的调试、阅读都不利。 | `compile` | `compiler:-Wall,-Wextra,-Wconversion`<br>`lint:boolean.*,control.*`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c` | `covered` |
| HW-C99-10-S-02 | 建议 10.2 赋值语句不要写在if等语句中，或者作为函数的参数使用。 | `compile` | `compiler:-Wall,-Wextra,-Wconversion`<br>`lint:boolean.*,control.*`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c` | `covered` |
| HW-C99-10-S-03 | 建议 10.3 用括号明确表达式的操作顺序，避免过分依赖默认优先级。 | `compile` | `compiler:-Wall,-Wextra,-Wconversion`<br>`lint:boolean.*,control.*`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c` | `covered` |
| HW-C99-10-S-04 | 建议 10.4 赋值操作符不能使用在产生布尔值的表达式上。 | `compile` | `compiler:-Wall,-Wextra,-Wconversion`<br>`lint:boolean.*,control.*`<br>`generated-demo/demo.c`<br>`generated-demo/advanced/*.c` | `covered` |

## 11 代码编辑、编译

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-11-R-01 | 规则 11.1 使用编译器的最高告警级别，理解所有的告警，通过修改代码而不是降低告警级别来消除所有告警。 | `compile` | `examples/c-kit.json:gates.gcc/clang.flags`<br>`scripts/verify.ps1` | `covered` |
| HW-C99-11-R-02 | 规则 11.2 在产品软件（项目组）中，要统一编译开关、静态检查选项以及相应告警清除策略。 | `compile` | `examples/c-kit.json:gates`<br>`config/skills/c99-standard-c/c-kit.schema.json` | `covered` |
| HW-C99-11-R-03 | 规则 11.3 本地构建工具（如PC-Lint）的配置应该和持续集成的一致。 | `compile` | `scripts/verify.ps1`<br>`scripts/verify.sh`<br>`examples/c-kit.json:gates` | `covered` |
| HW-C99-11-R-04 | 规则 11.4 使用版本控制（配置管理）系统，及时签入通过本地构建的代码，确保签入的代码不会影响构建成功。 | `compile` | `scripts/verify.ps1`<br>`scripts/verify.sh`<br>`examples/c-kit.json:gates` | `covered` |
| HW-C99-11-S-01 | 建议 11.1 要小心地使用编辑器提供的块拷贝功能编程。 | `compile` | `scripts/verify.ps1`<br>`scripts/verify.sh`<br>`examples/c-kit.json:gates` | `covered` |

## 12 可测性

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-12-P-01 | 原则 12.1 模块划分清晰，接口明确，耦合性小，有明确输入和输出，否则单元测试实施困难。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`generated-demo/advanced/state_machine.c:assert`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-12-R-01 | 规则 12.1 在同一项目组或产品组内，要有一套统一的为集成测试与系统联调准备的调测开关及相应打印函数，并且要有详细的说明。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`generated-demo/advanced/state_machine.c:assert`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-12-R-02 | 规则 12.2 在同一项目组或产品组内，调测打印的日志要有统一的规定。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`generated-demo/advanced/state_machine.c:assert`<br>`docs/spec/TRACEABILITY.md` | `covered` |
| HW-C99-12-R-03 | 规则 12.3 使用断言记录内部假设。 | `test` | `generated-demo/advanced/state_machine.c:内部私有函数 assert` | `covered` |
| HW-C99-12-R-04 | 规则 12.4 不能用断言来检查运行时错误。 | `test` | `generated-demo/:公开入口运行时错误返回`<br>`advanced_test.c:错误注入` | `covered` |
| HW-C99-12-S-01 | 建议 12.1 为单元测试和系统故障注入测试准备好方法和通道。 | `test` | `generated-demo/advanced/tests/advanced_test.c:DEMO_TestFaultInjection` | `covered` |

## 13 安全性

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-13-P-01 | 原则 13.1 对用户输入进行检查。 | `test` | `generated-demo/advanced/protocol.c`<br>`generated-demo/advanced/tests/advanced_test.c`<br>`lint:embedded.forbidden-call` | `covered` |
| HW-C99-13-R-01 | 规则 13.1 确保所有字符串是以NULL结束。 | `test` | `generated-demo/advanced/protocol.c:有界查找并显式写入空字符` | `covered` |
| HW-C99-13-R-02 | 规则 13.2 不要将边界不明确的字符串写到固定长度的数组中。 | `test` | `generated-demo/advanced/protocol.c:目标容量包含结尾空间检查` | `covered` |
| HW-C99-13-R-03 | 规则 13.3 避免整数溢出。 | `test` | `generated-demo/demo.c:uint32_t 累加上界`<br>`generated-demo/advanced/fixed_pool.c:显式代际边界` | `covered` |
| HW-C99-13-R-04 | 规则 13.4 避免符号错误。 | `test` | `compiler:-Wsign-conversion`<br>`generated-demo/advanced/protocol.c:无符号解码` | `covered` |
| HW-C99-13-R-05 | 规则 13.5 避免截断错误。 | `test` | `compiler:-Wconversion`<br>`generated-demo/:校验后显式窄化` | `covered` |
| HW-C99-13-R-06 | 规则 13.6 确保格式字符和参数匹配。 | `test` | `compiler:-Wformat=2`<br>`DEMO_FormatStatus:PRIu32` | `covered` |
| HW-C99-13-R-07 | 规则 13.7 避免将用户输入作为格式化字符串的一部分或者全部。 | `test` | `DEMO_FormatStatus:编译期固定格式串`<br>`lint:embedded.forbidden-call` | `covered` |
| HW-C99-13-R-08 | 规则 13.8 避免使用strlen()计算二进制数据的长度。 | `test` | `advanced/DEMO_DecodeFrame/DEMO_PoolRead:显式 size_t 二进制长度` | `covered` |
| HW-C99-13-R-09 | 规则 13.9 使用int类型变量来接受字符I/O函数的返回值。 | `test` | `AGENTS.md:字符 I/O 返回值必须使用 int`<br>`manual-review` | `covered` |
| HW-C99-13-R-10 | 规则 13.10 防止命令注入。 | `test` | `lint:embedded.forbidden-call(system,popen)`<br>`黄金示例不执行命令` | `covered` |

## 14 单元测试

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-14-R-01 | 规则 14.1 在编写代码的同时，或者编写代码前，编写单元测试用例验证软件设计/编码的正确。 | `test` | `generated-demo/advanced/tests/advanced_test.c`<br>`go test ./...` | `covered` |
| HW-C99-14-S-01 | 建议 14.1 单元测试关注单元的行为而不是实现，避免针对函数的测试。 | `test` | `advanced_test.c:只调用公开接口` | `covered` |

## 15 可移植性

| ID | 条款 | 主证据 | 证据定位 | 状态 |
| --- | --- | --- | --- | --- |
| HW-C99-15-R-01 | 规则 15.1 不能定义、重定义或取消定义标准库/平台中保留的标识符、宏和函数。 | `compile` | `lint:naming.reserved`<br>`compiler:-Wpedantic` | `covered` |
| HW-C99-15-S-01 | 建议 15.1 不使用与硬件或操作系统关系很大的语句，而使用建议的标准语句，以提高软件的可移植性和可重用性。 | `compile` | `scripts/verify.ps1:header-c99-cxx17`<br>`lint:naming.reserved`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h` | `covered` |
| HW-C99-15-S-02 | 建议 15.2 除非为了满足特殊需求，避免使用嵌入式汇编。 | `compile` | `scripts/verify.ps1:header-c99-cxx17`<br>`lint:naming.reserved`<br>`generated-demo/demo.h`<br>`generated-demo/advanced/*.h` | `covered` |

## 完整性门禁

`tools/rules/build_rule_catalog.py --check` 会核对：

- Markdown 中恰好存在 139 条原则、规则和建议；
- 每个条款 ID 唯一且章节范围为 1—15；
- 每条都有主证据、至少一种验证方法、至少一个证据定位和 `covered` 状态；
- JSON 与本 Markdown 都和当前参考 Markdown 的机械生成结果完全一致。
