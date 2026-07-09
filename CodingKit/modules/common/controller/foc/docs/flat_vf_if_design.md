# Flat VF/IF FOC Design

## 目标

本设计把 `CodingKit/modules/common/controller/foc` 的主 API 收敛为一层 `Foc` 字段，避免深层结构访问成为用户主路径。FOC 只保留两种核心模式：

```text
FOC_MODE_VF = dq 电压链路
FOC_MODE_IF = dq 电流链路
```

角度来源只保留：

```text
FOC_ANGLE_SENSOR = 上层提供真实电角度
FOC_ANGLE_OPEN_LOOP = foc_loop() 内部按频率积分电角度
```

## 模式组合

```text
IF + SENSOR
  正常闭环 FOC。位置环和速度环可选，最终都生成 cmd_iq。

IF + OPEN_LOOP
  I/f 开环启动。角度由内部积分，cmd_iq 是启动电流。

VF + OPEN_LOOP
  V/f 开环。角度由内部积分，电压来自 cmd_vd/cmd_vq 或 V/f 自动电压。
```

## 主流程

`foc_loop()` 的顺序固定为：

```text
1. 检查 controller、control_freq、vbus。
2. OPEN_LOOP 时积分 theta_e，并更新 omega_e。
3. 扣除 ia/ib/ic offset。
4. Clarke 得到 real_ialpha / real_ibeta。
5. Park 得到 real_id / real_iq。
6. VF 模式生成 out_vd / out_vq。
7. IF 模式执行 position -> velocity -> current PID。
8. 对 out_vd / out_vq 做 max_voltage 矢量限幅。
9. inverse Park 得到 out_valpha / out_vbeta。
10. SVPWM 得到 duty_a / duty_b / duty_c。
11. 写 saturated / valid。
```

## PID 复用

FOC 不再实现独立电流 PI。四个环路全部复用 `common/controller/pid`：

```text
pid_pos: cmd_pos - pos -> cmd_vel
pid_vel: cmd_vel - vel -> cmd_iq
pid_id : cmd_id - real_id -> out_vd
pid_iq : cmd_iq - real_iq -> out_vq
```

FOC 内部 wrapper 仅把 error-only 调用转换为现有 PID API 的 setpoint/feedback/feedforward 输入格式，不修改 PID 模块。

## 架构边界

当前目录只表达 v1.0-flat VF/IF 主架构，不保留历史兼容层、旧入口或旧控制模式映射。需要查看第一版实现时，应通过 Git history / tag 回溯。

## 非目标

本模块不处理 ADC 采样调度、PWM shadow 更新、编码器同步、状态机、故障保护、参数整定、齿槽补偿、观测器或硬件驱动对象。这些都属于上层应用或平台适配层。
