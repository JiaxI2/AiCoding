# 高级规则覆盖样例

这里的文件是最终用户可见的完整规范覆盖样例，不是多个相互竞争的入门 demo。

- `state_machine.c/.h`：状态迁移、中断协作、临界区、数组和整数边界；
- `protocol.c/.h`：网络字节序、不可信输入、有界字符串和安全格式化；
- `fixed_pool.c/.h`：固定资源池、资源复用、代际句柄和释放后访问防护；
- `tests/advanced_test.c`：只通过公开头文件验证正常、边界和故障注入行为。

初次阅读请先看上一级的 `demo.c/.h`。完整编译、lint 和行为测试统一由
`scripts/verify.ps1` 或 `scripts/verify.sh` 执行。

## 注释层级阅读入口

- `state_machine.c` 的 `DEMO_RunCycle`：编号控制流总览、状态 `case` 的领域意义、运行分支的
  一致快照和结果发布；
- `protocol.c` 的 `DEMO_DecodeFrame`：逐层输入校验、每个失败分支的业务原因，以及验证完成后
  一次发布输出；
- 上一级 `demo.c`：简单函数保持克制，只在原子发布、边界或失败保持语义不显然时补充注释。
- 简单对象宏的正例位于 Kit 根目录 `examples/c-snippets.json` 的 `C Simple Object Macro`；该片段
  对最终用户可见并可直接定制，不为展示规则向业务 demo 添加无用途宏。

完整规则、例外和正反例见 `../../docs/COMMENTING_METHOD.md`。注释数量不是目标；如果大量注释仍无法
说明主流程，应先按职责重构代码。
