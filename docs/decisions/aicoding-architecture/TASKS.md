# 任务：正交内核闭环

## 研究与边界

- [x] 审计仓库分层、命令、registry、adapter、runner、report 与 domain state。
- [x] 学习 Git MOC 与 12 个索引，重点采用稳定边界与架构停止规则。
- [x] 对照 Git/GitHub/mattpocock/skills。
- [x] 对照 Orthogonal Architecture Design Kit 的 God Core、状态所有权和局部验证规则。
- [x] 明确架构与功能扩展是两类工作。

## 核心对象

- [x] `ExecutionPlan`：descriptor、不可变选择、snapshot、digest。
- [x] Registry Snapshot + Digest：Kit/MCP 规范化 registry。
- [x] Typed Command Catalog：CLI routing、alias、namespace、help。
- [x] Catalog Snapshot：registry + referenced manifest 内容树。
- [x] Static Adapter Catalog：Kit/MCP/runtime Skill descriptor + static function。
- [x] Lifecycle ExecutionPlan：删除 scope switch，串行执行 adapter plan。
- [x] Evidence：catalog/input/plan digest 进入 JSON。

## 场景闭环

- [x] External Skill 的 source pin/package/runtime exposure 分离。
- [x] MCP component 的 manifest/runtime/config/state 分离。
- [x] install/update/sync/uninstall/status/doctor/verify/rollback 语义与所有权。
- [x] Agent/Skill 通过 CLI + JSON 调用，不实现第二 lifecycle。
- [x] 模块级、消费者级、Full/Release 验证半径。
- [x] capability graph/global journal/remote API/C core 的拒绝与解冻条件。

## 验收

- [x] Module/consumer Go tests。
- [x] 真实 CLI digest 与 lifecycle dry-run。
- [x] DocSync、Markdown、governance、hooks、diff checks。
- [x] Doctor、Smoke、Full、Release。
- [x] 子模块 clean 与 worktree diff audit。
- [x] 独立提交验收结果。
