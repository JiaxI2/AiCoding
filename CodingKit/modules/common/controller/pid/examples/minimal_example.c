#include "../src/pid.h"

int main(void)
{
    Pid controller = {0};
    (void)pid_init(&controller);
    float command;

    /* 最少必配：controlFreq(Hz)、kp/ki/kd、输出限幅(输出单位/pu)。 */
    controller.config.controlFreq = 1000.0f;
    controller.config.kp = 0.8f;
    controller.config.ki = 12.0f;
    controller.config.kd = 0.0f;
    controller.config.outputLimit.enable = true;
    controller.config.outputLimit.min = -1.0f;
    controller.config.outputLimit.max = 1.0f;
    controller.config.integralLimit.enable = true;
    controller.config.integralLimit.min = -0.8f;
    controller.config.integralLimit.max = 0.8f;
    controller.config.antiWindupGain = 1.0f;

    /* 每个控制周期只更新输入字段，然后调用 pid(&controller)。 */
    controller.input.setpoint = 1000.0f;
    controller.input.feedback = 920.0f;
    controller.input.feedforward = 0.0f;

    command = pid(&controller);

    (void)command;
    return 0;
}
