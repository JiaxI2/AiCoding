% build_pid_controller_family_model.m
% 生成 P / PI / PD / PID 四路对照模型。
% PI 和 PID 支路包含 back-calculation 积分抗饱和说明。

model = 'pid_controller_family_aw';
outDir = fullfile(fileparts(mfilename('fullpath')), '..', 'models', 'generated');
if ~exist(outDir, 'dir')
    mkdir(outDir);
end
modelPath = fullfile(outDir, [model '.slx']);

if bdIsLoaded(model)
    close_system(model, 0);
end
new_system(model);
open_system(model);

x0 = 40;
y0 = 40;
branches = {'P: kp!=0, ki=0, kd=0', 'PI: kp!=0, ki!=0, kd=0, AW enabled', ...
            'PD: kp!=0, ki=0, kd!=0', 'PID: kp!=0, ki!=0, kd!=0, AW enabled'};

for i = 1:numel(branches)
    y = y0 + (i - 1) * 120;
    add_block('simulink/Sources/Step', [model '/setpoint_' num2str(i)], 'Position', [x0 y x0+40 y+30]);
    add_block('simulink/Sources/Constant', [model '/feedback_' num2str(i)], 'Value', '0', 'Position', [x0 y+50 x0+40 y+80]);
    add_block('simulink/Math Operations/Sum', [model '/error_' num2str(i)], 'Inputs', '+-', 'Position', [x0+90 y+15 x0+120 y+55]);
    add_block('simulink/Commonly Used Blocks/Gain', [model '/controller_note_' num2str(i)], ...
        'Gain', '1', 'Position', [x0+170 y+15 x0+230 y+55]);
    add_block('simulink/Sinks/Scope', [model '/scope_' num2str(i)], 'Position', [x0+280 y+10 x0+310 y+60]);
    add_line(model, ['setpoint_' num2str(i) '/1'], ['error_' num2str(i) '/1']);
    add_line(model, ['feedback_' num2str(i) '/1'], ['error_' num2str(i) '/2']);
    add_line(model, ['error_' num2str(i) '/1'], ['controller_note_' num2str(i) '/1']);
    add_line(model, ['controller_note_' num2str(i) '/1'], ['scope_' num2str(i) '/1']);
    set_param([model '/controller_note_' num2str(i)], 'Name', branches{i});
end

save_system(model, modelPath);
close_system(model);
fprintf('Generated: %s\n', modelPath);
