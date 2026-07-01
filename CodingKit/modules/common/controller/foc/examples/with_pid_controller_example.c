/**
 * @file with_pid_controller_example.c
 * @brief 与外部 PID 控制器互补使用示例。
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

typedef struct {
    float output; /* 外部控制器输出，单位 V 或 pu。 */
} AppPidOutput;

void app_foc_with_external_pid_example(AppPidOutput dAxis, AppPidOutput qAxis)
{
    Foc focCtl;

    (void)foc_init(&focCtl);

    /* 外部 PID 已经计算好 vd/vq 时，FOC 只做坐标变换和 SVPWM。 */
    focCtl.config.controlMode = FOC_CONTROL_MODE_OPEN_VOLTAGE;
    focCtl.config.controlFreq = 20000.0f;
    focCtl.config.maxVoltage = 12.0f;
    focCtl.config.modulationLimit = FOC_SQRT3_BY_2;

    focCtl.input.vbusVoltage = 24.0f;
    focCtl.input.electricalAngleRad = 1.2f;
    focCtl.input.voltageFeedforward.d = dAxis.output;
    focCtl.input.voltageFeedforward.q = qAxis.output;

    (void)foc(&focCtl);
}
