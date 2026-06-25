# AI Debug Kit for Codex
## 平台化需求规格、技术方案、实施计划与测试计划

> **文档类型**：PRD + SRS + 架构约束 + 实施计划 + 测试验收计划  
> **实现对象**：OpenAI Codex CLI / Codex IDE / Codex App  
> **项目代号**：`ai-debug-kit`  
> **版本**：`v0.2.0-draft`  
> **日期**：2026-06-23  
> **首版交付范围**：双 Skill + CLI + Debug Core + Simulator/Replay  
> **扩展预留**：MCP Gateway、真实调试器、平台 Pack、领域扩展 Skill  
> **参考项目**：`Aladdin-Wang/Mklink-AI-Probe`

---

# 0. 最新需求结论

本项目第一版采用以下交付形态：

```text
Codex / 其他命令型 Agent
            │
            ├── Skill A：Kit 部署与能力验证
            │
            └── Skill B：通用 Debug 操作规范
                        │
                        ▼
                    ai-debug CLI
                        │
                        ▼
                  Debug Application
                        │
                        ▼
                     Core / Ports
                        │
             ┌──────────┴──────────┐
             ▼                     ▼
        Backend Adapter       Platform Pack
             │                     │
             └──────────┬──────────┘
                        ▼
           Simulator / Replay / Probe / Target
```

两个 Skill 的职责必须严格分离：

## Skill A：Kit 部署与能力验证

解决以下问题：

> 如何让 Codex 或其他 Agent 在不同操作系统、不同工作区、不同工具链和不同硬件平台上，快速部署、配置并验证 AI Debug Kit 是否可用。

## Skill B：通用 Debug 操作规范

解决以下问题：

> 在 Kit 已经部署完成的前提下，Agent 应当如何安全、可重复、可审计地调用 Debug Kit 完成通用调试操作。

Skill B **不负责**：

- 具体业务故障分析；
- 根因推断；
- 假设管理；
- 自动提出修改方案；
- FOC、EtherCAT、OTA 等领域知识；
- 自动调参；
- 自动修复代码；
- 依据波形主观判断业务是否正常。

领域分析能力必须作为独立扩展 Skill、项目内规则或用户提供的测试规范存在，不属于平台核心。

第一版必须实现：

1. `ai-debug-kit-deploy` Skill；
2. `ai-debug-operations` Skill；
3. `ai-debug` CLI；
4. 可独立运行的 Debug Core；
5. Simulator Backend；
6. Replay Backend；
7. 标准化能力描述；
8. 标准化调试会话；
9. 标准化操作结果和报告；
10. MCP 接口目录与映射设计，但不把 MCP 作为运行依赖。

---

# 1. 项目背景

嵌入式调试通常依赖多个独立工具：

- CCS、Keil、IAR、STM32CubeIDE；
- J-Link、XDS、CMSIS-DAP、ST-Link、MKLink；
- OpenOCD、GDB、pyOCD、厂商烧录工具；
- UART、RTT、SWO、CAN、EtherCAT；
- MATLAB/Simulink；
- 示波器和 HIL 测试系统。

Codex 可以读取代码、执行命令和修改工程，但如果直接接触各类调试工具，容易出现：

- 命令格式不统一；
- 返回结果不可机器稳定解析；
- Agent 与特定 Probe、芯片和 IDE 强耦合；
- 地址单位、字节序、地址空间处理错误；
- 高风险操作缺少审批；
- 调试过程不可审计；
- 不同平台部署流程差异大；
- Skill 混入大量底层实现；
- 平台功能与业务分析边界不清。

本项目的目标不是建立“自动诊断所有嵌入式问题”的 AI，而是建立：

> 一个可以被 Agent 稳定部署、验证、调用和扩展的通用 Debug Kit 平台。

---

# 2. 产品定位

## 2.1 产品定义

AI Debug Kit 是一个面向 Codex 等命令型 Agent 的嵌入式调试基础设施。

它提供：

- 统一 CLI；
- 统一数据模型；
- 统一调试会话；
- 统一能力发现；
- 统一安全策略；
- 可插拔 Backend；
- 可插拔 Platform Pack；
- 可重放测试环境；
- Agent Skill 使用规范。

它不直接提供完整的业务故障专家系统。

## 2.2 平台核心与领域扩展边界

```text
平台核心负责：
连接、读取、写入、暂停、复位、采集、快照、符号解析、
能力协商、审批、记录、回放、报告、契约测试。

领域扩展负责：
业务变量选择、业务判据、控制算法判断、故障根因模型、
参数整定规则、业务修复方案、领域验收标准。
```

例如：

| 能力 | 平台核心 | 领域扩展 |
|---|---:|---:|
| 读取 `motor.iq` | 是 | 否 |
| 采集 `motor.iq` 波形 | 是 | 否 |
| 计算 RMS/峰值 | 可选通用工具 | 否 |
| 判断电流环参数是否合理 | 否 | 是 |
| 自动修改 PI 参数 | 否 | 是 |
| 读取 EtherCAT 状态寄存器 | 是 | 否 |
| 判断 FoE 状态机设计是否正确 | 否 | 是 |
| 捕获 HardFault 寄存器 | 是 | 否 |
| 根据项目上下文推断根因 | 否 | 是 |

## 2.3 首版产品形态

| 模块 | v0.1 状态 | 职责 |
|---|---:|---|
| Kit Deploy Skill | 必须 | 安装、配置、Agent 部署、平台验证 |
| Debug Operations Skill | 必须 | 通用调试操作顺序、安全和记录规范 |
| CLI | 必须 | 稳定执行 Kit 能力 |
| Debug Core | 必须 | 会话、能力、工件、内存、变量、采集、报告 |
| Simulator | 必须 | 无硬件验证 |
| Replay | 必须 | 离线回放和回归 |
| MCP Gateway | 仅预留 | 后续映射相同 Application Service |
| 领域分析 Skill | 不包含 | 独立扩展项目 |

---

# 3. 建设目标

## 3.1 一级目标

### G-01：跨环境快速部署

Codex 可以根据 Skill A，在 Windows、Linux 和工作区内完成 Kit 初始化、依赖检查、配置生成和 Smoke Test。

### G-02：能力真实验证

系统不只检查“文件是否存在”，还应验证：

- CLI 是否可执行；
- Backend 是否能加载；
- Target Profile 是否合法；
- Artifact Provider 是否可用；
- Simulator 是否能完成读写和采集；
- 真实设备是否能完成声明的只读操作；
- Skill 是否能被 Agent 发现。

### G-03：通用 Debug 操作标准化

Skill B 指导 Agent 通过统一步骤完成：

```text
准备 → 建立会话 → 检查身份 → 查询能力
→ 执行通用调试动作 → 验证动作结果
→ 保存证据 → 关闭会话 → 生成报告
```

### G-04：平台解耦

Core 不依赖：

- 特定 Probe；
- 特定芯片；
- 特定 IDE；
- 特定工具链；
- 特定 Agent；
- 特定业务领域。

### G-05：确定性与可审计

所有调试动作必须：

- 有结构化输入；
- 有结构化输出；
- 有错误码；
- 有超时；
- 有副作用说明；
- 有审批策略；
- 有会话记录；
- 可在 Replay 中重现。

### G-06：MCP 无损扩展

CLI 和未来 MCP 共用：

- Application Service；
- Request/Response Model；
- Error Model；
- Policy；
- Backend；
- Session。

---

# 4. 非目标

v0.1 不要求：

- 业务根因自动分析；
- 假设管理；
- 领域 Playbook；
- 自动调参；
- 自动修改业务源码；
- 自动验证业务控制性能；
- 自动运行高功率电机；
- 自动完成 FoE/OTA 业务判断；
- 完整桌面 GUI；
- 远程实验室管理；
- MCP 正式交付；
- 支持所有芯片和所有调试器；
- 自研 SWD/JTAG 底层协议；
- 替代 GDB、CCS、Keil；
- 自动处理 OTP、熔丝、安全锁；
- LLM 自己决定操作是否成功。

---

# 5. Skill 体系设计

## 5.1 Skill 总体结构

第一版使用两个独立 Skill：

```text
.agents/skills/
├── ai-debug-kit-deploy/
└── ai-debug-operations/
```

可选增加极薄路由 Skill：

```text
.agents/skills/ai-debug-kit/
```

路由 Skill 不是 v0.1 必选项，不得包含业务逻辑。

---

# 6. Skill A：Kit 部署与能力验证

## 6.1 Skill 名称

```text
ai-debug-kit-deploy
```

## 6.2 目标

让 Agent 在未知或半未知环境中完成：

- 环境识别；
- Kit 安装；
- CLI 验证；
- Agent Skill 部署；
- 工作区配置；
- Backend/Platform 探测；
- Simulator Smoke Test；
- 真实设备分级验证；
- 部署报告生成。

## 6.3 触发条件

以下请求应触发 Skill A：

- 安装 AI Debug Kit；
- 在新电脑部署 Kit；
- 让 Codex 能使用 Kit；
- 验证当前环境是否支持 Kit；
- 检查 MKLink/OpenOCD/CCS 是否可用；
- 初始化目标板配置；
- 检查某个平台能支持哪些能力；
- 修复 Kit 安装或依赖问题；
- 生成部署报告；
- 验证 Skill 是否安装成功。

## 6.4 不应触发

以下请求不应由 Skill A 处理：

- 分析具体固件 Bug；
- 读取某个业务变量并解释含义；
- 判断控制环是否稳定；
- 修改参数；
- 分析业务日志；
- 推断 HardFault 根因；
- 设计 FoE 升级流程。

这些请求应转交 Skill B 或独立领域 Skill。

## 6.5 部署流程

```text
1. 识别 Agent 和工作区
2. 检查操作系统与 Shell
3. 检查 Python、uv、Git
4. 检查 ai-debug CLI
5. 创建/校验虚拟环境
6. 安装 Kit
7. 初始化 .ai-debug 配置
8. 安装两个 Skill
9. 校验 Skill 元数据
10. 运行 doctor
11. 运行 Simulator Smoke Test
12. 探测可选 Backend
13. 生成 Capability Profile
14. 生成 Deployment Report
```

## 6.6 验证等级

### Level 0：安装验证

验证：

- Python；
- uv；
- 包安装；
- CLI；
- 配置；
- Skill 文件。

### Level 1：Simulator 验证

验证：

- 建立 Session；
- 加载 Fixture Artifact；
- 读取变量；
- 读取内存；
- 采集数据；
- 保存 Session；
- 生成报告；
- Replay。

### Level 2：真实设备只读验证

验证：

- Probe 发现；
- 连接；
- Target ID；
- CPU 状态；
- 内存读取；
- 符号变量读取；
- 只读日志或采样。

### Level 3：受控设备操作验证

验证：

- halt；
- resume；
- reset；
- RAM write/readback。

必须显式批准。

### Level 4：高风险操作验证

包括：

- Flash；
- 擦除；
- OTP；
- 安全配置；
- 电机运行；
- OTA。

v0.1 不自动执行。

## 6.7 输出文件

```text
.ai-debug/deployment/
├── environment.json
├── installation.json
├── agents.json
├── toolchains.json
├── backends.json
├── platforms.json
├── capabilities.json
├── smoke-test.json
├── active-profile.json
└── deployment-report.md
```

## 6.8 Skill A 约束

Skill A 不得：

- 未经批准修改系统级 PATH；
- 自动安装内核驱动；
- 自动卸载已有工具链；
- 自动执行 Flash；
- 自动执行 reset/halt；
- 自动运行电机；
- 将“发现可执行文件”等同于“能力已验证”；
- 将未执行的能力标记为 PASS；
- 在用户目录之外写入未知路径；
- 通过 `shell=True` 执行命令。

---

# 7. Skill B：通用 Debug 操作规范

## 7.1 Skill 名称

```text
ai-debug-operations
```

## 7.2 定位

Skill B 是 Agent 使用 Debug Kit 的通用操作规范。

它指导 Agent：

- 如何建立和关闭调试会话；
- 如何确认目标、固件和符号身份；
- 如何查询能力；
- 如何执行只读调试动作；
- 如何申请高风险动作；
- 如何验证命令执行结果；
- 如何保存原始证据；
- 如何记录副作用；
- 如何生成可审计报告。

它不是业务分析 Skill，也不是故障专家系统。

## 7.3 Skill B 的核心边界

### Skill B 可以做

- 检查部署 Profile；
- 检查 Kit 和 CLI 版本；
- 建立 Session；
- 连接/断开设备；
- 查询 Capability；
- 加载 Artifact；
- 校验 Artifact Hash；
- 读取内存；
- 读取寄存器；
- 读取符号变量；
- 捕获通用日志；
- 捕获通用 Telemetry；
- 捕获 CPU/Fault Snapshot；
- 保存快照；
- 执行用户明确指定的操作；
- 执行项目已有的确定性测试脚本；
- 比较原始值或用户提供的阈值；
- 调用 CLI Verifier；
- 导出 Session；
- 生成操作报告。

### Skill B 不可以做

- 自行建立业务假设；
- 管理假设优先级；
- 推断业务根因；
- 将通用指标解释为业务结论；
- 自动选择业务关键变量；
- 自动提出业务参数修改；
- 自动执行 PI/PID 调参；
- 自动生成领域修复方案；
- 判断 FOC 控制器优劣；
- 判断 EtherCAT 状态机业务设计；
- 判断 OTA 策略正确性；
- 根据少量波形直接声明问题已解决；
- 把 Agent 自己的自然语言判断作为 PASS。

## 7.4 Skill B 的标准操作流程

```text
A. 确认任务是通用调试操作，而不是领域分析
B. 读取 active-profile.json
C. 检查 Profile 是否有效和未过期
D. 执行轻量 doctor
E. 创建 Session
F. 确认 Target、Backend、Core、Artifact
G. 查询实际 Capability
H. 确认请求动作和风险级别
I. 执行允许的调试动作
J. 校验退出码、JSON Envelope 和读取长度
K. 保存原始输出与快照
L. 根据用户提供的判据或项目测试执行 Verifier
M. 记录副作用、警告和未验证项
N. 关闭 Session
O. 导出操作报告
```

## 7.5 通用操作分类

Skill B 只按“操作性质”分类，不按“业务故障类型”分类。

### Inspect

- 检查版本；
- 检查配置；
- 检查 Artifact；
- 查询符号；
- 查询能力。

### Observe

- 读内存；
- 读寄存器；
- 读变量；
- 采集日志；
- 采集 Telemetry；
- 捕获 Snapshot。

### Control

- halt；
- resume；
- reset；
- step；
- breakpoint。

需要 Capability 和审批。

### Modify

- RAM write；
- register write；
- Flash；
- 配置修改。

必须通过 Policy；Flash 默认关闭。

### Validate

- readback；
- hash；
- 长度；
- 状态；
- 用户阈值；
- 项目测试脚本；
- 预定义 Contract。

### Record

- actions；
- observations；
- side effects；
- warnings；
- artifacts；
- final report。

## 7.6 Skill B 完成标准

一个通用调试操作只有满足以下条件才算完成：

- 命令退出码明确；
- JSON `ok/code` 明确；
- 目标和 Artifact 身份已记录；
- 实际 Capability 已记录；
- 读取范围和返回长度一致；
- 写操作完成 readback；
- 高风险动作有批准记录；
- 原始证据已保存；
- 未验证项被明确标记；
- Session 已关闭或安全保留；
- 报告路径已输出。

## 7.7 Skill B 禁止的措辞

除非有独立领域规则或用户提供的验收标准，Skill B 不得输出：

- “根因是……”
- “控制参数太大……”
- “应该降低 Kp……”
- “这个波形证明电机失步……”
- “问题已经修复……”
- “该 OTA 设计正确……”
- “该 EtherCAT 状态机正常……”

允许输出：

- “已读取变量 X，值为 Y。”
- “命令返回成功，读取长度为 N。”
- “采集数据中峰值为 Y；本 Skill 不解释其业务含义。”
- “项目提供的测试脚本返回 PASS。”
- “用户提供的阈值检查通过。”
- “未提供业务判据，因此仅完成数据采集和记录。”

## 7.8 与领域 Skill 的交接

如果用户要求业务分析，Skill B 应：

1. 完成必要的通用证据采集；
2. 不擅自解释业务含义；
3. 查找项目内是否存在领域 Skill；
4. 如果存在，传递 Session Bundle；
5. 如果不存在，输出缺少的领域判据；
6. 保留原始数据和 Capability Profile。

交接对象示例：

```text
motor-control-debug
ethercat-debug
foe-ota-debug
c28x-multicore-debug
power-stage-validation
```

这些不属于本仓库 v0.1 核心交付。

---

# 8. 两个 Skill 的交接协议

## 8.1 Skill A 输出

```text
.ai-debug/deployment/active-profile.json
```

示例：

```json
{
  "schema_version": "1.0",
  "kit_version": "0.1.0",
  "agent": "codex",
  "workspace": "F:/workspace/project",
  "installation_status": "ready",
  "validated_at": "2026-06-23T15:00:00Z",
  "backend": "simulator",
  "platform": "generic",
  "capabilities": {
    "artifact_load": true,
    "memory_read": true,
    "memory_write": true,
    "variable_read": true,
    "telemetry_capture": true,
    "fault_snapshot": true,
    "flash": false
  },
  "evidence": {
    "smoke_test": ".ai-debug/deployment/smoke-test.json"
  }
}
```

## 8.2 Skill B 输入检查

Skill B 启动时检查：

- 文件存在；
- Schema 支持；
- Kit 版本匹配；
- 工作区匹配；
- Backend 可加载；
- Profile 未被标记失效；
- 当前目标与验证目标一致。

状态处理：

```text
ready
→ 正常进入调试操作

partial
→ 仅使用已验证能力

stale
→ 运行轻量重新验证

missing / invalid
→ 交回 Skill A
```

## 8.3 Session Bundle 交接

Skill B 输出：

```text
.ai-debug/sessions/<session-id>/
```

领域 Skill 只能读取 Session Bundle 和调用标准 CLI，不应直接访问 Backend 内部对象。

---

# 9. CLI 产品需求

## 9.1 CLI 名称

```bash
ai-debug
```

## 9.2 CLI 分区

### 部署与验证命令

```bash
ai-debug setup init
ai-debug setup install
ai-debug setup status
ai-debug agent detect
ai-debug agent install-skills
ai-debug agent validate-skills
ai-debug doctor
ai-debug backend list
ai-debug backend discover
ai-debug backend validate
ai-debug platform list
ai-debug platform validate
ai-debug smoke-test
ai-debug deployment report
```

### 通用 Debug 操作命令

```bash
ai-debug session new
ai-debug session status
ai-debug session close
ai-debug session export

ai-debug connect
ai-debug disconnect

ai-debug target show
ai-debug backend capabilities

ai-debug artifact load PATH
ai-debug artifact inspect
ai-debug artifact identity
ai-debug artifact symbols
ai-debug artifact resolve NAME
ai-debug artifact addr2line ADDRESS

ai-debug memory read ADDRESS LENGTH
ai-debug memory snapshot
ai-debug memory diff A B
ai-debug memory write ADDRESS DATA

ai-debug register read NAME
ai-debug register write NAME VALUE

ai-debug variable read NAME
ai-debug variable read-many NAME...
ai-debug variable write NAME VALUE

ai-debug telemetry capture SIGNAL...
ai-debug telemetry export INPUT

ai-debug fault capture
ai-debug fault decode SNAPSHOT

ai-debug verify readback
ai-debug verify threshold
ai-debug verify script

ai-debug report generate
```

## 9.3 全局参数

```bash
--config PATH
--workspace PATH
--profile PATH
--session-id ID
--output text|json
--timeout SECONDS
--log-level LEVEL
--dry-run
--no-color
--version
```

## 9.4 JSON Envelope

所有命令支持：

```json
{
  "schema_version": "1.0",
  "ok": true,
  "code": "OK",
  "message": "Operation completed",
  "data": {},
  "warnings": [],
  "side_effects": [],
  "duration_ms": 12,
  "trace_id": "tr-001",
  "session_id": "dbg-001"
}
```

## 9.5 错误码

| 退出码 | 代码 | 含义 |
|---:|---|---|
| 0 | OK | 成功 |
| 2 | INVALID_ARGUMENT | 参数错误 |
| 3 | DEPENDENCY_MISSING | 依赖缺失 |
| 4 | DEVICE_NOT_FOUND | 设备未发现 |
| 5 | RESOURCE_BUSY | 资源占用 |
| 6 | CAPABILITY_UNSUPPORTED | 能力不支持 |
| 7 | ARTIFACT_ERROR | 工件错误 |
| 8 | IO_ERROR | 读写失败 |
| 9 | POLICY_DENIED | 策略拒绝 |
| 10 | VALIDATION_FAILED | 验证失败 |
| 11 | TIMEOUT | 超时 |
| 12 | CANCELLED | 取消 |
| 13 | PROFILE_INVALID | Profile 无效 |
| 20 | INTERNAL_ERROR | 内部错误 |

Agent 必须同时检查：

- 进程退出码；
- JSON `ok`；
- JSON `code`；
- warning；
- side effects。

---

# 10. Debug Core 架构

## 10.1 分层

```text
Presentation
├── CLI
├── Skill
└── Future MCP

Application
├── DeploymentService
├── SessionService
├── DeviceService
├── ArtifactService
├── MemoryService
├── VariableService
├── TelemetryService
├── FaultService
├── ValidationService
└── ReportService

Domain/Core
├── Models
├── Capability
├── Policy
├── Events
├── Errors
└── Contracts

Infrastructure
├── Backends
├── Platform Packs
├── Artifact Providers
├── Storage
└── Subprocess Adapter
```

## 10.2 Core 约束

Core 不得：

- import 某个具体 Backend；
- 固定 Cortex-M 寄存器地址；
- 固定 `arm-none-eabi-*`；
- 假设小端；
- 假设 8-bit address unit；
- 假设单核；
- 假设统一地址空间；
- 假设所有读取都是非侵入式；
- 直接输出终端文本；
- 直接依赖 Codex 或 MCP。

---

# 11. 核心接口

## 11.1 Backend Protocol

```python
class DebugBackend(Protocol):
    async def discover(self) -> list[DeviceDescriptor]: ...
    async def connect(self, target: TargetConfig) -> ConnectionInfo: ...
    async def disconnect(self) -> None: ...
    async def capabilities(self) -> Capabilities: ...
    async def read_memory(self, request: MemoryReadRequest) -> MemoryBlock: ...
    async def write_memory(self, request: MemoryWriteRequest) -> ActionResult: ...
    async def read_registers(self, request: RegisterReadRequest) -> RegisterSet: ...
    async def control(self, request: ControlRequest) -> ActionResult: ...
    async def capture(self, request: CaptureRequest) -> CaptureResult: ...
```

## 11.2 Artifact Provider

```python
class ArtifactProvider(Protocol):
    def load(self, path: Path) -> ArtifactIdentity: ...
    def list_symbols(self, query: SymbolQuery) -> list[Symbol]: ...
    def resolve_variable(self, path: str) -> TypedLocation: ...
    def resolve_source(self, address: TargetAddress) -> SourceLocation: ...
```

## 11.3 Platform Pack

```python
class PlatformPack(Protocol):
    def address_model(self) -> AddressModel: ...
    def register_model(self) -> RegisterModel: ...
    def fault_decoder(self) -> FaultDecoder | None: ...
    def default_policy(self) -> Policy: ...
```

## 11.4 Validation Service

只执行确定性检查：

- 退出码；
- 返回长度；
- hash；
- readback；
- 用户阈值；
- 用户正则；
- 项目测试脚本；
- Contract；
- Fixture 基准。

不负责业务结论推断。

---

# 12. 地址与数据模型

## 12.1 地址模型

必须显式描述：

```yaml
address_unit_bits: 8
endianness: little
pointer_width_bits: 32
memory_spaces:
  - data
  - program
```

禁止把：

```text
地址增量 = 字节数
```

作为全局假设。

## 12.2 C28x 支持约束

为 C28x 预留：

```yaml
architecture: c28x
address_unit_bits: 16
endianness: little
cores:
  - cpu1
  - cpu2
  - cm
memory_spaces:
  - program
  - data
```

所有 API 使用：

- `TargetAddress`
- `AddressSpace`
- `AddressUnitCount`
- `OctetLength`

避免歧义。

## 12.3 TypedValue

```json
{
  "name": "motor.iq",
  "type": "float32",
  "address": {
    "space": "data",
    "value": 8192,
    "address_unit_bits": 16
  },
  "raw_octets": "00002040",
  "value": 2.5,
  "unit": "A"
}
```

---

# 13. 会话与审计

## 13.1 Session 状态

```text
CREATED
READY
CONNECTED
ACTIVE
CLOSING
COMPLETED
FAILED
CANCELLED
```

## 13.2 Action Record

```json
{
  "action_id": "act-001",
  "operation": "variable.read",
  "requested_by": "codex",
  "risk_level": "L1",
  "approved": true,
  "started_at": "...",
  "completed_at": "...",
  "result_code": "OK",
  "side_effects": []
}
```

## 13.3 Session Bundle

```text
.ai-debug/sessions/<id>/
├── manifest.json
├── target.json
├── artifact.json
├── capabilities.json
├── policy.json
├── actions.jsonl
├── observations.jsonl
├── snapshots/
├── telemetry/
├── validation.json
├── warnings.json
└── final-report.md
```

---

# 14. 安全策略

## 14.1 风险等级

| 等级 | 类型 | 默认策略 |
|---|---|---|
| L0 | 配置、解析、查询 | 自动允许 |
| L1 | 只读设备操作 | 自动允许 |
| L2 | halt/resume/step | 需确认 |
| L3 | reset/RAM write/register write | 明确批准 |
| L4 | Flash/擦除/启动硬件 | 默认关闭 |
| L5 | OTP/熔丝/安全锁 | 禁止 |

## 14.2 审批参数

```bash
--approve-control
--approve-reset
--approve-write
--approve-flash
--policy PATH
```

## 14.3 写操作要求

所有写操作必须：

1. 解析目标；
2. 显示地址和类型；
3. 记录原始值；
4. 检查 Policy；
5. 获得批准；
6. 执行写入；
7. readback；
8. 记录结果；
9. 支持可回滚时生成 rollback 数据。

---

# 15. 技术栈

## 15.1 首版必选

| 类别 | 技术 |
|---|---|
| 语言 | Python 3.11+ |
| 包管理 | uv |
| 项目配置 | pyproject.toml |
| CLI | Typer |
| 数据模型 | Pydantic v2 |
| 配置 | TOML + YAML |
| ELF/DWARF | pyelftools |
| 串口 | pyserial |
| 异步 | asyncio / AnyIO |
| 数值基础 | NumPy |
| 日志 | structlog |
| 测试 | pytest |
| 属性测试 | Hypothesis |
| 静态检查 | Ruff + pyright |
| 覆盖率 | pytest-cov |
| 打包 | wheel |

## 15.2 可选依赖

| 能力 | 技术 |
|---|---|
| Parquet | PyArrow 或 Polars |
| 高级信号计算 | SciPy |
| MCP | Python MCP SDK |
| HTTP | FastAPI |
| MATLAB | MATLAB Engine API |
| GUI | Tauri + Vue |

## 15.3 依赖约束

- CLI 基础安装不得依赖 Node.js；
- MCP 不进入默认依赖；
- SciPy/Parquet 使用 extras；
- 所有外部工具路径可配置；
- 所有 subprocess 有 timeout；
- 禁止 `shell=True`；
- stdout/stderr 必须保存。

---

# 16. 推荐仓库结构

```text
ai-debug-kit/
├── AGENTS.md
├── README.md
├── CHANGELOG.md
├── LICENSE
├── pyproject.toml
├── uv.lock
├── .agents/
│   └── skills/
│       ├── ai-debug-kit-deploy/
│       │   ├── SKILL.md
│       │   ├── references/
│       │   │   ├── installation.md
│       │   │   ├── platform-validation.md
│       │   │   ├── backend-validation.md
│       │   │   ├── agent-installation.md
│       │   │   └── deployment-report.md
│       │   ├── scripts/
│       │   │   ├── detect_environment.py
│       │   │   ├── install_skills.py
│       │   │   ├── validate_skills.py
│       │   │   └── smoke_test.py
│       │   └── assets/
│       │       ├── config-template.toml
│       │       ├── target-template.yaml
│       │       └── deployment-report-template.md
│       └── ai-debug-operations/
│           ├── SKILL.md
│           ├── references/
│           │   ├── operation-lifecycle.md
│           │   ├── cli-command-map.md
│           │   ├── safety-policy.md
│           │   ├── evidence-standard.md
│           │   ├── session-standard.md
│           │   └── domain-handoff.md
│           └── assets/
│               ├── operation-report-template.md
│               └── validation-template.yaml
├── src/
│   └── ai_debug/
│       ├── app/
│       ├── cli/
│       ├── core/
│       ├── deployment/
│       ├── artifacts/
│       ├── memory/
│       ├── registers/
│       ├── variables/
│       ├── telemetry/
│       ├── faults/
│       ├── validation/
│       ├── reports/
│       ├── backends/
│       │   ├── simulator/
│       │   ├── replay/
│       │   ├── mklink/
│       │   ├── openocd/
│       │   └── ti_ccs/
│       ├── platforms/
│       │   ├── generic/
│       │   ├── cortex_m/
│       │   └── c28x/
│       └── gateway/
│           └── mcp/
│               ├── README.md
│               ├── schemas.py
│               ├── tool_mapping.py
│               └── server.py.stub
├── configs/
├── examples/
├── tests/
└── docs/
```

---

# 17. 配置设计

## 17.1 配置优先级

```text
package defaults
→ user config
→ repository config
→ active profile
→ command arguments
```

## 17.2 配置位置

```text
~/.ai-debug/config.toml
<repo>/.ai-debug/config.toml
<repo>/.ai-debug/targets/*.yaml
<repo>/.ai-debug/policies/*.yaml
```

## 17.3 Target Profile

```yaml
schema_version: "1.0"
id: demo.simulator
backend: simulator
platform: generic

address_model:
  address_unit_bits: 8
  endianness: little
  pointer_width_bits: 32

artifact:
  provider: fixture
  path: examples/demo/artifact.json

policy:
  read_only: true
  max_capture_duration_s: 10
```

---

# 18. MCP 扩展预留

## 18.1 v0.1 要求

只建立目录、Schema 和映射说明，不要求 MCP 可运行。

## 18.2 设计原则

```text
CLI ─┐
     ├→ Application Service → Core
MCP ─┘
```

MCP 不得：

- 直接访问 Backend；
- 复制 CLI 业务逻辑；
- 使用不同错误码；
- 绕过 Policy；
- 使用不同 Session。

## 18.3 候选工具

```text
system.doctor
deployment.validate
debug.connect
debug.disconnect
debug.capabilities
artifact.load
artifact.resolve
memory.read
memory.write
register.read
register.write
variable.read
variable.write
telemetry.capture
fault.capture
validation.run
session.export
report.generate
```

---

# 19. 扩展方式

## 19.1 Backend Adapter

目录：

```text
backends/<name>/
├── adapter.py
├── config.py
├── capabilities.py
├── doctor.py
├── errors.py
├── README.md
└── tests/
```

## 19.2 Platform Pack

目录：

```text
platforms/<name>/
├── address_model.py
├── register_model.py
├── fault_decoder.py
├── default_policy.yaml
├── artifact_provider.py
└── tests/
```

## 19.3 Artifact Provider

支持扩展：

- ELF/DWARF；
- TI `.out`；
- COFF；
- MAP；
- BIN/HEX manifest；
- 仿真 fixture。

## 19.4 Telemetry Source

支持扩展：

- Memory Sampling；
- RTT；
- SWO；
- UART；
- EtherCAT；
- CAN；
- Shared Memory；
- Simulator。

## 19.5 Domain Skill

领域能力必须独立注册，例如：

```text
.agents/skills/motor-control-debug/
.agents/skills/ethercat-debug/
.agents/skills/foe-ota-debug/
```

领域 Skill 的输入只能是：

- Session Bundle；
- Capability Profile；
- 标准 CLI；
- 项目代码和项目规则。

领域 Skill 不得要求修改平台 Core 才能表达业务规则。

## 19.6 插件注册

推荐 Python entry points：

```toml
[project.entry-points."ai_debug.backends"]
simulator = "ai_debug.backends.simulator:SimulatorBackend"
replay = "ai_debug.backends.replay:ReplayBackend"

[project.entry-points."ai_debug.platforms"]
generic = "ai_debug.platforms.generic:GenericPlatform"
cortex_m = "ai_debug.platforms.cortex_m:CortexMPlatform"
c28x = "ai_debug.platforms.c28x:C28xPlatform"
```

---

# 20. 实施计划

## Phase 0：工程骨架

任务：

- uv 项目；
- CLI 入口；
- Pydantic Model；
- JSON Envelope；
- Error Code；
- Ruff/pyright/pytest；
- CI；
- AGENTS.md。

验收：

```bash
uv sync
uv run ai-debug --help
uv run ai-debug version --output json
uv run pytest
```

## Phase 1：双 Skill

任务：

- 编写 Skill A；
- 编写 Skill B；
- 编写 frontmatter；
- 编写 reference 路由；
- 编写 Skill 校验脚本；
- 建立触发/不触发 Prompt 集；
- 明确领域边界。

验收：

- 部署请求触发 Skill A；
- 通用调试操作触发 Skill B；
- 业务分析请求不会被 Skill B 冒充处理；
- Skill B 会执行领域交接提示。

## Phase 2：部署与 Smoke Test

任务：

- environment detect；
- setup init；
- doctor；
- Agent detect；
- Skill install；
- active-profile；
- Simulator Smoke Test；
- deployment report。

## Phase 3：Core + Simulator

任务：

- Session；
- Capability；
- Backend Protocol；
- Simulator；
- Memory；
- Register；
- Variable；
- Capture；
- Validation；
- Report。

## Phase 4：Replay

任务：

- Session Bundle；
- Replay Backend；
- 固定 Fixture；
- 旧 Session 兼容；
- Regression Test。

## Phase 5：Artifact

任务：

- ELF/DWARF；
- Symbol；
- Variable Path；
- addr2line；
- MAP 基础解析。

## Phase 6：首个真实 Backend

建议顺序：

```text
Simulator/Replay
→ MKLink 或 OpenOCD
→ TI CCS/C28x
```

## Phase 7：MCP

启动条件：

- CLI 契约稳定；
- JSON Schema 版本化；
- Core API 稳定；
- 双 Skill E2E 通过；
- 安全策略评审完成。

---

# 21. 测试计划

## 21.1 测试层级

```text
Skill Prompt Tests
CLI End-to-End
Backend Contract Tests
Application Integration Tests
Core Unit Tests
Property Tests
Simulator/Replay Tests
Hardware Validation Tests
```

## 21.2 Skill A 测试

### 应触发

- “帮我安装 AI Debug Kit”
- “在这台电脑上验证 Kit”
- “让 Codex 能调用 ai-debug”
- “检查 OpenOCD 后端是否可用”
- “初始化 C28x 平台配置”

### 不应触发

- “分析 iq 振荡”
- “找出 EtherCAT 通信异常根因”
- “修改 PID 参数”
- “解析这段业务日志”

### 行为检查

- 先识别环境；
- 不执行高风险动作；
- 先跑 Simulator；
- 生成 active-profile；
- 区分 detected 与 validated；
- 输出未验证项。

## 21.3 Skill B 测试

### 应触发

- “读取变量 motor.iq”
- “连接目标并导出寄存器快照”
- “捕获 5 秒 Telemetry”
- “对这个地址做 readback”
- “导出本次调试 Session”
- “运行项目提供的验证脚本”

### 不应直接处理

- “分析为什么电流环振荡”
- “判断 Kp 是否太大”
- “自动优化 FOC 参数”
- “推断 FoE 升级失败根因”
- “给出 OTA 架构修复方案”

### 边界检查

Skill B 对业务分析请求必须：

- 不生成假设列表；
- 不输出根因；
- 不自动选业务参数；
- 可收集用户明确指定的数据；
- 明确需要领域 Skill 或项目判据。

## 21.4 Core 单元测试

- Session 状态机；
- Capability；
- Policy；
- 地址单位；
- 字节序；
- 多地址空间；
- JSON Envelope；
- Error Mapping；
- readback；
- Report。

## 21.5 Property Test

- 编码/解码可逆；
- 地址转换无歧义；
- 任意非法长度稳定失败；
- JSON 输出始终符合 Schema；
- Config 合并确定；
- Session 状态转换合法。

## 21.6 Backend Contract Test

每个 Backend 必须通过：

```text
discover
connect
disconnect
capabilities
valid read
invalid read
timeout
cancel
unsupported
error normalization
resource release
```

## 21.7 CLI 测试

- `--help`；
- JSON 输出；
- text 输出；
- 退出码；
- timeout；
- dry-run；
- Unicode 路径；
- Windows 路径；
- Linux 路径；
- 配置缺失；
- Profile 过期；
- Capability 不支持；
- 命令注入防护。

## 21.8 Simulator E2E

### E2E-01：部署

```text
setup init
→ install skills
→ doctor
→ smoke-test
→ active-profile
→ deployment report
```

### E2E-02：通用只读操作

```text
session new
→ connect
→ artifact load
→ capability
→ variable read
→ memory snapshot
→ session export
→ report
```

### E2E-03：受控写入

```text
read original
→ request approval
→ write
→ readback
→ record side effect
→ rollback
→ verify rollback
```

### E2E-04：业务边界

向 Skill B 提交 FOC 根因分析请求，预期：

- 不自动分析；
- 不生成假设；
- 提示缺少领域 Skill；
- 可建议先收集哪些“用户明确指定”的通用证据；
- 不越过平台边界。

## 21.9 Replay 测试

- 结果可重复；
- 不接触真实设备；
- 相同 Session 多次回放一致；
- Schema 不兼容有明确错误；
- warning 与 side effects 保留。

## 21.10 硬件测试

### Level 2

- discover；
- connect；
- identity；
- memory read；
- variable read；
- snapshot。

### Level 3

- halt/resume；
- reset；
- RAM write/readback。

### Level 4

不进入 v0.1 自动测试。

---

# 22. 质量指标

## 22.1 功能

- Skill A 触发准确率 ≥ 90%；
- Skill B 通用操作触发准确率 ≥ 90%；
- Skill B 业务越界率 = 0；
- Simulator E2E 全部通过；
- Replay 结果确定；
- JSON Schema 合法率 100%。

## 22.2 代码

- Core 覆盖率 ≥ 85%；
- 总体覆盖率 ≥ 75%；
- Ruff 无错误；
- pyright 无错误；
- 无未处理异常；
- 无 `shell=True`；
- 无用户路径硬编码。

## 22.3 性能

- `ai-debug --help` < 1 s；
- 本地 doctor < 3 s；
- Simulator Smoke Test < 10 s；
- 取消后 2 s 内退出；
- Session 关闭后无残留锁。

---

# 23. v0.1 验收标准

## 23.1 Skill A

- [ ] 可被 Codex 发现；
- [ ] 可初始化工作区；
- [ ] 可安装两个 Skill；
- [ ] 可运行 doctor；
- [ ] 可运行 Simulator Smoke Test；
- [ ] 可生成 active-profile；
- [ ] 可生成部署报告；
- [ ] 不执行 Level 4 操作。

## 23.2 Skill B

- [ ] 可被 Codex 发现；
- [ ] 可读取 active-profile；
- [ ] 可建立 Session；
- [ ] 可查询 Capability；
- [ ] 可执行通用只读操作；
- [ ] 可执行受控写入和 readback；
- [ ] 可记录副作用；
- [ ] 可导出 Session；
- [ ] 可生成操作报告；
- [ ] 不包含假设管理；
- [ ] 不进行业务根因推断；
- [ ] 不进行业务参数优化。

## 23.3 CLI/Core

- [ ] 安装成功；
- [ ] JSON 输出稳定；
- [ ] Error Code 稳定；
- [ ] Simulator；
- [ ] Replay；
- [ ] Session；
- [ ] Capability；
- [ ] Artifact；
- [ ] Memory；
- [ ] Register；
- [ ] Variable；
- [ ] Telemetry；
- [ ] Fault Snapshot；
- [ ] Validation；
- [ ] Report。

## 23.4 平台化

- [ ] Backend 可插拔；
- [ ] Platform 可插拔；
- [ ] Artifact Provider 可插拔；
- [ ] 显式 address unit；
- [ ] 显式 endianness；
- [ ] 支持多地址空间；
- [ ] Windows/Linux 测试；
- [ ] MCP 预留目录存在；
- [ ] 领域 Skill 可以独立接入。

---

# 24. Codex 执行约束

Codex 开发本项目时必须：

1. 先读 `AGENTS.md`；
2. 先确认当前 Phase；
3. 不提前实现 MCP；
4. 不把业务分析写入 Skill B；
5. 不新增假设管理模块；
6. 不加入 FOC/EtherCAT/OTA 业务 Playbook；
7. 所有新能力提供 Simulator 或 Mock；
8. 所有新 Backend 通过 Contract Test；
9. 所有写操作加入 Policy；
10. 所有命令支持 JSON；
11. 不固定用户路径；
12. 不使用 `shell=True`；
13. 每次提交执行：

```bash
uv run ruff format --check .
uv run ruff check .
uv run pyright
uv run pytest --cov=ai_debug
```

14. 任务汇报包含：
   - 修改内容；
   - 测试结果；
   - Schema 变化；
   - 风险；
   - 未完成项；
   - 是否涉及平台边界。

---

# 25. 推荐 Issue 顺序

1. `chore: initialize uv project`
2. `feat(core): add result envelope and error codes`
3. `feat(cli): add typer command shell`
4. `feat(skill): add kit deployment skill`
5. `feat(skill): add generic debug operations skill`
6. `test(skill): add boundary prompt tests`
7. `feat(deployment): add environment detection`
8. `feat(deployment): add active capability profile`
9. `feat(backend): add simulator backend`
10. `feat(core): add session and capability`
11. `feat(memory): add portable address model`
12. `feat(variable): add typed variable access`
13. `feat(telemetry): add generic capture`
14. `feat(validation): add deterministic validation`
15. `feat(report): add session bundle`
16. `feat(replay): add replay backend`
17. `test(e2e): add deployment flow`
18. `test(e2e): add generic debug operation flow`
19. `docs(mcp): add future gateway mapping`
20. `docs(extension): add backend/platform/domain skill guide`

---

# 26. Definition of Done

每个任务必须满足：

```text
[ ] 功能属于当前 Phase
[ ] 没有越过平台边界
[ ] 没有加入业务根因分析
[ ] 没有加入假设管理
[ ] 有类型定义
[ ] 有标准错误码
[ ] 有 JSON 输出
[ ] 有单元测试
[ ] 有 Simulator/Mock
[ ] 文档已更新
[ ] 无固定用户路径
[ ] 无 shell=True
[ ] 安全策略已覆盖
[ ] Ruff/pyright/pytest 通过
```

---

# 27. 最终架构原则

> **Skill A 证明 Kit 能不能部署和使用。**

> **Skill B 规定 Agent 应当如何安全、规范、可审计地使用 Kit。**

> **Skill B 不负责业务分析，不管理假设，不推断根因，不自动调参。**

> **CLI 负责确定性执行，Core 负责平台能力，Adapter 负责移植。**

> **领域知识必须作为独立 Skill、独立扩展包或项目测试规范存在。**

> **MCP 在 CLI/Core 契约稳定后接入，不成为第一版依赖。**
