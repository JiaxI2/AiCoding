# AiCoding FOC Controller Template v0.7

这是一个去耦的 C99 FOC 通用模板。源码面向嵌入式实时控制，C 源码和示例使用 GBK 编码，文档和测试脚本使用 UTF-8。

本版在 v0.6 的基础上增加 `foc_motion` 外环控制层，用于把位置、速度、输入滤波、前馈和齿槽补偿转换为 dq 电流指令，再送入 FOC 电流环。

## 模块边界

角度模块仍保留 `FOC_ANGLE_MODE_OPEN_LOOP`，主 FOC 仍保留 `FOC_CONTROL_MODE_CLOSED_CURRENT`；`foc_angle_update()` 负责角度更新，外环模块只负责生成 dq 电流指令。


```text
foc_math   = Clarke / Park / 反 Park / 反 Clarke
foc_svpwm  = SVPWM / duty 生成
foc_angle  = 闭环角度、开环角度、固定角度、零位偏置
foc_motion = 位置环、速度环、输入滤波、前馈、anti-cogging
foc        = 电流环、dq 电压生成、反 Park、SVPWM 串联
```

不包含：ADC 读取、PWM 写寄存器、编码器读取、Hall 读取、驱动对象、电机对象、故障状态机、通信协议。

## 推荐控制链

```text
上层命令
  -> foc_motion 输入整形
  -> 位置环
  -> 速度环
  -> 电流前馈 / 加速度前馈 / anti-cogging
  -> dq 电流指令
  -> foc 闭环电流环
  -> 反 Park
  -> SVPWM
  -> dutyA / dutyB / dutyC
```

## 新增外环控制模式

```c
FOC_MOTION_CONTROL_CURRENT
FOC_MOTION_CONTROL_VELOCITY
FOC_MOTION_CONTROL_POSITION
```

含义：

```text
CURRENT  = 直接输出 q 轴电流，可叠加限幅和 anti-cogging。
VELOCITY = 速度误差经 P/I 输出 q 轴电流。
POSITION = 位置误差先生成速度指令，再进入速度环。
```

## 新增输入模式

```c
FOC_MOTION_INPUT_PASSTHROUGH
FOC_MOTION_INPUT_POS_FILTER
FOC_MOTION_INPUT_VEL_RAMP
FOC_MOTION_INPUT_CURRENT_RAMP
```

`FOC_MOTION_INPUT_POS_FILTER` 是二阶位置输入滤波，用于让位置目标变得连续，并生成速度和加速度前馈。

## anti-cogging 和前馈

`foc_motion` 支持外部传入齿槽补偿表：

```c
controller.motion.config.enableAntiCogging = true;
controller.motion.config.antiCoggingTable = table;
controller.motion.config.antiCoggingTableLength = tableLength;
```

表项单位是 A，表示该位置需要叠加到 q 轴的电流前馈。表内存由上层提供，模板不分配内存，也不做标定流程。

前馈包括：

```text
qCurrentFeedforward      = q 轴电流前馈
 dCurrentFeedforward     = d 轴电流前馈
inertiaFeedforwardGain   = 输入滤波或速度斜坡产生的加速度前馈系数
voltageFeedforward       = foc 电流环内的 dq 电压前馈
```

## 和 foc() 的集成

启用外环级联：

```c
Foc controller;
foc_init(&controller);

controller.config.controlMode = FOC_CONTROL_MODE_MOTION_CURRENT;
controller.config.enableMotionControl = true;

controller.motion.config.controlMode = FOC_MOTION_CONTROL_POSITION;
controller.motion.config.inputMode = FOC_MOTION_INPUT_POS_FILTER;

controller.motion.input.positionSetpoint = 1.0f;
controller.motion.input.positionFeedback = 0.2f;
controller.motion.input.velocityFeedback = 0.0f;

foc(&controller);
```

`foc()` 会先调用 `foc_motion_update()` 生成 `currentSetpoint`，再进入闭环电流环。

## 验证范围

本包仍是 no-compile 验证，覆盖：

```text
- 开环角度积分
- 开环电压 duty 输出
- 闭环电流 duty 输出
- 零电流偏置扣除
- 位置环 -> 速度环 -> 电流指令
- 二阶位置输入滤波
- 速度斜坡加速度前馈
- anti-cogging 电流前馈
- 外环级联进入 FOC 电流环
```

额外做过一次 CMake/GCC smoke compile，用于检查 C99 语法和库目标能否构建。

不包含真实硬件验证，不读取外设。
