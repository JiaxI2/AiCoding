# 已选方案：Option D 简单入口与公开高级样例

Decision Status: Selected

## 选择

采用一对入门文件、3 个公开高级职责模块和 1 个行为测试入口：

- `demo.c/.h`：最多 8 个采样的平均值和等级判定，作为第一阅读入口；
- `advanced/state_machine.c/.h`：状态机、ISR/DMA、并发、数值和边界安全；
- `advanced/protocol.c/.h`：字节序、不可信输入、字符串和二进制边界、安全格式化；
- `advanced/fixed_pool.c/.h`：固定资源池、资源复用、生命周期和释放后防护；
- `advanced/tests/advanced_test.c`：行为、边界与故障注入测试。

## 规则表达方式

PDF 每条原则、规则和建议都必须在规则目录中映射为 `demo`、`lint`、`compile`、`test` 或 `manual`。禁止项不通过危险代码演示，而通过安全替代实现和负向 lint fixture 证明。

用户自定义与规则配置分离：`c-kit.json` 管门禁，VS Code 兼容 `c-snippets.json` 管文件头、函数头、
结构体、枚举、分隔符和常用 include。`init` 安装两者，`snippet` 命令列举或渲染常用占位符。

## 明确边界

- 完整 PDF 和经过校验的 Markdown 参考文档保留在 `references/`；
- 根 `AGENTS.md` 提供 Agent 可执行规则；
- 独立 Kit 构建阶段保持 `c-userstyle-kit` 自包含；
- 当前发布快照已通过 `CodingKit/tools/c-userstyle-kit` 接入 AiCoding，仍保留独立 Go module 与验证边界。
