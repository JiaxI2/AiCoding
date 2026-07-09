/**
 * @file with_pid_controller_example.c
 * @brief Flat VF example using voltage commands from an external controller.
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

typedef struct {
    float output;
} AppPidOutput;

void app_foc_with_external_pid_example(AppPidOutput dAxis, AppPidOutput qAxis)
{
    Foc focCtl;

    (void)foc_init(&focCtl);

    focCtl.mode = FOC_MODE_VF;
    focCtl.angle_mode = FOC_ANGLE_SENSOR;
    focCtl.control_freq = 20000.0f;
    focCtl.vbus = 24.0f;
    focCtl.theta_e = 1.2f;
    focCtl.cmd_vd = dAxis.output;
    focCtl.cmd_vq = qAxis.output;
    focCtl.max_voltage = 12.0f;
    focCtl.modulation_limit = FOC_SQRT3_BY_2;

    (void)foc_loop(&focCtl);
}
