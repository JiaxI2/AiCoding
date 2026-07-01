# FOC 控制模板设计说明

版本：v0.6 GBK

## 模块边界

本 kit 把 FOC 拆成四层：

1. `foc_math`：纯坐标变换数学；
2. `foc_svpwm`：调制矢量到 duty；
3. `foc_angle`：闭环角度、开环角度、固定角度和零位偏置；
4. `foc`：把电角度、dq 指令和可选电流反馈串成一次 FOC 更新。

`foc.c` 可以使用前三个模块，但前三个模块不依赖 `foc.c`。

## 开环和闭环设计

开环/闭环分成两个独立问题：

```text
角度是否闭环：由 foc_angle 决定。
电流是否闭环：由 foc 控制模式决定。
```

这样可以避免把启动状态机、传感器读取和电流控制耦合在一个函数里。

| 组合 | 角度模块 | FOC 主单元 | 用途 |
|---|---|---|---|
| 开环电压 | `FOC_ANGLE_MODE_OPEN_LOOP` | `FOC_CONTROL_MODE_OPEN_VOLTAGE` | 对齐、开环启动、扫频 |
| 开环角度 + 电流闭环 | `FOC_ANGLE_MODE_OPEN_LOOP` | `FOC_CONTROL_MODE_CLOSED_CURRENT` | 无可靠角度反馈时驱动电流矢量 |
| 闭环电流 | `FOC_ANGLE_MODE_SENSOR` | `FOC_CONTROL_MODE_CLOSED_CURRENT` | 正常电流闭环运行 |
| 固定矢量 | `FOC_ANGLE_MODE_FIXED` | `OPEN_VOLTAGE` 或 `CLOSED_CURRENT` | 角度对齐、锁定、测试 |

## 单周期 FOC 控制链

`foc()` 的控制思路是：先把测量量变换到转子同步旋转坐标系，在 dq 坐标系内生成电压，再变回静止坐标系交给 SVPWM。该函数不做硬件采样和 PWM 写入，只输出 duty。

```text
phaseCurrent abc
    -> foc_clarke()
    -> current alpha-beta
    -> foc_park(electricalAngle)
    -> current dq
    -> 开环电压模式：直接使用 voltageFeedforward
       闭环电流模式：dq 电流 PI + voltageFeedforward
    -> voltage dq
    -> foc_inv_park(electricalAngle)
    -> voltage alpha-beta
    -> 归一化 modulation alpha-beta
    -> foc_svpwm()
    -> dutyA / dutyB / dutyC
```

### 开环电压模式

开环电压模式下，`voltageFeedforward` 被当作 dq 电压指令使用。该模式不需要电流反馈，适合启动前对齐、开环旋转、扫频测试，以及外部 PID kit 已经给出 `vd/vq` 的场景。

### 闭环电流模式

闭环电流模式下，`currentSetpoint - currentDq` 得到 dq 电流误差，内部轻量 PI 生成 dq 电压。若电压矢量超过 `maxVoltage`，会按比例缩放，并对积分项做衰减，避免饱和时积分继续增大。

### 开环角度 + 闭环电流

这是开环启动时常见的折中组合：电角度按给定速度积分生成，但电流幅值仍由三相电流反馈闭环控制。该 kit 不自动判断切闭环条件，只提供可组合模块；切换条件由上层根据速度、反电动势、观测器收敛度或编码器状态决定。

### 单位和标幺

源码字段保留真实物理单位说明。若系统使用标幺，电流、电压、调制量和增益由上层按统一基值换算，字段注释不逐项重复“标幺时为 pu”。

## 子模块实现依据

### foc_angle_update()

闭环角度模式核心关系为：

```text
electricalAngle = direction * polePairs * mechanicalAngle + offsetRad
electricalSpeed = direction * polePairs * mechanicalSpeed
electricalAngleComp = electricalAngle + electricalSpeed * phaseCompTime
```

开环角度模式核心关系为：

```text
period = 1 / controlFreq
electricalSpeed = direction * openLoopElectricalSpeedRadPerSec
openLoopElectricalAngle = openLoopElectricalAngle + electricalSpeed * period
electricalAngleComp = openLoopElectricalAngle + offsetRad + electricalSpeed * phaseCompTime
```

`offsetRad` 来自零位校准，用于让 d/q 坐标系和实际转子磁链方向对齐；`phaseCompTime` 是等效延时补偿，用于高速时降低采样、计算和 PWM 更新延迟造成的角度滞后。函数最终把角度包装到 `0~2pi` 并生成 `sin/cos`，供 Park 和反 Park 使用。

### foc_svpwm()

该函数采用零序注入形式生成中心对齐三相 duty：

```text
phase = inverse_clarke(mod_alpha_beta)
commonMode = -0.5 * (max(phaseA, phaseB, phaseC) + min(phaseA, phaseB, phaseC))
dutyX = 0.5 + phaseX + commonMode
```

实现上先检查 alpha-beta 调制矢量幅值，超出 `maxModulation` 时可按比例缩放。零序注入和扇区时间计算属于等价的 SVPWM 实现思路，当前写法更适合 common 模块复用和快速审查。输出 duty 范围为 `0~1`，不直接写 PWM 硬件。

## 与 PID kit 的关系

FOC kit 不直接 include PID kit，避免耦合。

常用组合方式：

- 外部 PID 计算 `vd/vq`，FOC 使用开环电压模式；
- 外部位置/速度 PID 给 `Id/Iq`，FOC 使用闭环电流模式；
- 开环角度模块先启动，达到上层切换条件后改用闭环角度。

## 不加入的功能

以下功能不放入 common FOC 模块：

- ADC offset 自动标定；
- 编码器校准状态机；
- PWM 寄存器写入；
- 过流、欠压、堵转等故障策略；
- 位置环、速度环调度；
- 多电机管理；
- 无感观测器和开环到闭环自动切换状态机。

这些需要由板级平台层或上层应用组合。


## 统一初始化

`foc_init()`、`foc_angle_init()` 和 `foc_svpwm_init()` 负责把对象置于确定初始状态：清零输入、清零状态、写入安全默认值。初始化函数不替代工程参数配置，真实控制频率、极对数、电流环增益、母线电压和零位偏置仍由上层配置。

## 零电流偏置

三相电流进入 Clarke/Park 之前先扣除 `currentOffset`：

```text
phaseCurrentCorrected = phaseCurrent - currentOffset
```

`foc_current_offset_accumulate()` 使用递推平均估计零电流偏置，不分配内存，不依赖 ADC。该设计借鉴成熟工程中“未驱动状态下更新电流偏置”的控制思路，但保留 common 层去耦：采样条件、PWM 关闭和样本来源都由上层保证。

## 开环/闭环验证边界

no-compile 行为镜像已验证开环角度积分、开环电压输出、闭环电流 PI 输出、调制限幅和零电流偏置扣除。该验证只能证明控制链数学关系一致，不能替代硬件实测。硬件验证仍需覆盖电流采样极性、电角度零位、dq 方向、PWM 极性和过流保护。


## v0.7 外环级联控制层

新增 `foc_motion` 模块，用于补齐位置环、速度环、输入滤波、前馈和齿槽补偿。设计上它位于 `foc()` 电流环之前，只输出 `FocDq currentDq`，不参与 ADC、PWM 或传感器读取。

### 控制链

```text
positionSetpoint / velocitySetpoint / currentFeedforward
  -> input mode
  -> position loop
  -> velocity loop
  -> acceleration feedforward
  -> anti-cogging feedforward
  -> current limit / anti-windup
  -> dq current setpoint
```

### 输入滤波

二阶位置输入滤波使用内部位置和速度 setpoint 跟踪外部输入，计算加速度：

```text
accel = inputFilterKp * positionError + inputFilterKi * velocityError
velocitySetpoint += Ts * accel
positionSetpoint += Ts * velocitySetpoint
qCurrentFeedforward += accel * inertiaFeedforwardGain
```

### 位置环和速度环

位置环只生成速度指令：

```text
velocityCommand = velocitySetpoint + posGain * positionError
```

速度环生成 q 轴电流：

```text
qCurrent = qCurrentFeedforward + velGain * velocityError + velocityIntegratorCurrent
```

饱和时衰减积分，未饱和时积分速度误差，并按 `velIntegratorLimit` 限幅。

### anti-cogging

anti-cogging 表由上层标定和保存，本模块只按当前位置查表并叠加 q 轴电流前馈。这样可以把标定流程和实时控制解耦。

### 与 FOC 主模块的关系

`FOC_CONTROL_MODE_MOTION_CURRENT` 或 `enableMotionControl = true` 时，`foc()` 会先调用 `foc_motion_update()`，把外环输出写入 `input.currentSetpoint`，再进入闭环电流 PI、反 Park 和 SVPWM。
