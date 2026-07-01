/**
 * @file minimal_example.c
 * @brief FOC 开环电压模式最小示例。
 * @author HU JIAXUAN
 */

#include "../src/foc.h"

void app_foc_open_voltage_example(void)
{
    Foc focCtl;
    FocAngle angle;

    (void)foc_init(&focCtl);
    (void)foc_angle_init(&angle);

    angle.config.mode = FOC_ANGLE_MODE_OPEN_LOOP;
    angle.config.polePairs = 4U;
    angle.config.direction = 1;
    angle.config.controlFreq = 20000.0f;
    angle.input.openLoopElectricalSpeedRadPerSec = 100.0f;
    (void)foc_angle_update(&angle);

    focCtl.config.controlMode = FOC_CONTROL_MODE_OPEN_VOLTAGE;
    focCtl.config.controlFreq = 20000.0f;
    focCtl.config.maxVoltage = 12.0f;
    focCtl.config.modulationLimit = FOC_SQRT3_BY_2;

    focCtl.input.vbusVoltage = 24.0f;
    focCtl.input.electricalAngleRad = angle.state.electricalAngleRad;
    focCtl.input.voltageFeedforward.d = 0.0f;
    focCtl.input.voltageFeedforward.q = 3.0f;

    if (foc(&focCtl)) {
        /* 将 dutyA/dutyB/dutyC 交给平台 PWM 层。 */
        (void)focCtl.state.dutyA;
        (void)focCtl.state.dutyB;
        (void)focCtl.state.dutyC;
    }
}
