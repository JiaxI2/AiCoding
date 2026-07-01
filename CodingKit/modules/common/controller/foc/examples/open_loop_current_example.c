/**
 * @file open_loop_current_example.c
 * @brief 开环角度加闭环电流示例。
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

void app_foc_open_angle_current_example(void)
{
    Foc focCtl;
    FocAngle angle;

    (void)foc_init(&focCtl);
    (void)foc_angle_init(&angle);

    /* 角度开环：按给定电角速度积分生成旋转角度。 */
    angle.config.mode = FOC_ANGLE_MODE_OPEN_LOOP;
    angle.config.polePairs = 4U;
    angle.config.direction = 1;
    angle.config.controlFreq = 20000.0f;
    angle.input.openLoopElectricalSpeedRadPerSec = 80.0f;
    (void)foc_angle_update(&angle);

    /* 电流闭环：仍然使用三相电流反馈调节 Id/Iq。 */
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
    focCtl.input.currentSetpoint.q = 0.8f;

    (void)foc(&focCtl);
}
