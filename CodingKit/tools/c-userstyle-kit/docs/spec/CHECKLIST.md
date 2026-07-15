# 检查清单：华为 C 规范全覆盖黄金示例

- [x] 架构路线已选择并记录。
- [x] PDF Markdown 转换已检查完整性和可读性。
- [x] 每条 PDF 规则已归类并有证据。
- [x] 根目录只有简单 `demo.c/.h`，高级样例在公开 `advanced/` 目录中。
- [x] snippets JSON 可直接用于 VS Code，并可由 CLI 列举和渲染。
- [x] 公共头文件自包含并具有唯一保护宏。
- [x] 静态前置声明无 Doxygen，定义处注释完整。
- [x] 复杂函数模板具有编号控制流总览，主要逻辑段使用空行和段前意图注释。
- [x] 非显然 `if/else` 和有实际处理的 `case/default` 说明领域意图；连续空 case 正确豁免。
- [x] 黄金模板不保留仅含注释的空 `else`，失败保持语义位于对应判断前。
- [x] 简单对象宏 snippet 使用单行块注释；黄金模板省略未知工号和默认代码内修改历史。
- [x] 高级 README 模板指向编号流程样例和最终用户可见的简单宏 snippet。
- [x] `generated-demo` 统一再生后运行完整 JSON/schema、lint、编译和测试门禁。
- [x] 已按 AiCoding 架构集成，且未修改 skills submodule、生成插件、Marketplace 或插件缓存。
- [x] 交接包含已验证、未验证和回滚。
