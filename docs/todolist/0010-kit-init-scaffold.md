# TODO 0010: kit init 脚手架（快速扩展接口：生成即合规）

Status: Planned
Verify: bin/aicoding.exe kit init demo-kit --json 后，不改一个字直接 kit verify --kit demo-kit --profile Lifecycle 全绿

> 依赖 0009（管理标准先立，脚手架按标准生成）。
> 这是"既要灵活也要专业"的机械解法：**专业来自生成物天生合规（先验），
> 灵活来自生成后内部随便改（自由区），跑偏由门禁拦（奖励信号）。**
> 开源先例：`brew create`（按模板生成 formula，生成物直接过 `brew audit`）、
> `cargo new`（生成即可 build）。扩展的"快"不靠省步骤，靠**零决策成本**——
> 所有命名/路径/字段约定已由模板替你决定（Convention over Configuration）。

## 背景

现状：新增一个 kit 要手写 manifest（schemaVersion 2、additionalProperties:false、
7 个必填字段）、registry 条目（4 字段）、testdata、边界文档 —— loop-engineering-kit
来源包 9 字段非法/3 必填缺失就是没有脚手架的直接后果。外部功能包装（如 ai-debug-kit
借鉴上游开源项目）更缺标准：pin 在哪、什么不许 copy、入口怎么声明，全靠评审现场发明。

## 实现计划

1. **模板落点 `config/templates/kit/`**（若 governance layout 不允许，退而
   `internal/kit/templates/` 用 go:embed —— 让 layout 门禁裁决，不改 layout 规则）：

   ```text
   config/templates/kit/
   ├── manifest.tmpl.json        schema v2 合规骨架（enabled 不在此——那是 registry 的字段）
   ├── manifest-external.tmpl.json   外部包装变体（trust.thirdParty:true, updatePolicy:pinned）
   ├── boundary-card.tmpl.md     外部包装边界卡（由 ai-debug-kit 边界卡泛化，见 0008）
   └── workspec-example.tmpl.json    testdata 示例
   ```

2. **CLI `aicoding kit init <id> [--external] [--dry-run] --json`**：
   - `<id>` 校验：registry schema 的 pattern `^[a-z0-9][a-z0-9-]*[a-z0-9]$`，
     且不与现有 kit 冲突；`aicoding-` 前缀按 dependency-governance 保留命名空间规则拦截
     （第三方/通用能力不得占用）。
   - 生成：`config/kits/<id>.json`（从模板填 id/name，version 0.1.0，一条 builtin-check
     verify command 指向自身 manifest —— 保证生成即有可跑门禁）+ registry 条目
     （**enabled:false**，order = 现有最大+10）+ `--external` 时附边界卡
     `docs/reference/kits/<id>-BOUNDARY.md`。
   - `--dry-run` 输出将写入的文件清单与内容摘要，不落盘。
   - 幂等：目标已存在即 fail-closed（不覆盖），提示用 `--dry-run` 查看差异。
   - 同步 HelpForm + COMMANDS.md（老规矩）。
3. **外部包装标准**（写进 0009 的 KIT_MANAGEMENT_STANDARD 第 4 节，模板承载）：
   - 上游代码**只允许** submodule pin 或 go.mod 依赖，**禁止 copy 进仓改**
     （ai-debug-kit 边界卡的核心一条，升格为所有外部包装的通则）；
   - 边界卡四栏必填：上游地址与 pin 策略 / 不进控制面的声明或入口 command /
     不承担的门禁 / 同步纪律（上游变更 → 验证 → 前移 pin → 本仓只动引用）；
   - `trust.thirdParty: true` + `updatePolicy: pinned` 是机器可查锚点（0009 门禁项 4 消费）。
4. **生成即合规的回归保障**：contract test 在临时目录跑
   `kit init tmp-kit && kit verify --kit tmp-kit --profile Lifecycle`，断言零编辑全绿。
   模板一旦漂移（schema 变了模板没跟），这条测试先红 —— 模板与 schema 的同步由测试拥有，
   不靠人记。

## 明确不做

- 不做交互式向导（Agent 和脚本是主要调用方，参数即接口）。
- 不自动 enable、不自动写 skills（skills 属 Codex-Skills 仓库，路径①流程不变）。
- 不做 kit 删除/重命名命令（低频操作，手工 + 门禁足够）。
- 不做远程模板/模板市场（模板是仓库资产，随仓库演进）。

## 自测（可信任方式）

```powershell
go test ./internal/kit/... ./internal/cli/...
bin\aicoding.exe kit init demo-kit --dry-run --json      # 清单正确，零落盘（git status 干净）
bin\aicoding.exe kit init demo-kit --json
bin\aicoding.exe kit verify --kit demo-kit --profile Lifecycle --json   # 不改一字，全绿
bin\aicoding.exe kit list --json                          # demo-kit 出现且 enabled:false
bin\aicoding.exe kit init demo-kit --json                 # 二次执行 fail-closed，不覆盖
bin\aicoding.exe kit init aicoding-foo --json             # 保留前缀被拦（贴错误输出）
bin\aicoding.exe kit init demo-ext --external --json      # 边界卡生成，thirdParty:true
bin\aicoding.exe governance layout --json ; bin\aicoding.exe test --profile Full --json
# 清理 demo 后 git status 干净
```

通过判据：生成物零编辑过 Lifecycle 门禁（核心承诺）；幂等 fail-closed；保留前缀拦截；
模板-schema 同步测试存在且在故意改坏模板时变红（本地验证后撤销）。
