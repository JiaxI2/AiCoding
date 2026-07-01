/**
 * @file current_loop_example.c
 * @brief FOC 闭环电流模式示例。
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

void app_foc_closed_current_example(void)
{
    Foc focCtl;
    FocAngle angle;

    (void)foc_init(&focCtl);
    (void)foc_angle_init(&angle);

    angle.config.mode = FOC_ANGLE_MODE_SENSOR;
    angle.config.polePairs = 4U;
    angle.config.direction = 1;
    angle.config.phaseCompTime = 0.0f;
    angle.input.mechanicalAngleRad = 0.3f;
    angle.input.mechanicalSpeedRadPerSec = 20.0f;
    (void)foc_angle_update(&angle);

    focCtl.config.controlMode = FOC_CONTROL_MODE_CLOSED_CURRENT;
    focCtl.config.controlFreq = 20000.0f;
    focCtl.config.currentKp = 2.0f;
    focCtl.config.currentKi = 800.0f;
    focCtl.config.maxVoltage = 12.0f;
    focCtl.config.modulationLimit = FOC_SQRT3_BY_2;
    focCtl.config.integratorDecay = 0.99f;

    focCtl.input.vbusVoltage = 24.0f;
    focCtl.input.electricalAngleRad = angle.state.electricalAngleRad;
    focCtl.input.phaseCurrent.a = 0.0f;
    focCtl.input.phaseCurrent.b = 0.0f;
    focCtl.input.phaseCurrent.c = 0.0f;
    focCtl.input.currentSetpoint.d = 0.0f;
    focCtl.input.currentSetpoint.q = 1.0f;
    focCtl.input.voltageFeedforward.d = 0.0f;
    focCtl.input.voltageFeedforward.q = 0.0f;

    if (foc(&focCtl)) {
        /* 输出只包含通用占空比，不直接写硬件。 */
        (void)focCtl.state.dutyA;
        (void)focCtl.state.dutyB;
        (void)focCtl.state.dutyC;
    }
}
