# Apply AiCoding Agent Dev Kit Plan Mode Overlay

## 语言要求

本次任务默认中文优先。你向用户展示的执行计划、权限请求摘要、命令目的说明、验证结果、风险提示和 rollback 说明，都必须使用中文。英文术语可以保留，但应写成中文 + 英文括号，例如：计划模式（Plan Mode）、规格驱动开发（Spec-Driven Development / SDD）、注册表（registry）、总 hook bridge、子模块（hook module）。

当你请求执行命令时，不要生成英文权限摘要。应该写：读取 Plan Mode registry，用于验证前检查。

## 任务边界

- 不新建分支。
- 不修改 `CodingKit/agents/skills` submodule。
- 不修改 Codex plugin cache。
- 不替换 `bin/aicoding.exe`。
- 不改变一个总 hook bridge + 多个 hook 子模块的设计。

## 执行顺序

1. 读取现有 registry、docs、scripts 和 Skill 片段。
2. 只做最小中文优先修正。
3. 运行 Plan Mode、hook、DocSync 和 Git diff 验证。
4. 用中文总结已修改内容、验证结果和 rollback 方法。
