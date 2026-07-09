/**
 * @file current_loop_example.c
 * @brief Flat IF sensor-angle current-loop example.
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

void app_foc_if_current_loop_example(void)
{
    Foc focCtl;

    (void)foc_init(&focCtl);

    focCtl.mode = FOC_MODE_IF;
    focCtl.angle_mode = FOC_ANGLE_SENSOR;
    focCtl.control_freq = 20000.0f;
    focCtl.vbus = 24.0f;
    focCtl.theta_e = 0.3f;
    focCtl.omega_e = 80.0f;
    focCtl.ia = 0.0f;
    focCtl.ib = 0.0f;
    focCtl.ic = 0.0f;
    focCtl.cmd_id = 0.0f;
    focCtl.cmd_iq = 1.0f;
    focCtl.max_voltage = 12.0f;
    focCtl.modulation_limit = FOC_SQRT3_BY_2;
    focCtl.pid_id.config.kp = 2.0f;
    focCtl.pid_id.config.ki = 800.0f;
    focCtl.pid_iq.config.kp = 2.0f;
    focCtl.pid_iq.config.ki = 800.0f;

    if (foc_loop(&focCtl)) {
        (void)focCtl.duty_a;
        (void)focCtl.duty_b;
        (void)focCtl.duty_c;
    }
}
