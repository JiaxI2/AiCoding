# 实现计划：C UserStyle Kit 1.2.0 集成

Plan Status: Approved

## 范围

在不修改 skills submodule 和插件的前提下，导入 C Kit 源码与完整参考资产，登记 Kit，扩展既有
C99 Skill 快速验证，并把真实 fast gate 纳入 Smoke/全局测试和平台 Release `v0.8.0`。

## 实现步骤

1. 导入 C Kit 1.2.0 到 `CodingKit/tools/c-userstyle-kit`，排除本地状态、build 和旧集成草案。
2. 更新集成副本的 manifest、架构、分发说明和 changelog，消除“尚未集成”的陈述。
3. 新增 `config/kits/c-userstyle-kit.json` 并登记到 `config/kit-registry.json`。
4. 在现有 `internal/cstyle`/CLI 路由新增 `verify`，捕获并校验 C Kit JSON。
5. 更新 Skill 元数据、Taskfile、全局测试器、三份 README 和权威 C99 文档。
6. 运行专项验证，再依次运行 Smoke、Full、Release、DocSync、治理与 Hook 门禁。
7. 更新 `CHANGELOG.md` 和双语 Release Notes，提交、推送 main、annotated tag 与 GitHub Release。

## 验证计划

- C Kit：`go -C CodingKit/tools/c-userstyle-kit test ./...` 与 fast/full verify。
- AiCoding：Go test/build、C99 Skill status/templates/check/verify、Kit Smoke/Lifecycle。
- 聚合：`smoke`、`ci --profile Smoke`、`test full`、`test release`、`release gate`。
- 治理：Plan Mode、DocSync、Markdown links、governance lint、tag audit、hooks、`git diff --check`。
- 发布后：回读远端 main、tag 和 GitHub Release body。

## 回滚

发布前：删除本次新增的 C Kit 资产/manifest，移除 registry 与 C99 adapter 变更，并恢复本次文档；
可依据本提交的明确文件清单逐项反向应用。发布后禁止移动 `v0.8.0`，如需修正则提交 revert 并发布
新的 SemVer patch。不会使用 `git reset --hard`、force push 或删除历史 Tag。
