/**
 * @file motion_loop_example.c
 * @brief Flat IF position and velocity loop example.
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

int main(void)
{
    Foc controller;

    (void)foc_init(&controller);

    controller.mode = FOC_MODE_IF;
    controller.angle_mode = FOC_ANGLE_SENSOR;
    controller.control_freq = 10000.0f;
    controller.vbus = 24.0f;
    controller.theta_e = 0.0f;
    controller.ia = 0.0f;
    controller.ib = 0.0f;
    controller.ic = 0.0f;
    controller.cmd_pos = 1.0f;
    controller.pos = 0.0f;
    controller.vel = 0.0f;
    controller.cmd_id = 0.0f;
    controller.max_voltage = 12.0f;
    controller.enable_pos_loop = true;
    controller.enable_vel_loop = true;
    controller.enable_id_loop = true;
    controller.enable_iq_loop = true;
    controller.pid_pos.config.kp = 20.0f;
    controller.pid_vel.config.kp = 1.0f;
    controller.pid_vel.config.ki = 5.0f;
    controller.pid_id.config.kp = 2.0f;
    controller.pid_iq.config.kp = 2.0f;

    (void)foc_loop(&controller);

    return 0;
}
