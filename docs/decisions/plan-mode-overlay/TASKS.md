# 任务：C UserStyle Kit 1.2.0 集成

## 决策与资产

- [x] 核对 AiCoding 架构、Kit registry、C99 Skill 与发布治理。
- [x] 核对 C Kit 1.2.0 内容、工具链、构建产物和完整参考资产。
- [x] 记录选定架构、用户分发授权、版本与 Git Governance Decision。
- [x] 导入受控 C Kit 快照，不修改 skills submodule/plugin/cache。

## 实现

- [x] 登记 C Kit manifest/registry，并保持唯一 C99 用户入口。
- [x] 实现 fast/full verify Go adapter、标准 JSON 报告和单元测试。
- [x] 把真实 fast verify 与资产完整性检查纳入 Taskfile 和全局 Smoke/Full/Release 测试。
- [x] 同步三份 README、命令、C99 指南、测试文档和 changelog。

## 验证与发布

- [x] 通过 C Kit 自测、fast/full 功能门禁与 AiCoding Go 测试。
- [x] 通过 Kit Smoke/Lifecycle、平台 Smoke/Full/Release 和 fresh-clone。
- [x] 通过 Plan Mode、DocSync、Markdown、治理、Hook 与 Git 检查。
- [ ] 提交并推送 main，创建并推送 annotated `v0.8.0` Tag。
- [ ] 创建完整双语 GitHub Release，回读确认发布状态与正文。
