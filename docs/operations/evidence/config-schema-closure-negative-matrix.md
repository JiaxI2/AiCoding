# TODO 0035 config schema 闭合负例矩阵

日期：2026-07-22
仓库：`F:\\Study\\AI\\worktrees\\AiCoding`

以下输出均来自当前实施工作树编译出的 `bin\\aicoding.exe`。每次只注入所述单一破坏，
真实运行既有 `governance dependencies`，记录非零退出后立即还原。输出未改写；
`elapsedMs` 是各次真实墙钟结果。

## 1. 新绑定配置非法字段

注入：`config/common-registry.json` 顶层临时注入 `"illegal": true`。

命令：

```powershell
bin\aicoding.exe governance dependencies --repo-root . --json
```

原始退出码：`1`

原始输出：

```json
{
  "schemaVersion": 1,
  "command": "governance dependencies",
  "ok": false,
  "errorKind": "validation",
  "category": "validation",
  "retryable": false,
  "nextAction": "aicoding doctor --all --json",
  "message": "dependency direction governance gate",
  "repoRoot": "F:\\Study\\AI\\worktrees\\AiCoding",
  "data": {
    "schemaVersion": 1,
    "config": "config/dependency-governance.json",
    "direction": "higher-rank-may-depend-on-equal-or-lower-rank",
    "checks": [
      {
        "name": "layer model",
        "ok": true
      },
      {
        "name": "binding declarations",
        "ok": true
      },
      {
        "name": "kit registry coverage",
        "ok": true
      },
      {
        "name": "MCP registry coverage",
        "ok": true
      },
      {
        "name": "declared dependency direction",
        "ok": true
      },
      {
        "name": "lower-layer platform independence",
        "ok": true
      },
      {
        "name": "MCP component identity",
        "ok": true
      },
      {
        "name": "MCP and Skill responsibility boundary",
        "ok": true
      },
      {
        "name": "Skill naming and exposure",
        "ok": true
      },
      {
        "name": "asset identity version opacity",
        "ok": true
      },
      {
        "name": "README version badge authority",
        "ok": true
      },
      {
        "name": "activation manifests URL-free",
        "ok": true
      },
      {
        "name": "cloneable sources registry",
        "ok": true
      },
      {
        "name": "orthogonal Go package boundaries",
        "ok": true
      },
      {
        "name": "git process ownership",
        "ok": true
      },
      {
        "name": "gitx importer allowlist",
        "ok": true
      },
      {
        "name": "policy schema closure",
        "ok": false,
        "errors": [
          "policy config config/common-registry.json violates config/schemas/common-registry.schema.json: $: additional property \"illegal\" is not allowed"
        ]
      },
      {
        "name": "config schema completeness",
        "ok": true
      }
    ],
    "errors": [
      "policy schema closure: policy config config/common-registry.json violates config/schemas/common-registry.schema.json: $: additional property \"illegal\" is not allowed"
    ]
  },
  "errors": [
    "policy schema closure: policy config config/common-registry.json violates config/schemas/common-registry.schema.json: $: additional property \"illegal\" is not allowed"
  ],
  "elapsedMs": 242
}
```

## 2. 未登记 rogue 配置

注入：临时新增 `config/rogue.json`。

命令：

```powershell
bin\aicoding.exe governance dependencies --repo-root . --json
```

原始退出码：`1`

原始输出：

```json
{
  "schemaVersion": 1,
  "command": "governance dependencies",
  "ok": false,
  "errorKind": "validation",
  "category": "validation",
  "retryable": false,
  "nextAction": "aicoding doctor --all --json",
  "message": "dependency direction governance gate",
  "repoRoot": "F:\\Study\\AI\\worktrees\\AiCoding",
  "data": {
    "schemaVersion": 1,
    "config": "config/dependency-governance.json",
    "direction": "higher-rank-may-depend-on-equal-or-lower-rank",
    "checks": [
      {
        "name": "layer model",
        "ok": true
      },
      {
        "name": "binding declarations",
        "ok": true
      },
      {
        "name": "kit registry coverage",
        "ok": true
      },
      {
        "name": "MCP registry coverage",
        "ok": true
      },
      {
        "name": "declared dependency direction",
        "ok": true
      },
      {
        "name": "lower-layer platform independence",
        "ok": true
      },
      {
        "name": "MCP component identity",
        "ok": true
      },
      {
        "name": "MCP and Skill responsibility boundary",
        "ok": true
      },
      {
        "name": "Skill naming and exposure",
        "ok": true
      },
      {
        "name": "asset identity version opacity",
        "ok": true
      },
      {
        "name": "README version badge authority",
        "ok": true
      },
      {
        "name": "activation manifests URL-free",
        "ok": true
      },
      {
        "name": "cloneable sources registry",
        "ok": true
      },
      {
        "name": "orthogonal Go package boundaries",
        "ok": true
      },
      {
        "name": "git process ownership",
        "ok": true
      },
      {
        "name": "gitx importer allowlist",
        "ok": true
      },
      {
        "name": "policy schema closure",
        "ok": true
      },
      {
        "name": "config schema completeness",
        "ok": false,
        "errors": [
          "config JSON is not registered or excluded: config/rogue.json"
        ]
      }
    ],
    "errors": [
      "config schema completeness: config JSON is not registered or excluded: config/rogue.json"
    ]
  },
  "errors": [
    "config schema completeness: config JSON is not registered or excluded: config/rogue.json"
  ],
  "elapsedMs": 255
}
```

## 3. 幽灵排除

注入：`config/schema-closure-exclusions.json` 临时登记不存在的 `config/missing.json`。

命令：

```powershell
bin\aicoding.exe governance dependencies --repo-root . --json
```

原始退出码：`1`

原始输出：

```json
{
  "schemaVersion": 1,
  "command": "governance dependencies",
  "ok": false,
  "errorKind": "validation",
  "category": "validation",
  "retryable": false,
  "nextAction": "aicoding doctor --all --json",
  "message": "dependency direction governance gate",
  "repoRoot": "F:\\Study\\AI\\worktrees\\AiCoding",
  "data": {
    "schemaVersion": 1,
    "config": "config/dependency-governance.json",
    "direction": "higher-rank-may-depend-on-equal-or-lower-rank",
    "checks": [
      {
        "name": "layer model",
        "ok": true
      },
      {
        "name": "binding declarations",
        "ok": true
      },
      {
        "name": "kit registry coverage",
        "ok": true
      },
      {
        "name": "MCP registry coverage",
        "ok": true
      },
      {
        "name": "declared dependency direction",
        "ok": true
      },
      {
        "name": "lower-layer platform independence",
        "ok": true
      },
      {
        "name": "MCP component identity",
        "ok": true
      },
      {
        "name": "MCP and Skill responsibility boundary",
        "ok": true
      },
      {
        "name": "Skill naming and exposure",
        "ok": true
      },
      {
        "name": "asset identity version opacity",
        "ok": true
      },
      {
        "name": "README version badge authority",
        "ok": true
      },
      {
        "name": "activation manifests URL-free",
        "ok": true
      },
      {
        "name": "cloneable sources registry",
        "ok": true
      },
      {
        "name": "orthogonal Go package boundaries",
        "ok": true
      },
      {
        "name": "git process ownership",
        "ok": true
      },
      {
        "name": "gitx importer allowlist",
        "ok": true
      },
      {
        "name": "policy schema closure",
        "ok": true
      },
      {
        "name": "config schema completeness",
        "ok": false,
        "errors": [
          "schema closure exclusion file does not exist: config/missing.json"
        ]
      }
    ],
    "errors": [
      "config schema completeness: schema closure exclusion file does not exist: config/missing.json"
    ]
  },
  "errors": [
    "config schema completeness: schema closure exclusion file does not exist: config/missing.json"
  ],
  "elapsedMs": 242
}
```

## 4. 排除表非法字段

注入：排除表顶层临时注入 `"illegal": true`。

命令：

```powershell
bin\aicoding.exe governance dependencies --repo-root . --json
```

原始退出码：`1`

原始输出：

```json
{
  "schemaVersion": 1,
  "command": "governance dependencies",
  "ok": false,
  "errorKind": "validation",
  "category": "validation",
  "retryable": false,
  "nextAction": "aicoding doctor --all --json",
  "message": "dependency direction governance gate",
  "repoRoot": "F:\\Study\\AI\\worktrees\\AiCoding",
  "data": {
    "schemaVersion": 1,
    "config": "config/dependency-governance.json",
    "direction": "higher-rank-may-depend-on-equal-or-lower-rank",
    "checks": [
      {
        "name": "layer model",
        "ok": true
      },
      {
        "name": "binding declarations",
        "ok": true
      },
      {
        "name": "kit registry coverage",
        "ok": true
      },
      {
        "name": "MCP registry coverage",
        "ok": true
      },
      {
        "name": "declared dependency direction",
        "ok": true
      },
      {
        "name": "lower-layer platform independence",
        "ok": true
      },
      {
        "name": "MCP component identity",
        "ok": true
      },
      {
        "name": "MCP and Skill responsibility boundary",
        "ok": true
      },
      {
        "name": "Skill naming and exposure",
        "ok": true
      },
      {
        "name": "asset identity version opacity",
        "ok": true
      },
      {
        "name": "README version badge authority",
        "ok": true
      },
      {
        "name": "activation manifests URL-free",
        "ok": true
      },
      {
        "name": "cloneable sources registry",
        "ok": true
      },
      {
        "name": "orthogonal Go package boundaries",
        "ok": true
      },
      {
        "name": "git process ownership",
        "ok": true
      },
      {
        "name": "gitx importer allowlist",
        "ok": true
      },
      {
        "name": "policy schema closure",
        "ok": false,
        "errors": [
          "policy config config/schema-closure-exclusions.json violates config/schemas/schema-closure-exclusions.schema.json: $: additional property \"illegal\" is not allowed"
        ]
      },
      {
        "name": "config schema completeness",
        "ok": false,
        "errors": [
          "config/schema-closure-exclusions.json: json: unknown field \"illegal\""
        ]
      }
    ],
    "errors": [
      "policy schema closure: policy config config/schema-closure-exclusions.json violates config/schemas/schema-closure-exclusions.schema.json: $: additional property \"illegal\" is not allowed",
      "config schema completeness: config/schema-closure-exclusions.json: json: unknown field \"illegal\""
    ]
  },
  "errors": [
    "policy schema closure: policy config config/schema-closure-exclusions.json violates config/schemas/schema-closure-exclusions.schema.json: $: additional property \"illegal\" is not allowed",
    "config schema completeness: config/schema-closure-exclusions.json: json: unknown field \"illegal\""
  ],
  "elapsedMs": 233
}
```

## 5. 删除既有 b 类 schema

注入：临时移走 `config/schemas/agent-dev-kit-plan-mode.registry.schema.json`，命令结束前恢复原路径。

命令：

```powershell
bin\aicoding.exe governance dependencies --repo-root . --json
```

原始退出码：`1`

原始输出：

```json
{
  "schemaVersion": 1,
  "command": "governance dependencies",
  "ok": false,
  "errorKind": "validation",
  "category": "validation",
  "retryable": false,
  "nextAction": "aicoding doctor --all --json",
  "message": "dependency direction governance gate",
  "repoRoot": "F:\\Study\\AI\\worktrees\\AiCoding",
  "data": {
    "schemaVersion": 1,
    "config": "config/dependency-governance.json",
    "direction": "higher-rank-may-depend-on-equal-or-lower-rank",
    "checks": [
      {
        "name": "layer model",
        "ok": true
      },
      {
        "name": "binding declarations",
        "ok": true
      },
      {
        "name": "kit registry coverage",
        "ok": true
      },
      {
        "name": "MCP registry coverage",
        "ok": true
      },
      {
        "name": "declared dependency direction",
        "ok": true
      },
      {
        "name": "lower-layer platform independence",
        "ok": true
      },
      {
        "name": "MCP component identity",
        "ok": true
      },
      {
        "name": "MCP and Skill responsibility boundary",
        "ok": true
      },
      {
        "name": "Skill naming and exposure",
        "ok": true
      },
      {
        "name": "asset identity version opacity",
        "ok": true
      },
      {
        "name": "README version badge authority",
        "ok": true
      },
      {
        "name": "activation manifests URL-free",
        "ok": true
      },
      {
        "name": "cloneable sources registry",
        "ok": true
      },
      {
        "name": "orthogonal Go package boundaries",
        "ok": true
      },
      {
        "name": "git process ownership",
        "ok": true
      },
      {
        "name": "gitx importer allowlist",
        "ok": true
      },
      {
        "name": "policy schema closure",
        "ok": false,
        "errors": [
          "policy schema config/schemas/agent-dev-kit-plan-mode.registry.schema.json: open F:\\Study\\AI\\worktrees\\AiCoding\\config\\schemas\\agent-dev-kit-plan-mode.registry.schema.json: The system cannot find the file specified."
        ]
      },
      {
        "name": "config schema completeness",
        "ok": false,
        "errors": [
          "registered schema is missing: config/schemas/agent-dev-kit-plan-mode.registry.schema.json"
        ]
      }
    ],
    "errors": [
      "policy schema closure: policy schema config/schemas/agent-dev-kit-plan-mode.registry.schema.json: open F:\\Study\\AI\\worktrees\\AiCoding\\config\\schemas\\agent-dev-kit-plan-mode.registry.schema.json: The system cannot find the file specified.",
      "config schema completeness: registered schema is missing: config/schemas/agent-dev-kit-plan-mode.registry.schema.json"
    ]
  },
  "errors": [
    "policy schema closure: policy schema config/schemas/agent-dev-kit-plan-mode.registry.schema.json: open F:\\Study\\AI\\worktrees\\AiCoding\\config\\schemas\\agent-dev-kit-plan-mode.registry.schema.json: The system cannot find the file specified.",
    "config schema completeness: registered schema is missing: config/schemas/agent-dev-kit-plan-mode.registry.schema.json"
  ],
  "elapsedMs": 224
}
```

## 6. 全部还原后的正例

五项注入均已还原；恢复后的 `governance dependencies` 返回 `ok=true`，其中
`policy schema closure` 与 `config schema completeness` 同时为绿。归档后的同一 staged tree
以 `--reuse off` 真跑最终 Full；原始 summary 固定保存于
`test-results/0035-final-full/summary.json`。Release 的原始 summary 固定保存于
`test-results/0035-final-release/summary.json`。
