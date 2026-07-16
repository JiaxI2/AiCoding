# 产品闭环收敛方案选项（PRD Options）：product-convergence

Plan Status: Selected

## 目标

在不新增顶层产品能力、不引入第二套 CLI/测试/报告框架的前提下，将 AiCoding 收敛为：

```text
安装与环境准备
-> 统一 CLI 控制面
-> 统一测试引擎
-> 统一报告与退出码
-> 统一文档入口
-> 统一 Release 闭环
```

## 当前事实

当前仓库已经以 Go CLI 为默认控制面，但仍存在以下并行路径：

- 测试入口同时存在 `smoke`、`ci --profile`、`full`、`test full|release` 和 `release gate`。
- `test full|release` 会通过 `go run ./tools/aicoding-global-tester` 启动第二个测试进程；该 tester 又调用 `full`、`release gate` 和 `fresh-clone`。
- `tools/aicoding-global-tester`、`internal/runner`、`internal/report` 和 `internal/cli/test.go` 分别维护测试/报告模型。
- kit 生命周期同时暴露 `lifecycle` 与 `kit lifecycle`；MCP 生命周期和 runtime Skill profile 又位于独立入口。
- CLI 大量忽略 `flag.Parse` 错误，全局 `--help` 不受支持，参数错误通常退出 `1`，与文档声明的退出码 `2` 不一致。
- README 三件套、`docs/COMMANDS.md`、架构文档、测试文档和静态测试共同固化多组旧入口。
- DocSync policy 没有把 `cmd/**/*.go`、`internal/**/*.go` 和 `Taskfile.yml` 完整纳入命令契约漂移治理。

## 方案 A：兼容优先的统一控制面（推荐）

### 正式产品入口

```text
aicoding bootstrap
aicoding lifecycle ...
aicoding doctor --all
aicoding verify --profile Smoke|Full|Release
aicoding test --profile Smoke|Full|Release
aicoding release verify|gate
```

### 收敛方法

- 将现有 global tester 内聚为 `internal/testengine`，只保留一个 Registry、Profile、Runner、Report 和 ExitCode 实现。
- `test --profile` 成为唯一正式测试入口。
- `release gate` 调用同一 test engine 的 Release plan，不通过 CLI 递归调用。
- `smoke`、`ci`、`full`、旧 `test full|release` 等保留一个版本，统一返回 `CLI_DEPRECATED` 并路由到正式入口。
- `lifecycle` 成为产品生命周期命名空间；kit/MCP/runtime Skill 使用静态 adapter 组合，不引入插件系统。
- `kit lifecycle` 和 MCP 的生命周期动词保留为领域兼容入口一个版本。
- 保持 `report.Result` schemaVersion 1 的兼容字段，补充统一错误分类、deprecation warning 和严格 JSON stdout 契约。

### 优点

- 满足“一入口、一测试体系、一报告体系、一生命周期、一发布闭环”。
- 不要求一次性重写 PowerShell/Python。
- 现有 CI 可通过兼容入口继续运行，并可逐步迁移。
- 实现可拆成小提交，回滚边界清晰。

### 风险

- 生命周期 adapter 边界需要严格限制默认写操作。
- 新旧命令并存一个版本时，帮助和文档必须明确正式/兼容身份。
- test engine 内聚时必须保持现有报告字段和 Required gate 覆盖率。

## 方案 B：只收敛内部实现，保留所有现有命令

### 收敛方法

- 合并 global tester、Runner 和 Report。
- 消除 Full/Release 递归调用。
- 保留 `smoke`、`ci`、`full`、`test full|release`、`release gate` 等全部现有入口，不标记废弃。
- 生命周期只合并内部实现，不调整命令归属。

### 优点

- CLI 迁移风险最低。
- CI 和现有脚本改动较少。

### 缺点

- 用户仍然面对多入口，不能完整实现产品闭环目标。
- 文档仍需解释多组等价命令。
- 后续仍需第二轮 CLI 收敛。

## 方案 C：扁平顶层动词

### 正式产品入口

```text
aicoding install
aicoding update
aicoding doctor
aicoding verify
aicoding test
aicoding release
```

### 收敛方法

- 将 `lifecycle install|update` 提升为顶层动词。
- kit、MCP 和 Skill 生命周期全部隐藏为内部 adapter。
- 现有 `lifecycle`、`mcp install|update`、`kit lifecycle` 全部进入兼容层。

### 优点

- 对最终用户最直观。
- 顶层产品语言最短。

### 风险

- 迁移面最大，容易让 install/update 的作用域不透明。
- 顶层写操作需要新的 scope/confirmation 契约。
- 对已有自动化和文档的冲击高于本次“收敛而非重构产品”的目标。

## 选择结果

选择方案 A。方案 B 不满足“产品入口唯一”，方案 C 会扩张新的扁平顶层写操作命令面；方案 A 是同时满足兼容一个版本、保持现有 CI 和不新增顶层产品能力的路线。
