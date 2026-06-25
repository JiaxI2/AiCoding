# AI Debug Kit
## AI 介入方式、数据流与 MCU 非干扰约束实施前汇报

> **文档用途**：在交由 Codex 实施前冻结 AI 介入边界、数据通道、实时性约束、安全策略与验证方法  
> **适用项目**：AI Debug Kit for Codex  
> **版本**：v0.1-draft  
> **日期**：2026-06-23  
> **适用范围**：通用嵌入式平台、Cortex-M、C28x、多核 MCU、Simulator/Replay、调试探针与通信采集后端

---

# 1. 执行摘要

AI Debug Kit 不应让大模型进入 MCU 的控制闭环，也不应让 MCU 的正常运行依赖 AI、Codex、CLI、网络或主机连接。

推荐架构为：

```text
AI / Codex：主机侧编排与操作入口
CLI/Core：确定性执行、安全校验、记录与验证
Backend：调试器和通信适配
MCU：仅运行原有业务，以及可选的最小确定性采集 Shim
```

最重要的约束：

1. **AI 只运行在主机侧，不进入 MCU 实时控制路径。**
2. **AI 不直接访问 Probe、串口或 MCU，必须经过 CLI/Core/Policy。**
3. **MCU 不等待主机、不等待 AI、不等待网络响应。**
4. **实时中断中禁止阻塞、动态内存、格式化输出和跨核互斥锁。**
5. **采集缓冲区满时丢弃数据并计数，绝不阻塞控制算法。**
6. **默认只读；控制、复位、写入、烧录按风险等级审批。**
7. **参数修改优先通过受控 Mailbox 在安全点生效，不允许 AI 默认任意写内存。**
8. **所有在线采集能力必须有经过实测的 Impact Profile。**
9. **业务是否正常不由平台 Skill 主观判断，只由用户判据、项目测试或领域扩展判断。**
10. **主机断开、数据丢失、Agent 崩溃时，MCU 必须继续独立运行。**

---

# 2. AI 介入边界

## 2.1 AI 所在位置

AI/Codex 仅存在于主机侧：

```text
Codex
  ↓
Skill
  ↓
ai-debug CLI
  ↓
Application Service
  ↓
Policy / Capability / Session
  ↓
Backend Adapter
  ↓
Probe / UART / RTT / EtherCAT / Simulator
  ↓
MCU
```

AI 不进入：

- 控制 ISR；
- PWM 更新路径；
- ADC 采样链；
- 电流环、速度环或位置环；
- EtherCAT 周期任务；
- Watchdog 服务路径；
- Boot 安全链；
- 实时故障保护路径。

## 2.2 AI 的允许职责

AI 可以：

- 选择和调用已定义 CLI；
- 读取 Capability Profile；
- 创建调试 Session；
- 请求读取、采集、快照和导出；
- 请求执行用户明确指定的控制操作；
- 组织和解释确定性返回结果；
- 根据项目已有测试规则调用 Verifier；
- 生成操作记录和报告。

AI 不可以：

- 绕过 CLI 直接操作调试器；
- 绕过 Policy 直接写目标内存；
- 在缺少 Capability 时假定功能可用；
- 将主机接收时间当作 MCU 采样时间；
- 将数据丢失后插值结果伪装为原始数据；
- 让 MCU 控制任务等待 AI 决策；
- 将大模型输出直接作为 MCU 执行指令；
- 默认执行任意地址写入、Flash 或复位；
- 把自然语言判断作为业务 PASS。

---

# 3. AI 介入等级

平台应将 AI 介入划分为四个等级。

## L0：离线分析

数据来源：

- ELF/OUT/MAP；
- 构建日志；
- 已保存 Session；
- Replay；
- Crash Dump；
- CSV/JSON/Parquet；
- Simulator。

对 MCU 影响：

```text
零影响
```

这是首选模式，也是开发和 CI 的默认模式。

## L1：在线被动观察

操作：

- Probe 运行时只读；
- 读取 Shadow Buffer；
- RTT/SWO/UART 接收；
- EtherCAT/CAN 镜像数据；
- 捕获通用 Telemetry；
- 读取目标状态。

约束：

- 不 halt；
- 不 reset；
- 不写入；
- 不改变控制状态；
- 使用经过验证的采样速率；
- 只能读取声明为安全的区域或通道。

## L2：受控交互

操作：

- halt/resume；
- reset；
- RAM 参数写入；
- breakpoint；
- 控制命令；
- 受控 Mailbox 参数更新。

约束：

- 必须确认 Capability；
- 必须执行 Policy；
- 必须用户批准；
- 必须记录副作用；
- 写后必须 readback；
- 可回滚时必须保存原值；
- 必须在允许的维护窗口或安全状态执行。

## L3：侵入性维护

操作：

- Flash；
- 擦除；
- Boot 配置；
- OTA；
- 启动功率设备；
- 高风险硬件测试。

首版策略：

```text
默认关闭，不允许 Agent 自动执行
```

---

# 4. 推荐的数据流

# 4.1 正向观察数据流

```text
MCU 业务变量 / 寄存器 / 日志
        │
        ▼
Shadow Copy / Trace / Ring Buffer
        │
        ▼
Probe 或通信通道
        │
        ▼
Backend Adapter
        │
        ▼
标准化：
时间戳、序号、类型、地址空间、质量状态
        │
        ▼
Session Buffer / File Store
        │
        ▼
ai-debug CLI JSON
        │
        ▼
Codex / 用户 / 项目验证脚本
```

关键要求：

- 数据先进入确定性 Core，再交给 AI；
- 原始数据和解释结果分开保存；
- MCU 时间戳和主机接收时间分开保存；
- 每条数据带序号，便于检测丢样；
- 采集质量必须标识 `valid/dropped/stale/partial`；
- AI 不直接解析厂商不稳定的终端文本。

## 4.2 反向控制数据流

```text
Codex / 用户请求
        │
        ▼
Skill 操作规范
        │
        ▼
CLI Request
        │
        ▼
Schema Validation
        │
        ▼
Capability Check
        │
        ▼
Policy / Approval / Risk Check
        │
        ▼
Backend Adapter
        │
        ▼
受控命令或 MCU Mailbox
        │
        ▼
MCU 安全点执行
        │
        ▼
ACK / Readback / Status
        │
        ▼
Session Action Record
```

AI 请求不能直接变成 MCU 写操作，中间必须经过：

1. 参数 Schema；
2. Capability；
3. Policy；
4. 审批；
5. 范围检查；
6. 目标身份检查；
7. 执行；
8. readback；
9. 记录。

## 4.3 验证数据流

```text
操作前快照
   +
操作请求
   +
操作后快照 / readback
   +
用户或项目提供的确定性规则
        │
        ▼
Validation Service
        │
        ▼
PASS / FAIL / INCONCLUSIVE
        │
        ▼
报告与 Session Bundle
```

平台不负责建立业务假设，只负责保证：

- 操作是否执行；
- 数据是否完整；
- 状态是否与请求一致；
- 用户规则是否通过；
- 未验证项是否明确。

---

# 5. MCU 侧实现模式

平台应同时支持两种模式。

## 5.1 Agentless 模式

MCU 不增加任何 AI Debug 代码，通过现有调试能力读取：

- 内存；
- 寄存器；
- Trace；
- RTT；
- 已有 UART；
- 已有 EtherCAT/CAN；
- Fault Dump。

优点：

- 不修改固件；
- 快速部署；
- 适合离线和初步调试。

风险：

- 某些 Probe 运行时读内存可能引起总线竞争或短暂停顿；
- 数据一致性可能不足；
- 采样率受 Probe 限制；
- 读取实时控制结构可能产生撕裂数据。

约束：

- Backend 必须声明读取是否侵入；
- 首选读取 Shadow Buffer，而不是直接读取正在更新的控制结构；
- 未验证的运行时内存轮询不得标记为 non-intrusive。

## 5.2 Optional Target Telemetry Shim

在 MCU 中增加一个很小的确定性采集模块，但不包含 AI。

建议命名：

```text
ai_debug_target_shim
```

职责仅包括：

- 固定格式采样；
- 固定大小 Ring Buffer；
- 时间戳；
- 序号；
- 丢样计数；
- Snapshot；
- 可选受控参数 Mailbox；
- 通信后端绑定。

禁止包含：

- Python；
- LLM；
- JSON 解析；
- 动态内存；
- 复杂字符串；
- 业务判断；
- 自动调参；
- 网络依赖；
- 主机同步等待。

---

# 6. MCU 侧采集架构

## 6.1 推荐结构

```text
控制 ISR
  │
  ├─ 原业务计算
  │
  └─ 最小化 Sample Copy
          │
          ▼
      SPSC Ring Buffer
          │
          ▼
低优先级 Debug/Telemetry Task
          │
          ▼
DMA / UART / RTT / Shared RAM / EtherCAT Mirror
```

## 6.2 ISR 中允许的动作

允许：

- 读取已经存在的局部值；
- 写入固定大小结构；
- 更新单生产者写索引；
- 写入单调递增序号；
- 可选记录硬件 Tick。

禁止：

- `printf/sprintf`；
- malloc/free；
- 文件操作；
- 等待锁；
- 等待 DMA；
- 等待主机 ACK；
- 解析命令；
- 跨核阻塞互斥；
- 复杂浮点分析；
- CRC 大块计算；
- 动态信号注册。

## 6.3 建议的最小采样记录

```c
typedef struct
{
    uint32_t sequence;
    uint32_t timestamp_ticks;
    uint16_t signal_id;
    uint16_t flags;
    uint32_t raw_value;
} AiDebugSample32;
```

多变量同步帧可使用：

```c
typedef struct
{
    uint32_t sequence;
    uint32_t timestamp_ticks;
    uint16_t frame_id;
    uint16_t payload_words;
    uint32_t payload[];
} AiDebugFrame;
```

实际位宽和地址单位由 Platform Pack 定义。

## 6.4 固定容量

必须在编译期或初始化期确定：

- Buffer 容量；
- 通道数量；
- 单帧大小；
- 最大采样率；
- 最大传输速率；
- 最大捕获时长。

运行时不得因 Agent 请求无限扩大。

---

# 7. 非阻塞与背压策略

## 7.1 MCU 永不等待主机

当：

- 主机断开；
- Probe 不可用；
- UART 拥堵；
- 网络变慢；
- AI 暂停；
- Codex 崩溃；

MCU 必须继续执行原业务。

## 7.2 Buffer 满处理

允许策略：

```text
drop_newest
drop_oldest
freeze_snapshot
```

默认实时采集推荐：

```text
drop_newest + dropped_count++
```

原因：

- 不改变已有数据；
- 不阻塞生产者；
- 主机可通过序号检测缺口。

## 7.3 主机侧背压

Host/Core 应：

- 限制请求采样率；
- 限制通道数；
- 限制捕获时间；
- 根据 Backend Capability 自动降级；
- 禁止将高于验证值的采样配置下发；
- 接收不及时时停止主机请求，不要求 MCU 等待。

---

# 8. 数据一致性实现

## 8.1 不直接读取活动结构

不推荐：

```text
Probe 随机读取正在被 ISR 更新的结构体
```

推荐：

```text
控制数据
→ 固定时间点复制到 Shadow Snapshot
→ Probe 或通信通道读取 Snapshot
```

## 8.2 双缓冲

```text
Buffer A：MCU 写
Buffer B：Host 读
在安全点原子切换
```

适合：

- 多通道同步快照；
- 低频状态；
- 故障前后数据；
- 大结构读取。

## 8.3 Sequence Lock

```text
seq_begin++
copy data
seq_end = seq_begin
```

Host 读取前后检查序号，发现变化则重读。

## 8.4 时间戳

必须区分：

```text
mcu_timestamp
host_receive_timestamp
```

业务时序分析只能以 MCU 时间戳为主。

---

# 9. 多核 MCU 实现建议

对于 CPU1/CPU2/CM 等多核结构，推荐：

```text
实时控制核
  │
  └─ 只写固定共享 Ring Buffer 或 Shadow Snapshot
          │
          ▼
通信核 / 辅助核
  │
  └─ 负责打包、传输和主机命令接收
```

约束：

- 控制核不处理 JSON、CLI 或网络协议；
- 控制核不等待通信核；
- 使用单向 SPSC Queue；
- 不在控制 ISR 中调用 IPC 阻塞 API；
- 共享内存所有权明确；
- 使用必要的 memory barrier；
- Buffer 溢出只计数；
- 通信核故障不得拖死控制核。

对于 F28388D 一类平台，优先评估：

```text
CPU1：控制
CPU2 或 CM：Telemetry/通信
共享 RAM：固定帧交换
```

该映射必须由 C28x Platform Pack 和项目配置决定，不能写死在 Core。

---

# 10. 写操作的安全实现

## 10.1 默认禁止任意内存写

平台可以保留底层 memory write 能力，但 Skill B 和默认 Policy 不应允许 Agent任意写地址。

推荐提供：

```text
Tunable Parameter Mailbox
```

## 10.2 Mailbox 流程

```text
Host 发送：
parameter_id
new_value
type
range
transaction_id

MCU 后台任务：
检查 ID
检查类型
检查范围
检查状态
暂存新值

MCU 安全点：
原子应用
记录旧值
返回 ACK
```

## 10.3 安全点

示例：

- 控制周期边界；
- PWM 禁止状态；
- 电机停止状态；
- 非 ISR 后台任务；
- 状态机允许的维护状态；
- 双缓冲参数切换点。

## 10.4 写入事务

每次写入必须具备：

```text
prepare
validate
approve
stage
commit
readback
record
rollback（可用时）
```

## 10.5 写入失败

任一环节失败：

- 不部分应用；
- 返回明确错误；
- 保留旧值；
- 记录失败原因；
- 不自动重试高风险写入。

---

# 11. Control 操作约束

## 11.1 Halt

Halt 会改变：

- 实时时序；
- PWM；
- 通信；
- Watchdog；
- 外设状态。

因此：

- 不得在设备带功率运行时默认执行；
- 必须由 Platform/Project Policy 声明安全条件；
- 必须有明确用户批准；
- 必须记录副作用。

## 11.2 Reset

Reset 必须明确：

- core reset；
- system reset；
- peripheral reset；
- warm/cold reset。

禁止用统一的 `reset` 文本模糊处理。

## 11.3 Breakpoint

Breakpoint 属于侵入操作：

- 可能停止控制核；
- 可能触发 Watchdog；
- 可能导致通信超时；
- 可能破坏实时系统外部状态。

默认只允许 Simulator 或安全维护模式。

## 11.4 Flash

Flash 操作不进入 v0.1 自动闭环。

---

# 12. Impact Profile

每个 Backend + Platform + Target 组合必须生成 Impact Profile。

示例：

```json
{
  "schema_version": "1.0",
  "backend": "mklink",
  "platform": "c28x",
  "target": "f28388d-cpu1",
  "mode": "runtime-memory-sampling",
  "validated": true,
  "limits": {
    "max_channels": 8,
    "max_sample_rate_hz": 5000,
    "max_capture_duration_s": 10,
    "max_bus_utilization_percent": 5
  },
  "measured_impact": {
    "cpu_load_delta_percent": 0.7,
    "control_isr_wcet_delta_percent": 1.2,
    "control_jitter_delta_percent": 1.0,
    "missed_deadlines": 0
  },
  "intrusiveness": "low",
  "validated_at": "2026-06-23T00:00:00Z"
}
```

Skill A 负责生成或更新该文件。

Skill B 必须：

- 读取 Impact Profile；
- 拒绝超过限制的请求；
- 在 Profile 缺失时降级到更安全模式；
- 不得把未测量模式标记为无影响。

---

# 13. 推荐的 MCU 资源预算

以下是默认建议值，具体项目可通过 Target Policy 调整。

## 13.1 实时预算

```text
控制 ISR 新增 WCET：≤ 控制周期预算的 2%
控制 ISR 抖动增量：≤ 配置阈值
Telemetry 平均 CPU 增量：≤ 1%
因 Debug 引起的 deadline miss：0
```

## 13.2 总线预算

```text
Debug/Telemetry 总线占用默认 ≤ 5%
```

高于该值必须重新验证。

## 13.3 内存预算

建议：

```text
静态 RAM 占用 ≤ 可用 RAM 的 2%
禁止运行时 Heap 增长
```

项目可以覆盖，但必须记录。

## 13.4 传输预算

根据：

- 通道数；
- 单样本字节数；
- 采样率；
- 帧开销；
- 链路有效吞吐率；

计算：

```text
required_bandwidth =
channels × sample_size × sample_rate × protocol_factor
```

配置验证必须在启动采集前完成。

---

# 14. 断连与异常隔离

## 14.1 主机断连

预期行为：

- MCU 继续运行；
- Debug Buffer 可继续丢弃或停止采样；
- 不进入 Fault；
- 不复位；
- 不等待重连。

## 14.2 Agent 崩溃

Core 必须：

- Session 标记异常；
- 超时释放 Probe/端口锁；
- 不留下 MCU 等待状态；
- 不自动继续写操作。

## 14.3 数据损坏

使用：

- 长度字段；
- Sequence；
- 可选 CRC；
- Schema Version；
- Channel ID；
- Payload Type。

损坏帧：

- 丢弃；
- 计数；
- 报告；
- 不作为有效数据进入业务分析。

## 14.4 命令损坏

MCU 命令通道必须：

- 验证版本；
- 验证命令 ID；
- 验证长度；
- 验证范围；
- 验证当前状态；
- 默认拒绝未知命令。

---

# 15. Skill A 的实施要求

`ai-debug-kit-deploy` 必须验证：

1. Kit 安装；
2. CLI；
3. Skill；
4. Backend；
5. Target；
6. Simulator；
7. 真实设备只读能力；
8. Impact Profile；
9. 断连行为；
10. Session 输出。

Skill A 不只输出：

```text
installed = true
```

必须区分：

```text
detected
configured
tested
validated
unsupported
not_tested
```

---

# 16. Skill B 的实施要求

`ai-debug-operations` 在每次操作前必须：

1. 读取 active-profile；
2. 读取 Impact Profile；
3. 检查 Target/Artifact 身份；
4. 查询实时 Capability；
5. 判断风险等级；
6. 判断请求是否超过已验证限制；
7. 默认选择只读模式；
8. 建立 Session；
9. 执行命令；
10. 验证退出码和 JSON；
11. 保存原始证据；
12. 关闭 Session。

Skill B 不得：

- 直接要求 MCU 增加采样率而不做带宽计算；
- 将 Probe 内存读取视为天然无干扰；
- 在线读取活动控制结构而不声明一致性风险；
- 在用户未批准时 halt/reset/write；
- 对业务波形给出根因判断；
- 自行创建控制参数修改方案。

---

# 17. 推荐模块划分

```text
src/ai_debug/
├── app/
│   ├── deployment_service.py
│   ├── operation_service.py
│   └── validation_service.py
├── core/
│   ├── capability.py
│   ├── impact_profile.py
│   ├── policy.py
│   ├── session.py
│   └── result.py
├── telemetry/
│   ├── schema.py
│   ├── bandwidth.py
│   ├── quality.py
│   └── storage.py
├── actions/
│   ├── risk.py
│   ├── approval.py
│   ├── transaction.py
│   └── rollback.py
├── backends/
├── platforms/
└── target_shim/
    ├── include/
    ├── src/
    ├── ports/
    └── examples/
```

---

# 18. Target Shim API 建议

```c
typedef struct
{
    uint32_t buffer_address;
    uint32_t buffer_size;
    uint32_t max_sample_rate_hz;
    uint16_t max_channels;
    uint16_t flags;
} AiDebugConfig;

typedef enum
{
    AI_DEBUG_OK = 0,
    AI_DEBUG_ERR_CONFIG,
    AI_DEBUG_ERR_FULL,
    AI_DEBUG_ERR_RANGE,
    AI_DEBUG_ERR_STATE
} AiDebugStatus;

AiDebugStatus AiDebug_Init(const AiDebugConfig *config);

AiDebugStatus AiDebug_PushSampleU32(
    uint16_t signal_id,
    uint32_t timestamp_ticks,
    uint32_t value);

AiDebugStatus AiDebug_PublishSnapshot(
    uint16_t snapshot_id,
    const void *data,
    uint16_t octet_length);

void AiDebug_Service(void);
```

ISR 版本必须：

- 内联或短函数；
- 固定执行路径；
- 不阻塞；
- 不调用传输；
- Buffer 满立即返回。

编译关闭：

```c
#if AI_DEBUG_ENABLE == 0
#define AiDebug_PushSampleU32(...) (AI_DEBUG_OK)
#endif
```

生产固件可以完全裁剪。

---

# 19. 测试与验证计划

## 19.1 Baseline 对比

构建两版：

```text
A：AI Debug Shim 关闭
B：AI Debug Shim 开启
```

在相同输入、相同编译优化、相同负载下比较：

- 控制 ISR WCET；
- 控制 ISR Jitter；
- CPU Load；
- RAM/Flash；
- 总线占用；
- PWM/ADC 时序；
- 通信周期；
- Deadline Miss；
- 业务输出偏差。

## 19.2 测量方法

平台适配器可使用：

- CPU cycle counter；
- GPIO Toggle + 示波器/逻辑分析仪；
- Trace；
- IDE Profiler；
- CPU Timer；
- 周期计数统计；
- 通信帧时间戳。

必须保存测量方法和环境，不允许只写“影响很小”。

## 19.3 故障注入

测试：

- 主机突然断开；
- Probe 拔出；
- Ring Buffer 满；
- 通信阻塞；
- 错误长度；
- 错误 CRC；
- 未知命令；
- 超量采样请求；
- 过期 Profile；
- Agent 进程被杀；
- MCU Reset；
- 多核通信核故障。

## 19.4 非干扰验收

必须同时满足：

```text
[ ] 无新增 deadline miss
[ ] 控制 ISR WCET 增量不超过配置阈值
[ ] 控制 ISR jitter 增量不超过配置阈值
[ ] MCU 不等待主机
[ ] Buffer 满不会阻塞
[ ] 主机断开不影响业务运行
[ ] 关闭 Shim 后编译可完全移除
[ ] 未批准写入被拒绝
[ ] 写操作可 readback
[ ] 数据丢失可被检测
[ ] Profile 中记录实际测量结果
```

## 19.5 Skill 边界测试

### Skill A

- 能否生成 Impact Profile；
- 能否区分检测与验证；
- 能否拒绝高风险自动测试；
- 能否在断连后恢复环境。

### Skill B

- 是否在采集前检查 Profile；
- 是否拒绝超额采样；
- 是否默认只读；
- 是否记录副作用；
- 是否避免业务根因判断；
- 是否保存原始证据。

---

# 20. Codex 实施顺序

Codex 不应一开始就接真实 MCU。

推荐顺序：

## Phase 1：Host-only

- Session；
- Capability；
- Policy；
- CLI；
- Simulator；
- Replay；
- Impact Profile Schema。

## Phase 2：Target Shim Simulator

- Ring Buffer；
- Drop Counter；
- Sequence；
- Timestamp；
- Mailbox 模拟；
- 断连模拟。

## Phase 3：真实平台只读

- Probe/通信发现；
- Shadow Buffer；
- 只读采集；
- Impact Measurement；
- Level 2 验证。

## Phase 4：受控写入

- Mailbox；
- Range Check；
- Safe Point；
- Readback；
- Rollback；
- Level 3 验证。

## Phase 5：领域扩展

只有平台基础稳定后，才接入独立领域 Skill。

---

# 21. Codex 代码实现约束

Codex 必须遵循：

1. AI 只在主机侧；
2. Core 不包含业务判断；
3. Target Shim 不包含动态内存；
4. ISR 不阻塞；
5. ISR 不传输；
6. Buffer 满必须立即返回；
7. 所有在线能力有 Capability；
8. 所有采集限制来自 Impact Profile；
9. 所有写操作通过 Policy；
10. 所有写操作 readback；
11. 所有外部命令有 timeout；
12. 禁止 `shell=True`；
13. 真实设备默认只读；
14. 无真实硬件时使用 Simulator/Replay；
15. 每个 Backend 通过 Contract Test；
16. 每个 Platform Pack 提供地址模型；
17. 每次开发完成必须输出资源影响报告。

---

# 22. 实施前必须冻结的架构决策

建议在 Codex 开始编码前确认以下决策：

```text
ADR-001：AI 仅运行在 Host，不进入 MCU
ADR-002：CLI/Core 是唯一确定性执行入口
ADR-003：默认只读，写入通过 Policy 和审批
ADR-004：在线采集必须有 Impact Profile
ADR-005：MCU 采集使用非阻塞固定 Ring Buffer
ADR-006：活动业务数据通过 Shadow/Double Buffer 暴露
ADR-007：主机断连不得影响 MCU
ADR-008：参数写入优先使用 Mailbox + Safe Point
ADR-009：Skill B 不做业务分析
ADR-010：MCP 后接入，不绕过 Application Service
```

---

# 23. 最终建议

第一版应优先实现三个闭环：

## 闭环一：零硬件部署

```text
Codex
→ Skill A
→ 安装 CLI
→ Simulator
→ Smoke Test
→ Impact Profile
→ Deployment Report
```

## 闭环二：通用只读操作

```text
Codex
→ Skill B
→ Session
→ Capability
→ Artifact
→ Read/Snapshot/Capture
→ Record
→ Report
```

## 闭环三：MCU 非干扰验证

```text
Baseline Firmware
vs
Debug-enabled Firmware
→ WCET/Jitter/CPU/RAM/Bus 对比
→ Disconnect/Overflow Fault Injection
→ Impact Profile
→ PASS/FAIL
```

在这三个闭环通过前，不建议实现：

- 自动 Flash；
- 自动复位；
- 自动在线调参；
- 真实电机运行；
- MCP；
- 领域故障分析。

---

# 24. 核心结论

> **AI 参与的是主机侧调试编排，不是 MCU 实时控制。**

> **MCU 侧只提供可裁剪、固定资源、非阻塞、确定性的观测与受控命令通道。**

> **数据采集宁可丢样，也不能阻塞控制任务。**

> **任何“无影响”必须通过 WCET、Jitter、CPU、RAM、总线和断连测试证明，而不能凭经验声明。**

> **平台 Skill 只保证调试动作正确、安全、可记录，不负责具体业务根因分析。**
