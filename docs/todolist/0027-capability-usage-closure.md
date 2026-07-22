# TODO 0027: 能力使用闭环（describe 一次答清"是什么/怎么用/怎么进 agent/怎么验证"）

Status: Planned
Verify: bin/aicoding.exe capability describe --id loop-engineering --json 一次输出：架构图链接 + 3 行 quickstart + activation 状态与命令 + 验证命令；且 README 生成区可点达

> 来源：owner "loopkit 我到现在看不见架构图，也不知道怎么把它弄到 agent 中使用。"
> **诊断：0023 建了 capability registry（注册可见），但"能力 → 怎么用 → 怎么进 agent"
> 这条链只做了第一环。本项补后两环。**

## 一、实测：缺口在哪（不是没信息，是链不全）

`capability describe --id loop-engineering` 现状已输出 summary / 4 条命令 /
architectureDoc / verification。**信息在，但有三个真实缺口：**

| 缺口 | 现状 | 后果 |
|---|---|---|
| **无 quickstart** | 列了 4 条命令，不说怎么串起来、示例输入在哪 | 看得见命令，不知道怎么跑 |
| **无 activation 答案** | 完全没说"怎么进 agent" | owner 的原话就是这个 |
| **架构图是 ASCII 不是渲染图** | LOOP_ENGINEERING_ARCHITECTURE 是文本框图（0025 的四张 Mermaid 没覆盖它） | "看不见架构图" |
| **README 不可达** | 得先知道 `capability describe` 存在 | 新用户零入口 |

## 二、一个必须澄清的概念混淆（本项的关键洞察）

owner 说"不知道怎么把 loopkit 弄到 agent 中使用" —— 这里混淆了两个东西：

```text
loopkit（domain-capability）    公共入口 = aicoding work validate/next/status/record
                               ★ 已经可用！agent 直接调这些命令即可，无需"导入"

loop-engineering-kit（kit）      enabled:false 的资产包
                               ← owner 以为要先"启用它"才能用 loopkit，其实不用
```

**loopkit 现在就能用** —— agent 调 `aicoding work next` 就是在用它。
困惑纯粹来自文档没说清"CLI-entry 类能力的 activation = 命令本身已存在，
agent 直接调即可；kit 打包是另一回事、可选"。

**因此 activation 要按能力类型分两种答案：**

```text
CLI-entry 能力（loopkit / validation / plan / test …）
  → activation = 命令已在 typed catalog，agent 直接调。describe 给"agent 调用示例"
kit-packaged 能力（需 lifecycle install 才落地到 agent 可见位置的）
  → activation = register → (0026 pinned) → lifecycle install → agent 可见路径
```

## 三、实现计划（聚焦有公共入口的 ~8 个能力，不是 28 个）

1. **capability registry schema 扩展**（可选字段，向后兼容）：
   ```json
   "quickstart": {
     "steps": ["aicoding work validate --file <spec>", "aicoding work next --file <spec>"],
     "exampleInput": "testdata/loopkit/examples/project-development.work.json"
   },
   "activation": {
     "kind": "cli-entry",          // 或 "kit-install"
     "note": "命令已在 CLI，agent 直接调用；work 系列无需 install",
     "agentUsage": "aicoding work next --file <spec> --json"
   }
   ```
   **有 publicEntries 的能力必须填 quickstart + activation**；internal-only / primitive 不要求。

2. **`capability describe` 一次答清五问**（组合既有字段，不新增数据源）：
   ```text
   ① 是什么      summary
   ② 架构图      architectureDoc（本项确保它是渲染 Mermaid，见第 4 步）
   ③ 怎么用      quickstart.steps + exampleInput
   ④ 怎么进 agent activation.kind + agentUsage
   ⑤ 怎么验证    verification
   + 当前状态    该 kit 是 registered / imported（CLI-entry 类恒为"已可用"）
   ```

3. **README 生成区接入**（复用 0023 的 `<!-- BEGIN GENERATED: CAPABILITIES -->`）：
   每个能力一行 + describe 深链，`capability index --write` 生成，docsync 门禁防漂移。

4. **loopkit 架构图补渲染 Mermaid**（0025 漏了它）：
   给 `LOOP_ENGINEERING_ARCHITECTURE.md` 的"系统位置"与"状态机"两张 ASCII 图
   各补一张 Mermaid 版（≤20 节点，与 0025 同规则），DOCS-006 校验图中命令真实存在。
   **这是"看不见架构图"的直接修复。**

5. **governance capabilities 门禁加一条**：
   `status: stable` 且有 publicEntries 的能力 → **必须有 quickstart + activation**，
   缺失即红（防止"命令存在但没人知道怎么用"重演）。

## 四、明确不做

- 不为 internal-only / primitive 能力写 quickstart（它们不面向用户）。
- 不新建第二注册表 —— quickstart/activation 是 internal-capabilities.json 的可选字段。
- 不做交互式教程 / 视频（参数即接口，示例输入即文档）。
- **不改变 loopkit 的"命令已可用"事实** —— 本项是补文档链，不是加激活步骤。
- 不实现 `loop run`（一如既往）。

## 五、自测（可信任方式）

```powershell
go test ./internal/capability/... ./internal/governance/... ./internal/docsync/...

bin\aicoding.exe capability describe --id loop-engineering --json
#   断言一次输出含：architectureDoc、quickstart.steps、activation.agentUsage、verification
#   activation.kind == "cli-entry"、note 说明 work 系列无需 install

# 端到端：照 describe 的 quickstart 真跑一遍 loopkit（证明文档可执行）
bin\aicoding.exe work validate --file testdata/loopkit/examples/project-development.work.json --json
bin\aicoding.exe work next --file testdata/loopkit/examples/project-development.work.json --json

bin\aicoding.exe capability index --write ; git diff README.md docs/CAPABILITIES.md
lychee --config lychee.toml README.md docs/          # describe 深链 + Mermaid 图链接全通

# 负例：
#  1) stable + publicEntries 但删掉 quickstart → governance capabilities 必须红
#  2) LOOP 架构文档的 Mermaid 写一个不存在的命令 → DOCS-006 红
#  3) README 生成区手改 → docsync all 红

bin\aicoding.exe test --profile Full --json
```

通过判据：
1. `describe loop-engineering` 一次答清五问（贴输出）。
2. 按 quickstart 真能跑通 loopkit（贴 work validate/next 输出）。
3. activation 明确告诉用户"work 命令已可用，无需 install"（消除概念混淆）。
4. LOOP 架构文档有渲染 Mermaid（GitHub 渲染确认）。
5. README 生成区可点达每个能力的 describe。
6. 三条负例被抓。
7. ~8 个有公共入口的能力全部有 quickstart + activation；internal-only 零要求。
