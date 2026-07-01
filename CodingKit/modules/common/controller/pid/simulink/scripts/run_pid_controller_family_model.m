% run_pid_controller_family_model.m
run(fullfile(fileparts(mfilename('fullpath')), 'build_pid_controller_family_model.m'));
open_system(fullfile(fileparts(mfilename('fullpath')), '..', 'models', 'generated', 'pid_controller_family_aw.slx'));
