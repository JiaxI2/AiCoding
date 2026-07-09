# C99 Standard C Embedded Rules

本规则文件是 AiCoding 仓库内 `c99-standard-c` skill 的本地 overlay。C skill 不直接写死格式模板；可执行的格式、模板和检查入口由本目录配置提供。

## 配置来源

- 机械格式来源：`config/skills/c99-standard-c/style/clang-format.yaml`。
- 根目录 `.clang-format` 仅作为现有工具兼容投影，不是 source of truth。
- C/H 注释模板来源：`config/skills/c99-standard-c/templates/comment-templates.json`。
- Go CLI 通过 `internal/cstyle` 读取 `config/skills/c99-standard-c/skill.json` 后执行格式化、检查和模板校验。

## Agent 执行规则

- 修改 `.c` 或 `.h` 文件后运行 `task fmt:c`，除非当前任务明确禁止格式化或只允许检查。
- 提交前优先运行 `task fmt-check-staged:c`，只检查 staged C/H 文件。
- 不格式化 `vendor/`、`third_party/`、`generated/`、`Drivers/`、`device/`、`build/`、`out/`、`dist/` 路径。
- 不为了迁移历史代码而格式化无关 C/H 文件。
- 保持 C99、嵌入式风格和 Doxygen 注释。

## 实时路径约束

ISR、current-loop、PWM 更新、采样同步等实时路径禁止引入：

- 阻塞调用；
- 动态内存分配；
- 复杂函数指针链；
- 无界循环或不可预测耗时路径。

实时路径中的新增抽象必须能证明调用开销、栈使用和最坏执行时间可控。
