# CLI and MCP Control Plane Architecture

Status: Proposed

## 1. Architecture Conclusion

AiCoding 使用 Go CLI 作为唯一正式产品控制面（Control Plane），
MCP 作为外部能力接入层（Capability Integration Layer）。

CLI 负责：

- 用户入口；
- 命令契约；
- 生命周期编排；
- 计划生成；
- 执行调度；
- 状态管理；
- 报告输出；
- 治理和验证。

MCP 不属于产品工作流层，而属于受控外部能力。

MCP 负责：

- component 注册；
- runtime 管理；
- 协议兼容；
- 安装状态；
- 配置同步；
- capability 暴露。

二者通过统一的 manifest、adapter 和 execution model 连接。

---

# 2. Overall Model

```text
User / Agent

      |
      v

CLI (Porcelain)

      |
      v

Control Plane

      |
      +----------------+
      |                |
      v                v

 Kit Adapter      MCP Adapter

      |                |

 Capability       MCP Component

      |                |

 Runtime          External Runtime
```

设计目标：

新增能力时：

```
新增 Manifest
+
新增 Adapter
+
注册 Capability
```

而不是：

```
修改 CLI
修改 Runner
修改 Report
修改生命周期核心
```

---

# 3. CLI Control Plane

## 3.1 Responsibility

CLI 是 AiCoding 唯一正式入口。

所有：

- Taskfile
- CI
- Hook
- Skill
- Agent workflow

只能调用 CLI。

禁止：

- PowerShell 实现第二套控制逻辑；
- Python 实现第二套生命周期；
- CI 自己聚合业务流程。


---

# 4. CLI Layer Model

```text
cmd/aicoding

        |

internal/cli

        |

Command Catalog

        |

Control Plane

        |

Adapter

        |

Runner

        |

Report
```


---

# 5. Command Catalog

Command Catalog 是 CLI 的唯一命令事实来源。


负责：

- command name；
- arguments；
- flags；
- help；
- JSON contract；
- deprecated command；
- documentation binding。


示例：

```go
Command{
    Name:"lifecycle",
    Description:"Manage capability lifecycle",
    Actions:[
        "plan",
        "apply",
        "status",
        "verify"
    ]
}
```


生成：

```
--help

docs/COMMANDS.md

Taskfile validation

CI command check
```


避免：

```
CLI
README
Taskfile
CI

四份命令列表
```


---

# 6. CLI Command Categories


## bootstrap

初始化环境：

```bash
aicoding bootstrap
```


职责：

- 环境检查；
- CLI 初始化；
- 基础状态建立。


---

## lifecycle

生命周期控制核心：

```bash
aicoding lifecycle plan

aicoding lifecycle apply

aicoding lifecycle status

aicoding lifecycle verify

aicoding lifecycle rollback
```


管理：

```
Kit

MCP

runtime Skill
```


执行模型：

```
Request

  |

Plan

  |

Runner

  |

Journal

  |

Result
```


---

## doctor

环境诊断：

```bash
aicoding doctor --all
```


只负责：

发现问题。


禁止：

自动修改。


---

## verify

结构验证：

```bash
aicoding verify --profile Smoke
```


验证：

- manifest；
- dependency；
- configuration；
- repository state。


---

## test

统一测试引擎：

```bash
aicoding test --profile Smoke

aicoding test --profile Full

aicoding test --profile Release
```


禁止：

创建第二测试入口。


---

# 7. CLI Extension Model


新增能力：

例如：

```
ethercat
cad
plc
simulation
```

流程：

```
Capability Definition

        |

Manifest

        |

Adapter

        |

Command Registration

        |

Automatic Lifecycle Support
```


不允许：

```
switch command {

case ethercat:

}

```

不断扩大核心。


---

# 8. MCP Control Plane


## 8.1 Position


MCP 是外部能力控制层。


关系：

```
AiCoding

 |

MCP Control Plane

 |

MCP Component

 |

Runtime
```


MCP 不负责：

- workflow prompt；
- Skill 编排；
- 用户流程。


---

# 9. MCP Responsibilities


MCP Control Plane 管理：

## Registry

位置：

```
config/mcp-registry.json
```


作用：

登记 MCP component。


---

## Component Manifest


位置：

```
config/mcp/components/*.json
```


描述：

```json
{
"id":"visio-mcp",

"runtime":"python",

"command":"server.py",

"timeout":30,

"verifyProfile":"Smoke"
}
```


---

# 10. MCP Lifecycle


统一流程：

```
discover

   |

validate

   |

install

   |

configure

   |

verify

   |

state record
```


修改状态：

必须：

```
backup

+

journal

+

rollback
```


---

# 11. MCP Adapter Model


MCP 不直接写入 CLI。


通过 Adapter：


```go
type MCPAdapter interface {

    Plan()

    Install()

    Status()

    Verify()

    Remove()

}
```


---

# 12. Adding New MCP Component


例如新增：

```
ethercat-mcp
cad-mcp
plc-mcp
```


步骤：

## Step 1

新增：

```
config/mcp/components/ethercat.json
```


---

## Step 2

注册：

```
config/mcp-registry.json
```


---

## Step 3

提供：

```
MCP package
runtime
verify command
```


---

## Step 4

执行：

```bash
aicoding mcp verify
```


核心无需修改。


---

# 13. CLI and MCP Boundary


正确：

```
CLI

 |

Control Plane

 |

MCP Adapter

 |

MCP Capability
```


错误：

```
CLI

 |

MCP

 |

Workflow
```


原因：

MCP 不应该知道：

- Skill 名称；
- Prompt；
- 用户流程。


---

# 14. Shared Kernel


CLI 和 MCP 共用：

## Manifest Snapshot

统一读取：

- registry；
- component；
- capability。


---

## Plan


所有写操作：

```
plan

↓

apply
```


---

## Runner


负责：

- timeout；
- cancellation；
- bounded concurrency。


---

## Report


统一输出：

```json
{
"status":"success",

"checks":[],

"errors":[]
}
```


---

## Journal


所有状态变化：

```
before

action

after

rollback
```


---

# 15. Design Rules


## Rule 1

Core does not know capability.


核心不知道：

```
Kit
MCP
Skill
EtherCAT
CAD
```

---

## Rule 2

Capability does not know product.


MCP 内禁止：

```
AICODING_xxx
aicoding workflow
```

---

## Rule 3

No dynamic plugin ABI


采用：

```
Static Adapter Registration
```


---

## Rule 4

One Source of Truth


必须唯一：

```
Command Catalog

Manifest

Report Schema

Lifecycle Engine
```


---

# 16. Future Evolution


长期目标：

```
             Agent

               |

          AiCoding CLI

               |

        Control Plane

               |

       Capability Graph

               |

 +-------------+-------------+

 Kit          MCP          Tool

               |

          Runtime
```


AiCoding 不成为工具集合。

AiCoding 成为：

> AI Agent 工程环境控制平面。

```