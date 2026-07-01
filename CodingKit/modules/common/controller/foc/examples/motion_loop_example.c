/**
 * @file motion_loop_example.c
 * @brief 外环级联控制示例。
 * @author HU JIAXUAN
 */

#include "foc.h"

int main(void)
{
    Foc controller;
    (void)foc_init(&controller);

    controller.config.controlMode = FOC_CONTROL_MODE_MOTION_CURRENT;
    controller.config.enableMotionControl = true;
    controller.config.currentKp = 2.0f;
    controller.config.currentKi = 800.0f;
    controller.config.maxVoltage = 12.0f;
    controller.input.vbusVoltage = 24.0f;
    controller.input.electricalAngleRad = 0.0f;

    controller.motion.config.controlMode = FOC_MOTION_CONTROL_POSITION;
    controller.motion.config.inputMode = FOC_MOTION_INPUT_POS_FILTER;
    controller.motion.config.posGain = 20.0f;
    controller.motion.config.velGain = 1.0f;
    controller.motion.config.velIntegratorGain = 5.0f;
    controller.motion.config.velIntegratorLimit = 5.0f;
    controller.motion.config.velLimit = 30.0f;
    controller.motion.config.currentLimit = 10.0f;
    controller.motion.config.inertiaFeedforwardGain = 0.01f;
    controller.motion.input.positionSetpoint = 1.0f;
    controller.motion.input.positionFeedback = 0.0f;
    controller.motion.input.velocityFeedback = 0.0f;

    controller.input.phaseCurrent.a = 0.0f;
    controller.input.phaseCurrent.b = 0.0f;
    controller.input.phaseCurrent.c = 0.0f;

    (void)foc(&controller);

    return 0;
}
