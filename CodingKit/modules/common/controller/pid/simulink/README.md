# Simulink 目录

`models/reference/` 保存用户给出的抗饱和参考模型副本。

`scripts/build_pid_controller_family_model.m` 用于生成 P / PI / PD / PID 四路对照模型：

```matlab
run('simulink/scripts/build_pid_controller_family_model.m')
```

生成模型路径：

```text
simulink/models/generated/pid_controller_family_aw.slx
```


## antiwindup_complete_reference.slx 拓扑

该模型用于说明反算式积分抗饱和：

```text
uc -> Saturation -> u
uc - u -> antiWindupGain -> 从积分输入中扣除
```

等价写法：

```text
积分输入 = e + antiWindupGain * (u - uc)
```

C 实现中的 `rawOutput` 对应 `uc`，`limitedOutput` 对应 `u`。
