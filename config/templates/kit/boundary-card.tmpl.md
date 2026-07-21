# {{.Name}} 外部包装边界卡

Status: Draft

## 上游地址与 pin 策略

- 上游地址：尚未选择；本 scaffold 保持 disabled，启用前必须替换为权威仓库地址。
- pin 策略：只接受 submodule 的不可变 commit pin 或 `go.mod` 的明确版本。
- 禁止把上游源码复制进本仓后分叉修改。

## 控制面声明或入口

- 当前不把上游运行时接入 AiCoding 控制面；唯一入口是只读结构验证：
  `aicoding lifecycle verify --scope kit --kit {{.ID}} --json`。
- 若未来需要真实外部命令，先在 manifest 登记 `external-command` 并经架构评审。

## 不承担的门禁

- scaffold 不提供上游正确性、安全性、许可证兼容性或发布质量担保。
- 在上游地址、pin 与验证证据确认前，不得让 Smoke、Full 或 Release 依赖其运行结果。

## 同步纪律

上游变更先在上游或 fork 验证，再评审目标版本与 commit；验证通过后前移 submodule 或
`go.mod` pin，本仓只提交引用、边界卡和对应验证证据，不复制上游实现。
