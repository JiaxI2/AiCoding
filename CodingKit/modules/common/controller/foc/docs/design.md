# FOC 控制模板设计说明

版本：v1.0-flat

## 模块边界

本目录只保留扁平 VF/IF FOC 主架构。当前源码由三类可复用单元组成：

1. `foc_math`：纯坐标变换数学；
2. `foc_svpwm`：调制矢量到 duty；
3. `foc`：把相电流、角度、dq 指令、可选 PID 环路和 SVPWM 串成一次 FOC 更新。

`foc_loop(Foc *controller)` 是唯一执行入口。ADC 采样、PWM 写入、传感器读取、状态机、故障保护、观测器和硬件驱动对象都由上层负责。

## 模式和角度来源

| 组合 | 模式 | 角度来源 | 用途 |
|---|---|---|---|
| V/f 开环 | `FOC_MODE_VF` | `FOC_ANGLE_OPEN_LOOP` | 对齐、开环启动、扫频 |
| I/f 开环启动 | `FOC_MODE_IF` | `FOC_ANGLE_OPEN_LOOP` | 无可靠角度反馈时驱动电流矢量 |
| 闭环电流 / 三环控制 | `FOC_MODE_IF` | `FOC_ANGLE_SENSOR` | 正常电流、速度、位置闭环运行 |

## 单周期控制链

```text
phase current abc
    -> offset correction
    -> Clarke
    -> Park(theta_e)
    -> VF voltage generation or IF PID chain
    -> voltage vector limit
    -> inverse Park(theta_e)
    -> normalized alpha-beta modulation
    -> SVPWM
    -> duty_a / duty_b / duty_c
```

## PID 关系

FOC 直接复用 `common/controller/pid`。位置环、速度环、d 轴电流环和 q 轴电流环分别对应 `pid_pos`、`pid_vel`、`pid_id`、`pid_iq`，通过 `enable_pos_loop`、`enable_vel_loop`、`enable_id_loop`、`enable_iq_loop` 控制是否参与本周期。

## 历史边界

本目录不保留历史兼容层。第一版接口、入口函数和辅助模块只通过 Git history / tag 查看，不在当前源码、示例、测试或文档中作为可用接口描述。
