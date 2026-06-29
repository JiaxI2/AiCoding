# TI XDS / CCS DSS Backend Guide

## 1. 设计目标

本 backend 面向 TI XDS + CCS DSS / DebugServer 链路，参考 DSS_DataVisualizer 的 host-side 访问思路，但默认只开放只读观测能力。

DSS_DataVisualizer 仓库说明它是面向 Texas Instruments 芯片的非侵入式数据可视化实时示波工具，并说明 CCS 支持的芯片和 XDS100v3、XDS110、XDS560v2 Plus 等调试器理论上可支持；其 README 还说明 Qt 通过 JNI 与 DSS Java 接口交互，访问 DebugServer，并通过 JTAG 访问芯片。该项目也列出了 GEL 表达式读/写、寄存器读写、CSV 导出、烧录、运行、挂起、复位等功能。AI Debug Repair Kit 只吸收 DSS/XDS 只读观测链路，默认不开放写入、烧录、运行、挂起或复位。

## 2. 默认安全边界

默认能力：

```text
expression_read = true
register_read = true
capabilities = true
doctor = true
```

默认禁止：

```text
reset = false
halt = false
run = false
flash = false
memory_write = false
expression_write = false
register_write = false
```

## 3. CLI

```powershell
airepair dss profile-template --profile .ai-debug-repair\profiles\ti-dss-readonly.json --output json
airepair dss validate-profile --profile .ai-debug-repair\profiles\ti-dss-readonly.json --output json
airepair dss doctor --profile .ai-debug-repair\profiles\ti-dss-readonly.json --output json
airepair dss capabilities --profile .ai-debug-repair\profiles\ti-dss-readonly.json --output json
airepair dss read-expression --profile .ai-debug-repair\profiles\ti-dss-readonly.json --expression "CpuTimer0Regs.TIM.all" --output json
```

默认 `read-expression` 只生成 DSS JavaScript，不执行。执行需要显式 `--execute`。
