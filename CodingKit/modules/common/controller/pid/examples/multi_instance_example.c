#include "../src/pid.h"

typedef struct {
    Pid loop0; /* 控制器实例 0。 */
    Pid loop1; /* 控制器实例 1。 */
    Pid loop2; /* 控制器实例 2。 */
} ControlGroup;

int main(void)
{
    ControlGroup group[2] = {0};
    float output0;
    float output1;

    (void)pid_init(&group[0].loop0);

    group[0].loop0.config.controlFreq = 1000.0f;
    group[0].loop0.config.kp = 0.5f;
    group[0].loop0.config.ki = 8.0f;
    group[0].loop0.config.outputLimit.enable = true;
    group[0].loop0.config.outputLimit.min = -2.0f;
    group[0].loop0.config.outputLimit.max = 2.0f;
    group[0].loop0.config.integralLimit.enable = true;
    group[0].loop0.config.integralLimit.min = -1.0f;
    group[0].loop0.config.integralLimit.max = 1.0f;
    group[0].loop0.config.antiWindupGain = 1.0f;

    group[1].loop0 = group[0].loop0;

    group[0].loop0.input.setpoint = 120.0f;
    group[0].loop0.input.feedback = 100.0f;
    group[1].loop0.input.setpoint = -80.0f;
    group[1].loop0.input.feedback = -70.0f;

    output0 = pid(&group[0].loop0);
    output1 = pid(&group[1].loop0);

    (void)output0;
    (void)output1;
    return 0;
}
