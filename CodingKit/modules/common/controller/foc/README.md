# AiCoding FOC Controller Template v1.0-flat

这是一个扁平化 C99 FOC 通用模板。源码主路径只保留 VF / IF 两种核心模式，以及 SENSOR / OPEN_LOOP 两种角度来源。三环 PID 直接复用 `common/controller/pid`，FOC 内部不再重复实现 PID。

本模块不绑定 ADC / PWM / 编码器 / Hall / 观测器 / 状态机 / 故障保护 / 通信协议。上层负责提供电流、母线电压、真实角度或开环频率，并读取 `duty_a` / `duty_b` / `duty_c` 写入硬件。

本目录不保留历史兼容层，旧版请通过 Git history / tag 查看。

## 核心模式

```text
FOC_MODE_VF
  dq 电压链路。最终使用 cmd_vd / cmd_vq，或用 V/f 参数自动生成 q 轴电压。

FOC_MODE_IF
  dq 电流链路。最终使用 cmd_id / cmd_iq，经 pid_id / pid_iq 输出 out_vd / out_vq。
```

```text
FOC_ANGLE_SENSOR
  theta_e 由上层真实位置、编码器、Hall 插值或观测器提供。

FOC_ANGLE_OPEN_LOOP
  theta_e 由 foc_loop() 根据 open_loop_freq_hz、dir 和 control_freq 积分。
```

典型组合：

```text
IF + SENSOR     = 正常三环闭环 FOC
IF + OPEN_LOOP  = I/f 开环启动
VF + OPEN_LOOP  = V/f 开环
```

## 扁平 API

`Foc` 结构体现在是用户可直接访问的一层字段。核心扭矩电流命令是 `cmd_iq`；位置环和速度环只是 `cmd_iq` 的上游生成器。

```c
Foc foc;

foc_init(&foc);

foc.mode = FOC_MODE_IF;
foc.angle_mode = FOC_ANGLE_SENSOR;
foc.vbus = 24.0f;
foc.ia = ia;
foc.ib = ib;
foc.ic = ic;
foc.theta_e = theta;
foc.cmd_iq = 2.0f;

foc_loop(&foc);

pwm_a = foc.duty_a;
pwm_b = foc.duty_b;
pwm_c = foc.duty_c;
```

唯一执行入口是 `foc_loop(Foc *controller)`。

## VF 执行路径

```text
foc_loop
  -> 可选 OPEN_LOOP 角度积分
  -> offset 扣除
  -> Clarke / Park 观测真实电流
  -> out_vd = cmd_vd
  -> out_vq = cmd_vq 或 cmd_vq + sign(dir) * V/f 电压
  -> max_voltage 矢量限幅
  -> inverse Park
  -> SVPWM
  -> duty_a / duty_b / duty_c
```

当 `vf_gain_v_per_hz` 或 `vf_boost_v` 非零时，V/f 电压为：

```text
vf_v = vf_boost_v + vf_gain_v_per_hz * abs(open_loop_freq_hz)
vf_v 限幅到 [vf_min_v, vf_max_v]
out_vq = cmd_vq + sign(dir) * vf_v
```

## IF 执行路径

```text
foc_loop
  -> SENSOR 使用上层 theta_e，OPEN_LOOP 由内部积分 theta_e
  -> offset 扣除
  -> Clarke / Park 得到 real_id / real_iq
  -> 可选 pid_pos: cmd_pos - pos -> cmd_vel
  -> 可选 pid_vel: cmd_vel - vel -> cmd_iq
  -> 可选 pid_id: cmd_id - real_id -> out_vd，否则 out_vd = cmd_vd
  -> 可选 pid_iq: cmd_iq - real_iq -> out_vq，否则 out_vq = cmd_vq
  -> max_voltage 矢量限幅
  -> inverse Park
  -> SVPWM
  -> duty_a / duty_b / duty_c
```

## PID 复用

FOC target 编译 `../pid/src/pid.c`，并通过 `#include "pid.h"` 使用现有 PID API。FOC 内部仅提供 error-only wrapper：

```c
controller->input.setpoint = error;
controller->input.feedback = 0.0f;
controller->input.feedforward = 0.0f;
return pid(controller);
```

四个 PID 字段分别是：

```text
pid_pos -> cmd_vel
pid_vel -> cmd_iq
pid_id  -> out_vd
pid_iq  -> out_vq
```

## 验证

静态验证入口：

```bash
python CodingKit/modules/common/controller/foc/tests/test_foc_flat_vf_if.py
```

建议同时运行 PID 模块验证和 FOC CMake smoke build：

```bash
python CodingKit/modules/common/controller/pid/tests/run_no_compile_validation.py
cmake -S CodingKit/modules/common/controller/foc -B CodingKit/modules/common/controller/foc/build
cmake --build CodingKit/modules/common/controller/foc/build
```

真实电机 bring-up 仍需覆盖相序、offset、电流方向、编码器方向/零位、开环 Vq、闭环 Id/Iq、速度闭环和位置闭环。本模块不声明硬件参数已调好。
