# Extension Adapter Contract

Status: Accepted and Frozen

## 1. 契约目的

Extension Adapter Contract 定义 AiCoding 如何把统一 lifecycle action 翻译到 Kit、MCP 和
runtime Skill 领域，同时避免 God Core、动态 plugin ABI 和跨领域状态耦合。

Adapter 是翻译层，不是业务引擎：

```text
Lifecycle Request
  -> Adapter Descriptor + static function
     -> typed domain call
        -> domain-owned policy/state/result
```

组件级扩展通常复用现有领域 adapter。新增一个 Kit、MCP component 或 external Skill
不等于新增 adapter；只有新增领域种类才需要新的 adapter descriptor。

## 2. 已实现的 descriptor

权威类型位于 `internal/lifecycle`：

```go
type AdapterDescriptor struct {
    ID         string
    InputKind  string
    StateOwner string
    Entrypoint string
    Actions    []AdapterAction
}

type AdapterAction struct {
    Name   string
    Effect string
}
```

字段语义：

| 字段 | 语义 | 不允许 |
|---|---|---|
| `ID` | 稳定领域 identity | 版本、实现路径、用户 profile |
| `InputKind` | adapter 接受的事实快照种类 | 绝对路径、mutable global state |
| `StateOwner` | write/rollback 的唯一领域所有者 | `core`、`global`、其他领域 |
| `Entrypoint` | `go-static` 或 `bounded-process` | dynamic Go plugin、任意 in-process code |
| `Actions` | 支持的 action 与 `read`/`write` effect | 隐式副作用、未登记 action |

Catalog 是静态编译的 descriptor + function 表。Snapshot digest 只覆盖 descriptor facts；
Go function address 不进入 identity。Descriptor 返回只读副本，重复 ID/action、缺失字段或
非法 effect 在 catalog 构建时失败。

## 3. 当前 adapter

| ID | InputKind | StateOwner | Entrypoint | Read | Write |
|---|---|---|---|---|---|
| `kit` | `kit-catalog` | `kit` | `go-static` | status, doctor, verify | install, update, uninstall, rollback |
| `mcp` | `mcp-catalog` | `mcp` | `go-static` | status, doctor, verify | install, update, uninstall |
| `runtime-skill` | `runtime-skill-registry` | `runtime-skill` | `bounded-process` | status, doctor, verify | install, update, uninstall |
| `repo-context` | `repo-context-facts` | `repo-context` | `go-static` | status, doctor, verify | install, update, uninstall |

`repo-context` 经 [ADR 0003](../decisions/0003-repo-context-domain.md) 按新领域 adapter 准入
（§10），进入时六模块零修改，是第 §10 步"可删除性证明"的又一先例。它暂不提供 `rollback`：
update 即重新收敛到当前事实，真实回滚需求出现后按"只增不改"追加。

Adapter catalog 的顺序是 lifecycle 的稳定执行顺序。`--scope all` 选择三个 descriptor；
单 scope 只选择一个。选择结果被转换成 `ExecutionPlan` tasks，runner 不了解各领域含义。

## 4. 输入契约

### 4.1 Kit

`kit-catalog` 是以下内容树：

```text
normalized kit registry digest
+ sorted (kit id, manifest path, manifest digest)
```

一次命令解析 registry 和每个 manifest 一次。选择、plan、apply、doctor、verify 与 view 使用
snapshot 中的 detached values；执行过程中不重新读取 manifest。

### 4.2 MCP

`mcp-catalog` 是以下内容树：

```text
normalized MCP registry digest
+ sorted (component id, manifest path, component digest)
```

Registry-only digest 保留用于诊断 registry 本身；catalog digest 标识 registry 与全部引用
manifest 的组合事实。Manifest-only 变化必须改变 catalog digest，但不改变 registry digest。

### 4.3 Runtime Skill

`runtime-skill-registry` 包含规范化 `config/codex-kit.json` 与可用的 Codex-Skills source
commit。绝对 source path 不进入 digest，因此相同 source facts 在不同机器保持同一 identity。

Runtime 扫描结果、plugin cache 和 junction 是 mutable observation，不伪装成输入对象；它们
由 status/doctor/verify 结果报告。

### 4.4 Repo Context

`repo-context-facts` 是规范化的仓库事实快照：仓库名、按扩展名统计的语言构成、检测到的
工具链标记、顶层域（目录 + 文件数 + 主语言）。全部字段排序，相对路径，不含绝对路径或
时间戳，因此同一仓库在不同机器与不同时刻的 digest 稳定。

生成的 scoped context 文件与其 manifest（`.aicoding/repo-context/`）是领域自有 owned state；
每个文件记录内容 digest，`uninstall` 只删除 digest 匹配的登记文件，`doctor` 以此发现被
外部篡改或缺失的生成物，`status` 用 facts digest 与 manifest 记录对账新鲜度。用户手写文件
（不在 manifest 中）永不被触碰。

## 5. Action 契约

每个 adapter action 必须声明 effect：

| Action | 前置输入 | 输出 | 不变量 |
|---|---|---|---|
| install | catalog snapshot + selection | domain result/state evidence | 只创建 owned assets |
| update | 同 identity snapshot + selection | converged result/state evidence | 不创建平行 identity |
| uninstall | snapshot + ownership evidence | removal result | 不删除 unknown/unmanaged assets |
| status | snapshot + runtime observation | status result | 不写状态 |
| doctor | snapshot + environment observation | findings | 不修复状态 |
| verify | snapshot + profile | verification result | 不执行 install/rollback |
| rollback | domain rollback evidence | restored domain state | 不跨领域补偿 |

顶层 `plan` 不是 adapter action。CLI 将 `lifecycle plan --action X` 转为 `X + dryRun=true`，
所以 plan/apply 复用同一领域路径，避免两套规则。

## 6. 输出契约

每个 adapter 返回：

```go
type AdapterResult struct {
    ID          string
    Action      string
    DryRun      bool
    InputDigest string
    OK          bool
    Status      string
    Data        any
    Warnings    []string
    Errors      []string
}
```

生命周期外壳另返回：

- `catalogDigest`：静态 adapter catalog；
- `planDigest`：本次 adapter/action/selection 意图；
- ordered `adapters`：每个领域的输入与结果证据；
- summary/warnings/errors：聚合视图。

Adapter 不改写 domain result。Lifecycle 只聚合，不推断缺失的 rollback 或业务成功。

## 7. State Ownership

State 永远属于领域：

| Domain | Owned state/evidence | 不拥有 |
|---|---|---|
| Kit | manifest-declared install state、plugin sync result、last Kit rollback snapshot | MCP venv/config、unknown files |
| MCP | component venv、Codex managed block、MCP install state、config backup/staged runtime | Skill roots、unmanaged MCP section |
| runtime Skill | registered junction、profile audit、migration backup/rollback manifest | canonical source、plugin cache internals |

Lifecycle 不保存 global state，也没有 `all` scope rollback。若未来需要跨领域一致性，只能先
定义明确的 prepare/expected-old/commit contract 和真实恢复语义；不能用“删除所有改动”代替。

## 8. External Skill 扩展

新增外部 GitHub Skill 的完整定义与实现顺序：

1. Codex-Skills 在 `external/` 增加 declared nested Git submodule，固定 reviewed commit/tag。
2. Codex-Skills 更新 binding manifest，指向含 `SKILL.md` 的精确目录。
3. 完成 source/package validation 并发布可供 AiCoding pin 的 commit。
4. AiCoding 更新 `CodingKit/agents/skills` gitlink。
5. AiCoding 在 `config/codex-kit.json` 的 profile、standalone registry 和 `sourcePaths` 登记
   runtime identity；不复制 source。
6. 运行 runtime Skill lifecycle plan、update、audit/verify；拒绝 duplicate active name。
7. 卸载只删除 target 精确匹配 registry source path 的 managed junction。

调用者：

```powershell
bin\aicoding.exe lifecycle plan --scope runtime-skill --action update --runtime-profile full --json
bin\aicoding.exe lifecycle update --scope runtime-skill --runtime-profile full --json
bin\aicoding.exe lifecycle verify --scope runtime-skill --runtime-profile full --json
```

Agent/Skill 只调用这些命令；不会直接执行 profile PowerShell，也不会编辑 user root。

## 9. MCP Component 扩展

新增 MCP component 的完整定义与实现顺序：

1. 增加 component runtime 与 MCP protocol implementation。
2. 增加 `config/mcp/components/<id>.json`，描述 runtime、Codex registration、doctor、verify。
3. 在 `config/mcp-registry.json` 增加稳定 ID、enabled/order 与 manifest ref。
4. 运行 registry/catalog contract、component doctor/verify、lifecycle dry-run。
5. apply install/update，确认 managed block、runtime 与 install state 一致。
6. uninstall 验证只移除 owned assets，并验证失败恢复。

不需要：

- 修改 lifecycle adapter catalog；
- 新增 CLI top-level command；
- 修改 runner、snapshot 或 report；
- 向 MCP server 增加 AiCoding workflow prompt。

## 10. 新领域 adapter

只有现有领域无法表达且存在真实功能时，才新增 adapter：

1. 定义具体领域模块，不先定义通用接口。
2. 指定 input facts、state owner、actions/effects 和 bounded entrypoint。
3. 增加 descriptor + static function；不使用 init-time hidden registration/global mutable map。
4. 将 request 翻译成 typed domain values，返回 domain result。
5. 增加 catalog contract、domain module tests、lifecycle consumer test 和 CLI JSON contract。
6. 证明删除该 adapter 不需要修改 snapshot/runner/report。

只有两个领域 adapter 出现相同且稳定的翻译变化点，才抽取更小的共用 helper；不能为“将来
可能有”预建 interface/factory hierarchy。

## 11. 测试契约与影响半径

| 修改 | 必须验证 | 何时扩大 |
|---|---|---|
| descriptor 内容/校验 | `internal/lifecycle` catalog tests | action/effect 变化时跑 CLI/Full |
| generic catalog snapshot | `internal/registry` tests | digest contract 变化时跑 Kit/MCP |
| Kit catalog/adapter | `internal/kit` + lifecycle Kit consumer | public JSON/state 变化时跑 CLI/Full |
| MCP catalog/adapter | `internal/mcpcontrol` + lifecycle MCP consumer | config/protocol 变化时跑 integration |
| runtime Skill adapter | lifecycle runtime tests + runtime audit | user root/source policy变化时跑 Full/Release |
| report fields/schema | report + CLI contracts | 任意外部字段语义变化时跑 Full/Release |

Module contract tests 固定输入、输出、不变量和生命周期；consumer regression 证明相邻模块仍
可组合；只有公开跨模块契约变化或交付验收才默认执行 Full/Release。

## 12. 禁止模式

- `Core`/`Manager` 按 domain ID 写不断增长的 switch；
- adapter 保存全部状态、包含业务策略或调用其他 adapter；
- runner 判断 action 业务含义；
- report 根据错误自动修复；
- dynamic Go plugin、reflection-based arbitrary loader、hidden global registration；
- 把 descriptor 当脚本语言，加入任意 shell 或 workflow DSL；
- 用一个 `installed=true` 折叠 source/package/exposure/discovery；
- 新 component/Skill 要求修改 snapshot、runner 或 report；
- 为不存在的 consumer 增加 capability graph、remote API 或 global transaction。

## 13. 冻结规则

本契约已由 Kit、MCP、runtime Skill 三个 adapter、Kit/MCP 两个内容树消费者和 lifecycle
ExecutionPlan/JSON digest 证据落地。至此 adapter architecture 冻结。

后续新增 component/Skill 是功能扩展；领域内部同步、性能或错误处理是模块维护；只有现实
问题证明本契约阻止正确实现，且新的稳定变化点有至少两个消费者，才通过 ADR 修改契约。
