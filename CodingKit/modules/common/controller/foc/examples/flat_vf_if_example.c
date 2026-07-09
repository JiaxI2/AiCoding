/**
 * @file flat_vf_if_example.c
 * @brief Flat VF/IF FOC usage examples.
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

static volatile float g_pwm_a;
static volatile float g_pwm_b;
static volatile float g_pwm_c;

void app_foc_vf_open_loop_example(void)
{
    Foc foc;

    (void)foc_init(&foc);
    foc.mode = FOC_MODE_VF;
    foc.angle_mode = FOC_ANGLE_OPEN_LOOP;
    foc.control_freq = 10000.0f;
    foc.vbus = 24.0f;
    foc.open_loop_freq_hz = 5.0f;
    foc.dir = 1.0f;
    foc.cmd_vd = 0.0f;
    foc.cmd_vq = 0.0f;
    foc.vf_boost_v = 1.0f;
    foc.vf_gain_v_per_hz = 0.2f;
    foc.vf_min_v = 0.0f;
    foc.vf_max_v = 6.0f;

    if (foc_loop(&foc)) {
        g_pwm_a = foc.duty_a;
        g_pwm_b = foc.duty_b;
        g_pwm_c = foc.duty_c;
    }
}

void app_foc_if_open_loop_startup_example(void)
{
    Foc foc;

    (void)foc_init(&foc);
    foc.mode = FOC_MODE_IF;
    foc.angle_mode = FOC_ANGLE_OPEN_LOOP;
    foc.control_freq = 10000.0f;
    foc.vbus = 24.0f;
    foc.ia = 0.1f;
    foc.ib = -0.05f;
    foc.ic = -0.05f;
    foc.open_loop_freq_hz = 8.0f;
    foc.dir = 1.0f;
    foc.cmd_id = 0.0f;
    foc.cmd_iq = 1.5f;
    foc.pid_id.config.kp = 2.0f;
    foc.pid_iq.config.kp = 2.0f;

    if (foc_loop(&foc)) {
        g_pwm_a = foc.duty_a;
        g_pwm_b = foc.duty_b;
        g_pwm_c = foc.duty_c;
    }
}

void app_foc_if_three_loop_example(void)
{
    Foc foc;
    float theta = 0.25f;

    (void)foc_init(&foc);
    foc.mode = FOC_MODE_IF;
    foc.angle_mode = FOC_ANGLE_SENSOR;
    foc.control_freq = 10000.0f;
    foc.vbus = 24.0f;
    foc.theta_e = theta;
    foc.ia = 0.1f;
    foc.ib = -0.05f;
    foc.ic = -0.05f;
    foc.cmd_pos = 1.0f;
    foc.pos = 0.4f;
    foc.vel = 0.0f;
    foc.enable_pos_loop = true;
    foc.enable_vel_loop = true;
    foc.enable_id_loop = true;
    foc.enable_iq_loop = true;
    foc.pid_pos.config.kp = 20.0f;
    foc.pid_vel.config.kp = 0.5f;
    foc.pid_id.config.kp = 2.0f;
    foc.pid_iq.config.kp = 2.0f;

    if (foc_loop(&foc)) {
        g_pwm_a = foc.duty_a;
        g_pwm_b = foc.duty_b;
        g_pwm_c = foc.duty_c;
    }
}
