# Changelog

## 1.2.0 - 2026-07-15

### Added

- 第 8 章导向的注释方法：复杂函数编号总览、逻辑段、分支、`case` 和贯穿注释规则。
- lint 可读性摘要，包括函数有效行、分支、嵌套、调用扇出和静态 helper 人工评审项。
- `cstylekit verify` 的 fast/full 外部候选文件门禁、配置 overlay 和 target manifest schema。
- 简单对象宏的 VS Code 片段，总片段数为 9。

### Changed

- 以自包含 Go module 形式集成到 AiCoding `CodingKit/tools`，并由既有 C99 Skill 路由提供快速验证。
- PDF、规范化 Markdown 与 raw 转换件按用户授权作为正式参考资产随 AiCoding 发布。
- 未提供工号时不再生成 `@employee_id`；源码修改历史默认禁用，由 Git 和本文件记录。
- 静态函数前置声明保持裸原型，完整函数注释放在定义处。
- 黄金 Demo 增加编号控制流和领域意图注释，高级规则覆盖样例继续对最终用户可见。

### Fixed

- 函数解析器现在识别 `TYPE *Function(...)`、`TYPE **Function(...)` 等指针返回写法，
  同时避免把 `return *Function()` 和 `sizeof *Function()` 表达式误判为函数声明。

### Verified

- 139/139 条华为规范规则均有证据映射，未分类为 0。
- Kit 发布门禁覆盖 Go、JSON、lint、GCC、Clang、独立 C99/C++17 头文件与行为测试。
- 外部 fast/full 验证只使用主机工具链和临时 host harness，不调用 TI/CCS 或固件构建。
- PDO 验收样例识别 19 个函数（15 个 static、4 个公开函数）；full 11 步门禁和
  baseline/candidate stdout 等价比较通过。
