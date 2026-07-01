# 设计说明

## 模块定位

本模块定位为“通用 PID 控制单元”，不是裸 PID 内核。它保留 `pid()` 单入口，但在单次调用内部完成目标整形、误差调节、微分滤波、输出保护和积分抗饱和。

这些功能属于控制器通用工具，不属于上层应用语义。模块不绑定具体被控对象、硬件外设或执行器。

## 单周期控制链

```text
setpoint/input
  -> 目标限幅
  -> 目标变化率限制
  -> feedback 可选限幅
  -> error = setpoint - feedback
  -> error 可选限幅
  -> P/I/D + feedforward
  -> D 项一阶滤波
  -> output 限幅
  -> output deadband
  -> back-calculation 抗积分饱和
  -> state 更新
```

`outputLimit` 与抗积分饱和保持在同一个控制单元内，是为了让积分器知道输出是否已经被限幅；如果输出限幅完全放到外部，积分器无法判断实际饱和状态。

## API 收敛

v1.6 对外只保留：

```c
float pid(Pid *controller);
```

输入从 `controller->input` 读取，输出写入 `controller->state.output` 并作为返回值返回。配置、输入和状态集中在一个对象里，适合多实例管理。

## 内置可选工具

| 工具 | 配置 | 说明 |
|---|---|---|
| 目标限幅 | `setpointLimit` | 限制目标绝对范围 |
| 目标斜率限制 | `setpointRateEnable / setpointRate` | 限制目标每秒最大变化量 |
| 反馈限幅 | `feedbackLimit` | 裁剪异常反馈输入 |
| 误差限幅 | `errorLimit` | 限制误差参与 P/I/D 的范围 |
| 微分滤波 | `derivativeFilterCoef` | 对 D 项微分做一阶低通滤波 |
| 积分限幅 | `integralLimit` | 限制 I 分量大小 |
| 输出限幅 | `outputLimit` | 限制最终输出范围 |
| 输出死区 | `deadband` | 小输出直接置 0 |
| 抗积分饱和 | `antiWindupGain` | 输出饱和时修正积分累积 |

以上工具默认关闭或不改变裸 PID 行为。需要时由用户显式配置。

## 去耦与去敏

源码只保留通用 PID 控制器语义，不出现第三方项目名，不绑定任何上层业务，不包含硬件接口、通信接口或执行器接口。

## 控制频率

用户配置 `controlFreq`，单位 Hz。内部计算离散周期：

```text
Ts = 1 / controlFreq
```

积分和微分仍使用 `Ts` 完成计算：

```text
I(k+1) = I(k) + Ki * [e(k) + Kaw * (u_sat(k) - u_raw(k))] * Ts
D(k) = [e(k) - e(k-1)] / Ts
```

## 单位和标幺

控制器不固定物理单位。只要求：

- `setpoint`、`feedback`、`error` 使用同一被控量单位；标幺时为 `pu`。
- `output`、`feedforward`、P/I/D 分量使用同一输出单位；标幺时为 `pu`。
- `controlFreq` 使用 Hz，内部周期 `Ts` 使用 s。
- `setpointRate` 使用被控量单位/s；标幺时为 `pu/s`。
- `antiWindupGain` 用于把输出饱和误差折算到误差域，单位为被控量单位/输出单位；标幺时为 `pu/pu`。

## 积分抗饱和

使用 back-calculation。参考模型中的结构可以写成：

```text
uc = 限幅前控制输出
u  = Saturation(uc)

积分输入 = e - Kaw * (uc - u)
         = e + Kaw * (u - uc)
```

离散实现为：

```text
I(k+1) = I(k) + Ki * [e(k) + Kaw * (u_sat(k) - u_raw(k))] * Ts
```

其中：

- `u_raw` 对应参考模型里的 `uc`；
- `u_sat` 对应参考模型里的 `u`；
- `Kaw` 对应参考模型里 `uc-u` 反馈支路上的增益。

该逻辑只在 `ki != 0.0f` 时运行，因此 PI 和 PID 都有效，P 和 PD 自动跳过。

注意：参考模型是连续 Simulink 模型，当前 C 模块是离散控制实现。两者拓扑一致，但若要做逐采样数值一致，需要先统一采样周期、离散积分器和求解器。


## 统一初始化

`pid_init()` 用于把 PID 对象置于确定初始状态：清零输入、状态和可选工具，设置默认控制频率、默认微分滤波系数和抗饱和增益。初始化函数不替代工程参数配置，实际 kp、ki、kd、controlFreq 和 outputLimit 仍需要按控制对象覆盖。
