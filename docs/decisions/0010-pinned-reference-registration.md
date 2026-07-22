# ADR 0010: Kit 内容钉死引用注册

PrimitiveReview: required

## Status

Accepted。

## 1. Decision

在既有 Kit manifest v2 上增加**可选** `source` 输入，使 registry 可以登记仓库外能力的
不可变内容身份，而不把能力正文 vendoring 进 AiCoding：

```json
{ "kind": "git", "url": "...", "commit": "<40-hex commit>" }
```

或：

```json
{ "kind": "content", "digest": "sha256:<64-hex>" }
```

Git source 只接受完整 commit object ID，branch、tag、缩写 SHA 与其他 revision expression
全部拒绝；content source 只接受小写 SHA-256 digest。`source` 缺省时沿用既有本仓路径语义，
因此旧 manifest 无迁移、无行为变化。

登记与导入保持两阶段：`kit register` 只把仓库内 manifest 登记进既有
`config/kit-registry.json`；`kit prefetch` 在登记阶段解析并获取 pin。`lifecycle
install|update --scope kit` 只从 Git common-dir 下的内容寻址 pin cache 本地物化，绝不发起
网络请求。缓存缺失时返回 `evidence-missing` 与可执行 required action，不把“有 pin”放松为
“内容可以不存在”。

## 2. 内容身份与缓存

pin identity 由 canonical `source` 值计算；Git pin 的缓存项同时保存 URL、请求 commit 与解析后
commit，并要求三者完全匹配。缓存根为：

```text
<git-common-dir>/aicoding/pins/<sha256-hex>/
```

Git prefetch 在临时目录获取指定 commit，验证 `^{commit}` 后原子发布不可变 cache entry；
materialization 从该本地 object database 以 `git archive` 写入 worktree-owned
`.aicoding/state/kits/<id>/source/`。content source 的 digest 本身就是 identity；它只接受已经由
外部可信获取流程放入同结构 cache 的内容，本 ADR 不为无 locator 的 digest 猜测网络来源。

pin cache 是 cache 治理的第六 scope。registry 当前引用的 identity 永不进入 clean 候选；只有
孤儿 pin 可由显式 `cache clean --scope pins` 删除。registry、manifest、pin identity 与
materialized digest 都进入现有 Kit catalog/install state；不新增 registry、Receipt 或验证权威。

## 3. 与 Plugin SDK §6 doctrine 一致

`source` 是 manifest 输入，不是 owned 资产中的用户定制。用户选择的 URL/commit/digest 只在
仓库配置里持有；prefetch cache 与 materialized tree 是该输入解析出的、可重建的不可变制品，
不得被用户手改成第二事实源。`update`/`uninstall` 只替换或删除精确物化目录与 install state，
不会改外部 checkout、manifest、用户配置或用户 Skill。

这正是 §6 的“定制流经输入”阶梯：登记层选择不可变能力，源码层升级通过评审后前移 pin；
不允许在 plugin cache、junction 或内核代码中保存可变个性化内容。runtime-skill 的
`git rev-parse HEAD` 仍服务其既有 submodule/checkout 路径；本 ADR 复用相同的 commit-object
身份纪律，将其推广到 Kit manifest，而不修改 runtime-skill adapter。

## 4. 可靠性边界

- prefetch 可以访问 source locator；install/update/import 的实现路径没有 network primitive。
- 坏 SHA、locator 内容与 pin 不一致、缓存元数据或内容摘要漂移全部 fail-closed。
- structure 校验只在“本地 required path 存在”或“source pin 已解析且 cache 完整”时通过。
- manifest snapshot 已参与 kit-catalog digest；前移 pin 必然改变 validation input identity，旧
  Receipt 不能命中新内容。
- `kit list` 只读取 registry/manifest，不扫描或拉取 cache；外部能力登记后立即可见。

## 5. 明确不做与回滚

不支持纯路径 source、branch/tag、import 隐式 fetch、可变 cache entry、第二 Kit registry、
动态 plugin 或把外部正文提交进仓库。content digest 不携带 locator，本轮不发明内容分发协议。

回滚时删除 manifest `source` 字段支持、register/prefetch 子命令和 pins cache scope；已存在的老
manifest 与 registry 不迁移。Git common-dir 下的孤儿 pin 与 worktree state 可按精确 ID 删除，
外部源仓库和 Validation Evidence Receipt 均保持不变。

## §12 Checklist 自评

**架构**

- 单一职责：source 只表达不可变外部内容身份；registry、lifecycle、cache 与 Receipt 权威不变。
- 可继续拆分：纯 pin 校验、prefetch 与本地 materialization 按 I/O 边界分离在既有 Kit 领域内。
- 可复用：沿用 `gitx`、registry snapshot、Kit lifecycle state 与 cache retention。
- 无重复实现：不新增 Git 调用边界、registry、Receipt、runner 或 lifecycle adapter。
- 新 Primitive 必要性：新增的最小纯函数只负责 canonical source identity 与封闭 shape 校验；
  既有 runtime-skill digest 不解析通用 Kit manifest，也不拥有 Kit cache。

**性能**

- Fast Path：`kit list` 不读取 pin 内容；identity 只哈希小型 manifest 值。
- 无关扫描：prefetch/物化按一个 identity 精确寻址；pins clean 只与 registry 引用集合做比较。
- 重复 I/O/计算：网络只在 prefetch；import 复用本地 object database，不再次解析远端。
- 最小 Context：报告只返回 identity、cache/materialized path、状态与 required action。

**质量**

- 确定性：同一 canonical source 得到同一 identity 与 cache path。
- 接口稳定：schema 只增加可选字段；既有 Kit manifest 与 lifecycle 命令保持兼容。
- 独立测试：覆盖 branch、坏 SHA、纯路径、未预取、老 manifest、改 pin Receipt 失效与零网络。
- 自由组合：Kit register/prefetch/lifecycle 与 Validation Evidence 通过内容摘要组合，不互相签发事实。
