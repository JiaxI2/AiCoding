# ADR 0011: pathpolicy 解析收敛与 policy schema 闭合

PrimitiveReview: required

## Status

Accepted。实现于 `internal/pathpolicy`，由 Plan Mode、测试影响面与 Validation Evidence
共同消费；六份 policy 配置仍保留各自语义和文件所有权。

## 1. Decision

新增只依赖 Go 标准库的 `internal/pathpolicy` Primitive，冻结公开操作为：

- `Compile(patterns)`：规范化、去重、稳定排序并编译既有 `*` / `**` / `?` 方言；
- `Match(compiled, path)`：对已编译 pattern 匹配一个经 fail-closed 校验的仓库相对路径；
- `Validate(patterns)`：不保留编译结果的校验入口。

`internal/plan` 不再拥有 glob 编译器；`internal/testengine` 拥有 change impact 的加载与裁决，
并调用 `pathpolicy`；`internal/validationevidence` 将 exact ref 编译为 exact pattern，将 prefix
编译为追加 `**` 的 pattern。三者不再自行定义 glob 方言。

只收敛解析 Primitive，不合并 `plan-policy.json`、`impact-policy.json` 与
`validation-policy.json`。它们分别拥有敏感路径、测试影响面与 push context 语义，配置字段
与裁决结果保持不变。

## 2. Schema closure

DocSync 对六份配置建立显式的一对一绑定：plan、impact、validation、tagging、docs-sync policy
与 docs-sync semantic。五份新 schema 加既有 plan schema 构成 6/6；`docsync` 与
`governance dependencies` 都执行同一份 checked-in schema 校验，未知字段 fail-closed。
schema 校验属于 DocSync 领域实现，不进入 `pathpolicy`，避免把路径匹配 Primitive 扩成通用
配置框架。

## 3. Compatibility proof

在重构前固定同一 Git index 输入，分别保存 `plan check --staged --json` 与
`change verify --staged --json`；重构后在同一 detached worktree 上调用新二进制，移除
`elapsed*` / `duration*` 字段后做字节比较。只有两份 JSON 均完全相等才允许完成本项。

## 4. Dependency boundary

`pathpolicy` 只 import stdlib。`dependency-governance.json` 禁止它 import 任一 `internal/*`
包，并禁止 gitx、registry、runner、report 反向依赖它；领域层只能单向消费该 Primitive。

## 5. Rollback

恢复三个消费方原实现，删除 `internal/pathpolicy`、能力登记、五份新 schema 与 DocSync
schema closure 检查即可。三份业务 policy 从未合并或迁移，回滚不需要转换配置数据。

## §12 Checklist 自评

**架构**

- 单一职责：只处理冻结的路径 pattern 编译、校验与匹配；不读取文件、不做领域裁决。
- 可继续拆分：三个公开函数共享一个私有规范化器与 glob-to-regexp 编译器，拆分会重新制造方言。
- 可复用：Plan、testengine、validationevidence 三个独立领域直接消费。
- 无重复实现：旧 `normalizePattern` / `globRegex` / `MatchPattern` 被删除，全仓编译逻辑只剩一处。
- 新 Primitive 必要性：此前三个 policy 的同构解析分散在领域代码中，已形成重复实现。

**性能**

- Fast Path：pattern 在一次裁决前只编译一次，随后复用预编译 regexp；无子进程、网络或文件 IO。
- 无关扫描：输入只有调用方给出的 pattern/path；不扫描仓库。
- 重复 IO / 计算：Primitive 零 IO；Plan 与 impact 均在路径循环外编译。
- 最小输入输出：输入为字符串切片或单个 path，输出为 compiled value、bool 或 error。
- 实测：`BenchmarkCompileAndMatch` 独立覆盖编译与匹配成本，正式证据随 Full 报告保留。

**质量**

- 确定性：规范化后去重并按字节序排序，相同输入产生相同 compiled 顺序。
- 接口稳定：公开函数只有 `Compile`、`Match`、`Validate`，无配置或领域类型泄漏。
- 独立测试：覆盖目录边界、`*`/`**`/`?`、去重排序、绝对路径与 traversal 拒绝。
- 自由组合：调用方保留 reason/profile/context 等业务状态，只组合 pattern 结果。
