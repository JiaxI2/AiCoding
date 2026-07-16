% build_foc_reference_model.m
% 说明：生成 FOC 模块参考模型框架。
% 本脚本不依赖硬件，只用于搭建 Clarke、Park、电角度和 SVPWM 的仿真结构。

model = 'foc_reference_model';
if bdIsLoaded(model)
    close_system(model, 0);
end
new_system(model);
open_system(model);

add_block('simulink/Sources/Sine Wave', [model '/mechanical_angle']);
add_block('simulink/Sources/Constant', [model '/pole_pairs']);
add_block('simulink/Sources/Constant', [model '/vbus_voltage']);
add_block('simulink/Sinks/Scope', [model '/scope']);

set_param([model '/pole_pairs'], 'Value', '4');
set_param([model '/vbus_voltage'], 'Value', '24');

save_system(model, fullfile('simulink', 'models', [model '.slx']));
