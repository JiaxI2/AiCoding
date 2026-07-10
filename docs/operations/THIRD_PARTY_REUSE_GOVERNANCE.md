# 可复用模块治理

## 目标

本治理只管理经审查后吸收的外部能力，不创建第二套命令、生命周期、配置加载或领域模型。默认控制面始终是 `bin\aicoding.exe`；外部能力只能以可替换的实现或声明性数据接入。

## 分类

| 分类 | 使用条件 | 当前处理 |
|---|---|---|
| 直接复用 | 模块可独立运行、授权与告知材料完整、无运行时反向依赖 | 首轮不引入 |
| 修改后复用 | 适配后仍可单独替换，且变更边界、测试和告知材料完整 | 首轮不引入 |
| 重新实现 | 只保留通用问题与验收目标，使用 AiCoding 的配置和 Go 控制面实现 | 首个试点：证据门禁 |
| 仅吸收思想 | 不引入代码或原始表达，映射到现有 Plan Mode、DocSync、lifecycle 或审查流程 | 规划、上下文加载、对抗式复核、增量交付 |
| 不采用 | 会复制平台专属命令、角色、Shell hook 或建立平行编排 | 专属 slash command、persona 编排、会话缓存脚本、Web 专项流程 |

含有原始表达的复制必须保留所需的授权与告知材料；不能满足该条件时必须改为重新实现或仅吸收思想。产品命令、日志和业务文档不暴露外部来源标识。

## 首个闭环：证据门禁

`config/reuse-governance.json` 是唯一机器配置。首个试点记录为 `reimplemented`，不携带外部运行时依赖或公共 API 耦合，并以以下证据证明已接入现有控制面：

- `aicoding governance reuse --json` 输出独立证据报告；
- `skill verify` 同步验证登记配置；
- pre-commit、`smoke`、`ci`、`full` 与 `release gate` 运行同一门禁；
- DocSync 将该配置和实现视为风险变更，要求同时更新文档；
- `reuse-governance` kit 进入既有 lifecycle registry，只记录本地状态，不复制任何外部资产。

## 演进、移除与回滚

新增模块前，先在登记配置中确定分类、独立性、证据、所需路径和回滚状态路径。直接或修改后复用还必须具备可审计的告知文件；缺失时校验会失败。

升级应先更新独立模块及其测试，再更新登记配置，最后运行完整验证。移除时先禁用 registry 条目、删除模块自有文件，再使用现有 lifecycle rollback 恢复状态；不得修改 Go CLI 核心领域模型、Skill 子模块或插件缓存。

## 验证

```powershell
bin\aicoding.exe governance reuse --json
bin\aicoding.exe skill verify --all --profile Smoke --json
bin\aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe docsync ci --json
```
