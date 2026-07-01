# 积分抗饱和方案说明

## 参考模型

文件：

```text
simulink/models/reference/antiwindup_complete_reference.slx
```

模型中可识别的关键块：

| 模型块 | 作用 |
|---|---|
| `Sum1` | 计算误差 `e = reference - output` |
| `Integrator` | 积分状态 |
| `Sum` | 组合比例路径和积分路径，得到限幅前输出 `uc` |
| `Saturation` | 将 `uc` 限制为实际输出 `u` |
| `Sum2` | 计算 `uc - u` |
| `Gain2` | anti-windup 反馈增益 |
| `Sum3` | 将 `e - Kaw * (uc - u)` 作为积分输入 |
| `Gain` | 积分增益 `Ki` |

## 控制方程

```text
uc = P/I 组合后的限幅前输出
u  = saturation(uc)
aw = u - uc

I(k+1) = I(k) + Ki * [e(k) + Kaw * aw(k)] * Ts
```

这就是 back-calculation 积分抗饱和。输出没有饱和时，`u - uc = 0`，积分按普通 PI/PID 工作；输出饱和时，`u - uc` 会把积分项向脱离饱和的方向拉回。

## 与 C 实现的对应关系

| Simulink | C 字段 |
|---|---|
| `uc` | `state.rawOutput` |
| `u` | `state.output` / `limitedOutput` |
| `Kaw` | `config.antiWindupGain` |
| `Ki` | `config.ki` |
| `e` | `state.error` |

C 实现保留通用 PID 边界，不绑定参考模型中的具体数值。参考模型用于验证拓扑，工程中可按对象修改 `kp / ki / antiWindupGain / outputLimit`。
