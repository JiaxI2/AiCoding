# AiCoding

AiCoding 是面向嵌入式开发的本地 CodingKit 仓库，用于沉淀模块、调试工具、AI 辅助开发流程和 Codex/Git 治理规则。

## 状态

- 默认分支：`main`
- 远程仓库：`https://github.com/JiaxI2/AiCoding.git`
- Git 治理：使用 `CodingKit/agents/skills` 中的 `Git-Skill`
- `CodingKit/agents/skills` 是 submodule，来源于 `https://github.com/JiaxI2/Codex-Skills.git`

## 快速开始

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
git config core.hooksPath .githooks
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/lint-git-governance.ps1 -Mode all
```

## 目录

```text
CodingKit/
  agents/skills/      Codex-Skills submodule
  modules/            可复用模块
  tests/              测试、验证和实验工程
.github/              仓库治理配置
.githooks/            Git hook 入口
scripts/              本仓库 lint 和治理脚本
```

## Git 提交流程

1. 每次提交选择一个主类型：`feat`、`fix`、`docs`、`style`、`refactor`、`perf`、`test`、`build`、`ci`、`chore`。
2. 提交标题使用 `<type>(<scope>): <summary>`。
3. 每次普通提交都更新 `CHANGELOG.md`，并在条目中显式标注类型，例如 `**docs**` 或 `**chore**`。
4. 本仓库通过 `.githooks` 调用 `scripts/lint-git-governance.ps1` 做本地门禁。

## 文档

- [CHANGELOG](CHANGELOG.md)
- [Repository Governance](.github/repository-governance.toml)