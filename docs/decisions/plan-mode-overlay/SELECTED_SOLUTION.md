# 已选方案：C UserStyle Kit 作为 CodingKit 外部工具

Decision Status: Selected

Selected option: 将 C UserStyle Kit 1.2.0 作为 `CodingKit/tools/c-userstyle-kit` 自包含 Go module，
通过 Kit registry 登记，并由既有 `c99-standard-c` Skill 和 Go CLI 提供快速验证。

## 选择理由

- `CodingKit/tools` 是 AiCoding 对确定性 CLI/utility 的权威归属位置。
- 保留嵌套 `go.mod` 可维持 Kit 的独立测试、生成器和验证边界。
- 现有 `internal/cstyle` 继续拥有用户入口，避免第二套顶层命令或重复 Skill。
- 有界子进程只接受固定 C Kit 命令和结构化参数，不执行 target 提供的 shell 命令。
- PDF 与完整参考副本按用户本轮明确授权纳入公开资产。

## Git Governance Decision

- Mode：`release`
- Branch profile：`github-flow-lite`，稳定分支 `main`
- README profile：`minimal-public` / `existing-custom`
- CHANGELOG：`unreleased`，中文优先并保留英文摘要
- Version scheme：SemVer；Release profile：`minor`
- Platform tag：`v0.8.0`，annotated、unsigned，只推目标 Tag
- Artifact storage：Git 源码与 GitHub Release Notes；无独立二进制包
- Commit、branch push、Tag、Release Notes、GitHub Release：`REQUIRED`
- Pull Request、firmware package、VERSION 文件：`NOT APPLICABLE`

## 禁止路径

- 不修改 `CodingKit/agents/skills`、`BUILDINFO.json`、Marketplace、plugin cache 或生成插件。
- 不把 C Kit 复制为 standalone user skill。
- 不提交 `build/`、`.aicoding/`、旧 `integration/` 草案或编译产物。
