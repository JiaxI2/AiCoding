# C UserStyle Kit 架构

Status: Accepted and Frozen

本文是 AiCoding 平台侧 C UserStyle Kit 的架构权威，只描述现有 1.2.0 行为。Kit 内部的规则证据链
仍由 `CodingKit/tools/c-userstyle-kit/docs/ARCHITECTURE.md:1` 说明。

## 结论

C UserStyle Kit 是 C99 风格裁决面：`config/skills/c99-standard-c/skill.json:1-32` 拥有
formatter、注释模板、规则、Kit 绑定和排除目录配置；`internal/cstyle/cstyle.go:84-175` 与
`internal/cstyle/verify.go:67-191` 负责平台适配，`CodingKit/tools/c-userstyle-kit/cmd/cstylekit/main.go:1`
和 clang-format 执行确定性检查。消费者只有统一 Skill 命令、pre-commit 的 staged check，以及
全局 C99 用例；锚点分别为 `internal/cli/catalog.go:181-185`、`internal/cli/cli.go:369-375`、
`internal/testengine/engine.go:397-404`。Kit 已由 `config/kit-registry.json:31-34` 以 order 75 启用。

## 系统位置图

```text
aicoding skill c99-standard-c {status,templates,fmt,check,verify}
                         │
                         ▼
                 internal/cli + report.Result
                         │
                         ▼
                    internal/cstyle
              ┌──────────┼──────────────┐
              │          │              │
      status/templates  fmt/check      verify
          builtin       external       external-command
          文件校验      clang-format   go -C <kit> run ./cmd/cstylekit
                                             │
                                             └─ host gcc/g++/clang/clang++/测试程序
```

`status` 和 `templates` 在 Go 控制面内读配置并校验文件；`fmt/check` 直接启动 clang-format，
对应 `internal/cli/cli.go:198-290` 与 `internal/cstyle/cstyle.go:355-403`。`verify` 以 6 分钟
上限启动 `go -C <kit> run ./cmd/cstylekit`，对应 `internal/cstyle/verify.go:15-64`；Kit 只运行
host 工具，不接受 shell 命令、不调用 TI/CCS 或固件构建，见
`CodingKit/tools/c-userstyle-kit/README.md:105-120`。manifest 对同一路径的分类是 status=
`builtin-check`、verify/test=`external-command`，见 `config/kits/c-userstyle-kit.json:23-100`。

## 邻居边界判据

- **DocSync** 裁决文档/变更记录同步以及架构文档 Status，不检查 C/H 风格；入口与 Status 范围见
  `internal/docsync/check.go:30-90`、`internal/docsync/check.go:93-120`。
- **governance lint** 裁决仓库治理文件、占位符、依赖方向和提交规范，不替代 formatter/lint；
  其快速检查见 `internal/governance/governance.go:13-125`。
- **C UserStyle** 只接收 `.c/.h`。`changed` 取 HEAD diff 加未跟踪文件，`staged` 取 index diff，
  `all` 遍历仓库，`paths` 使用显式路径；四种语义见 `internal/cstyle/cstyle.go:178-235`。
  Git 查询统一经 `gitx.Run`，不复制 Git 进程边界，见 `internal/cstyle/cstyle.go:406-430`。
- 输入路径必须仍在 repo 内、确实存在、扩展名为 C/H 且不命中排除集；路径逃逸直接不进入候选，
  见 `internal/cstyle/cstyle.go:267-293`。

## 数据契约

| 合同 | 权威与约束 |
| --- | --- |
| Skill 配置 | `config/skills/c99-standard-c/skill.json:1-32` 是 Skill 绑定、工具路径与排除策略权威。 |
| Kit manifest | `config/kits/c-userstyle-kit.json:1-167` 声明 1.2.0、external-cli 命令、导出和 trust。 |
| clang-format | `config/skills/c99-standard-c/style/clang-format.yaml:1-26` 是源；根 `.clang-format:1-27` 是兼容投影。C99-005 对两者的关键字段及 source-of-truth 声明做静态校验，见 `internal/testengine/engine.go:908-926`。 |
| CLI 外壳 | 唯一 `report.Result` 字段为 `schemaVersion/command/ok/errorKind/message/repoRoot/inputDigest/planDigest/checked/data/warnings/errors/elapsedMs`，见 `internal/report/result.go:22-36`；领域细节放入 `data`。 |
| verify payload | 外部进程必须返回单一 `cstylekit.verify.v1` JSON，profile 与请求一致、`ok` 为 true，见 `internal/cstyle/verify.go:158-190`。 |

三个只读示例分别核对环境/绑定、单文件格式和 Kit 主机门禁：

```powershell
bin\aicoding.exe skill c99-standard-c status --json
bin\aicoding.exe skill c99-standard-c check --scope paths --path testdata/style-samples/foc_sample.c --json
bin\aicoding.exe skill c99-standard-c verify --depth fast --json
```

## 可靠性与安全

| 场景 | 现有行为 |
| --- | --- |
| 配置或 Kit 资产缺失 | status/verify 返回非零和具体缺失路径；校验链见 `internal/cstyle/skill.go:172-255`、`internal/cstyle/verify.go:90-127`。 |
| clang-format 缺失 | status 失败；fmt/check 在有候选文件时 fail-closed，见 `internal/cli/cli.go:198-237`、`internal/cstyle/cstyle.go:147-156`。 |
| 外部执行失败 | check 收集 stderr；verify 对超时、坏 JSON、schema/profile/ok 不匹配和非零进程逐项拒绝，见 `internal/cstyle/cstyle.go:362-403`、`internal/cstyle/verify.go:149-190`。 |
| 写入边界 | check 使用 `--dry-run --Werror`，preview 只比较输出，只有 fmt 非 preview 使用 `-i`，见 `internal/cstyle/cstyle.go:362-403`。 |
| 排除目录 | 配置排除 vendor/third_party/generated/Drivers/device/build/out/dist 和 Kit 自身；C99-006 保证集合不丢失，见 `config/skills/c99-standard-c/skill.json:21-31`、`internal/testengine/engine.go:929-964`。 |

Validation Evidence 当前 toolchain digest 只指纹化 Go 与 Git 可执行文件和版本，见
`internal/validationevidence/fingerprint.go:28-36`、`internal/validationevidence/fingerprint.go:134-169`；
因此 clang-format（以及 Kit 使用的 host 编译器）版本**尚未**直接进入该 digest。命令输出仍报告
clang-format 路径/版本，但跨机器复用不能把这视为已绑定的工具链证据。

## 演进边界（明确不做）

- 不扩展到 C/C++ 之外的语言，不把 style check 变成语义自动重构或固件构建器。
- 不新增平行顶层命令；能力继续挂在 `skill c99-standard-c`，外壳继续复用 `report.Result`。
- 不把 DocSync、Git 治理或业务评审规则塞进 cstyle；邻居各自保留单一职责。
- 规则、模板、排除项、Kit 版本或投影变化必须同步源配置、兼容投影、C99-001..008 与本文，
  经 DocSync、Smoke/Full/Release 和评审收口。
- 若要让 Validation Receipt 跨机器证明 formatter/compiler 等价，须单独设计并评审 toolchain
  fingerprint 扩展；本文只登记缺口，不在文档阶段暗改复用语义。
