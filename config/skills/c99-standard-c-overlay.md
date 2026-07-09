# C99 Standard C Overlay

本 overlay 约束 AiCoding 仓库内 C99 嵌入式 C/H 文件的格式来源和 Agent 执行方式。

## 格式来源

- C skill 不直接写死文件头、函数头、结构体、枚举或分块注释模板。
- C/H 注释模板统一来自 `config/cstyle/comment-templates.json`。
- 机械格式统一来自仓库根目录 `.clang-format`。
- 保持 ISO C99、嵌入式工程风格和 Doxygen 注释。

## Agent 执行规则

- Agent 修改 `.c` 或 `.h` 文件后运行 `task fmt:c`，除非当前任务明确禁止格式化或只允许检查。
- 提交前优先运行 `task fmt-check-staged:c`，只检查 staged C/H 文件。
- 不格式化 `vendor/`、`third_party/`、`generated/`、`Drivers/`、`device/` 路径。
- 不为了迁移历史代码而格式化无关 C/H 文件。

## 实时路径约束

ISR、current-loop、PWM 更新、采样同步等实时路径禁止引入：

- 阻塞调用；
- 动态内存分配；
- 复杂函数指针链；
- 无界循环或不可预测耗时路径。

实时路径中的新增抽象必须能证明调用开销、栈使用和最坏执行时间可控。