/**
 * @file minimal_example.c
 * @brief Minimal flat VF open-loop voltage example.
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

void app_foc_vf_minimal_example(void)
{
    Foc focCtl;

    (void)foc_init(&focCtl);

    focCtl.mode = FOC_MODE_VF;
    focCtl.angle_mode = FOC_ANGLE_OPEN_LOOP;
    focCtl.control_freq = 20000.0f;
    focCtl.vbus = 24.0f;
    focCtl.open_loop_freq_hz = 15.0f;
    focCtl.dir = 1.0f;
    focCtl.cmd_vd = 0.0f;
    focCtl.cmd_vq = 3.0f;
    focCtl.max_voltage = 12.0f;
    focCtl.modulation_limit = FOC_SQRT3_BY_2;

    if (foc_loop(&focCtl)) {
        (void)focCtl.duty_a;
        (void)focCtl.duty_b;
        (void)focCtl.duty_c;
    }
}
