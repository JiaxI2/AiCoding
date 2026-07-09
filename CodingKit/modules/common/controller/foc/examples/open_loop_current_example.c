/**
 * @file open_loop_current_example.c
 * @brief Flat IF open-loop-angle startup example.
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

void app_foc_if_open_loop_startup_example(void)
{
    Foc focCtl;

    (void)foc_init(&focCtl);

    focCtl.mode = FOC_MODE_IF;
    focCtl.angle_mode = FOC_ANGLE_OPEN_LOOP;
    focCtl.control_freq = 20000.0f;
    focCtl.vbus = 24.0f;
    focCtl.open_loop_freq_hz = 12.0f;
    focCtl.dir = 1.0f;
    focCtl.ia = 0.0f;
    focCtl.ib = 0.0f;
    focCtl.ic = 0.0f;
    focCtl.cmd_id = 0.0f;
    focCtl.cmd_iq = 0.8f;
    focCtl.max_voltage = 12.0f;
    focCtl.modulation_limit = FOC_SQRT3_BY_2;
    focCtl.pid_id.config.kp = 2.0f;
    focCtl.pid_id.config.ki = 800.0f;
    focCtl.pid_iq.config.kp = 2.0f;
    focCtl.pid_iq.config.ki = 800.0f;

    (void)foc_loop(&focCtl);
}
