/**
 * @file foc_angle.c
 * @brief 电角度计算、校准和开环角度模块实现。
 * @author HU JIAXUAN
 */

#include "foc_angle.h"

#include <math.h>

/**
 * @brief 规范方向配置。
 * @param[in] direction 用户配置方向。
 * @return 小于 0 返回 -1，否则返回 +1。
 */
static int8_t foc_safe_direction(int8_t direction)
{
    return (direction < 0) ? -1 : 1;
}

/**
 * @brief 将频率转换为周期。
 * @param[in] controlFreq 更新频率，单位 Hz。
 * @return 更新周期，单位 s；频率非法时返回 0。
 */
static float foc_angle_safe_period(float controlFreq)
{
    if (controlFreq <= 0.0f) {
        return 0.0f;
    }

    return 1.0f / controlFreq;
}

bool foc_angle_init(FocAngle *angle)
{
    if (angle == 0) {
        return false;
    }

    *angle = (FocAngle){0};
    angle->config.mode = FOC_ANGLE_MODE_SENSOR;
    angle->config.polePairs = 1U;
    angle->config.direction = 1;
    angle->config.controlFreq = FOC_ANGLE_DEFAULT_CONTROL_FREQ_HZ;
    angle->state.sincos.cosValue = 1.0f;

    return true;
}

float foc_wrap_0_2pi(float angleRad)
{
    float result = fmodf(angleRad, FOC_TWO_PI);

    if (result < 0.0f) {
        result += FOC_TWO_PI;
    }

    return result;
}

float foc_wrap_pm_pi(float angleRad)
{
    float result = foc_wrap_0_2pi(angleRad);

    if (result > FOC_PI) {
        result -= FOC_TWO_PI;
    }

    return result;
}

bool foc_angle_set_open_loop_phase(FocAngle *angle, float electricalAngleRad)
{
    if (angle == 0) {
        return false;
    }

    angle->state.openLoopElectricalAngleRad = foc_wrap_0_2pi(electricalAngleRad);
    return true;
}

/**
 * @brief 根据机械角度计算零位偏置。
 * @param[in,out] angle 电角度对象；输入为当前机械角度和目标对齐电角度。
 * @return true 表示校准成功；false 表示对象为空或极对数无效。
 *
 * 实现方法：先按方向和极对数计算当前原始电角度，再求出让当前机械位置
 * 对齐到 alignElectricalRad 所需的 offsetRad。该函数只计算偏置，不驱动电机。
 */
bool foc_angle_calibrate(FocAngle *angle)
{
    float rawElectrical;
    int8_t direction;

    if ((angle == 0) || (angle->config.polePairs == 0U)) {
        return false;
    }

    direction = foc_safe_direction(angle->config.direction);
    rawElectrical = (float)direction * (float)angle->config.polePairs * angle->input.mechanicalAngleRad;

    angle->config.offsetRad = foc_wrap_pm_pi(angle->input.alignElectricalRad - rawElectrical);
    angle->state.calibrated = true;

    return true;
}

/**
 * @brief 更新一次电角度和对应三角函数。
 * @param[in,out] angle 电角度对象；输入来自 angle->input 和 angle->config。
 * @return true 表示电角度输出有效；false 表示对象为空、极对数无效或开环周期无效。
 *
 * 实现方法：
 * 1. SENSOR 模式：electrical = direction * polePairs * mechanicalAngle + offset；
 * 2. OPEN_LOOP 模式：openLoopAngle += direction * openLoopElectricalSpeed * period；
 * 3. FIXED 模式：electrical = direction * fixedElectricalAngle + offset；
 * 4. 对实时电角度叠加 electricalSpeed * phaseCompTime；
 * 5. 将结果包装到 0~2pi，并计算 sin/cos。
 *
 * 控制依据：闭环 FOC 需要转子同步电角度；开环启动或开环电压输出时没有可靠
 * 转子角度反馈，只能用给定电角速度积分生成旋转电压/电流矢量。该模块只生成
 * 电角度，不判断能否切闭环，不实现传感器读取或观测器。
 */
bool foc_angle_update(FocAngle *angle)
{
    float electrical;
    float compensated;
    float period;
    int8_t direction;

    if ((angle == 0) || (angle->config.polePairs == 0U)) {
        return false;
    }

    direction = foc_safe_direction(angle->config.direction);
    angle->state.valid = false;

    if (angle->config.mode == FOC_ANGLE_MODE_OPEN_LOOP) {
        period = foc_angle_safe_period(angle->config.controlFreq);
        if (period <= 0.0f) {
            return false;
        }

        angle->state.electricalSpeedRadPerSec =
            (float)direction * angle->input.openLoopElectricalSpeedRadPerSec;
        angle->state.openLoopElectricalAngleRad = foc_wrap_0_2pi(
            angle->state.openLoopElectricalAngleRad + angle->state.electricalSpeedRadPerSec * period);
        electrical = angle->state.openLoopElectricalAngleRad + angle->config.offsetRad;
    } else if (angle->config.mode == FOC_ANGLE_MODE_FIXED) {
        angle->state.electricalSpeedRadPerSec = 0.0f;
        electrical = (float)direction * angle->input.fixedElectricalAngleRad + angle->config.offsetRad;
    } else {
        angle->state.electricalSpeedRadPerSec =
            (float)direction * (float)angle->config.polePairs * angle->input.mechanicalSpeedRadPerSec;
        electrical = (float)direction * (float)angle->config.polePairs * angle->input.mechanicalAngleRad;
        electrical += angle->config.offsetRad;
    }

    compensated = electrical + angle->state.electricalSpeedRadPerSec * angle->config.phaseCompTime;

    angle->state.electricalAngleRad = foc_wrap_0_2pi(compensated);
    angle->state.sincos = foc_sincos(angle->state.electricalAngleRad);
    angle->state.valid = true;

    return true;
}
