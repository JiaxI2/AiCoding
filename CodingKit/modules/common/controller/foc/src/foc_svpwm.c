/**
 * @file foc_svpwm.c
 * @brief SVPWM 调制模块实现。
 * @author HU JIAXUAN
 *
 */

#include "foc_svpwm.h"

#include <math.h>

static float foc_svpwm_abs(float value);
static float foc_svpwm_max3(float a, float b, float c);
static float foc_svpwm_min3(float a, float b, float c);
static float foc_svpwm_safe_max_modulation(float maxModulation);

/**
 * @brief 计算浮点绝对值。
 * @param[in] value 输入值，单位由调用方决定。
 * @return 非负绝对值，单位同输入。
 */
static float foc_svpwm_abs(float value)
{
    if (value < 0.0f) {
        return -value;
    }

    return value;
}

/**
 * @brief 返回三个浮点数中的最大值。
 * @param[in] a 第一个输入值，单位由调用方决定。
 * @param[in] b 第二个输入值，单位同 a。
 * @param[in] c 第三个输入值，单位同 a。
 * @return 最大值，单位同输入。
 */
static float foc_svpwm_max3(float a, float b, float c)
{
    float result = a;

    if (b > result) {
        result = b;
    }
    if (c > result) {
        result = c;
    }

    return result;
}

/**
 * @brief 返回三个浮点数中的最小值。
 * @param[in] a 第一个输入值，单位由调用方决定。
 * @param[in] b 第二个输入值，单位同 a。
 * @param[in] c 第三个输入值，单位同 a。
 * @return 最小值，单位同输入。
 */
static float foc_svpwm_min3(float a, float b, float c)
{
    float result = a;

    if (b < result) {
        result = b;
    }
    if (c < result) {
        result = c;
    }

    return result;
}

/**
 * @brief 归一化 SVPWM 调制矢量上限。
 * @param[in] maxModulation 用户配置的调制矢量上限，无单位。
 * @return 有效调制矢量上限，无单位，范围 (0, sqrt(3)/2]。
 */
static float foc_svpwm_safe_max_modulation(float maxModulation)
{
    float result = maxModulation;

    if (result <= 0.0f) {
        result = FOC_SQRT3_BY_2;
    }

    if (result > FOC_SQRT3_BY_2) {
        result = FOC_SQRT3_BY_2;
    }

    return result;
}

/**
 * @brief 初始化 SVPWM 控制器对象。
 * @param[out] controller SVPWM 控制器对象；不能为空。
 * @return 初始化成功返回 true；controller 为空返回 false。
 */
bool foc_svpwm_init(FocSvpwm *controller)
{
    if (controller == 0) {
        return false;
    }

    *controller = (FocSvpwm){0};
    controller->config.maxModulation = FOC_SVPWM_DEFAULT_MAX_MODULATION;
    controller->config.enableAutoScale = true;
    controller->state.dutyA = 0.5f;
    controller->state.dutyB = 0.5f;
    controller->state.dutyC = 0.5f;
    controller->state.scale = 1.0f;

    return true;
}

/**
 * @brief 执行一次 SVPWM 调制计算。
 * @param[in,out] controller SVPWM 控制器对象；不能为空。
 * @return duty 输出有效返回 true；输入为空或超限且不允许自动缩放时返回 false。
 */
bool foc_svpwm(FocSvpwm *controller)
{
    float vectorMag;
    float maxModulation;
    float scale = 1.0f;
    FocAb mod;
    FocPhase phase;
    float phaseMax;
    float phaseMin;
    float commonMode;

    if (controller == 0) {
        return false;
    }

    mod = controller->input.modulation;
    maxModulation = foc_svpwm_safe_max_modulation(controller->config.maxModulation);
    vectorMag = sqrtf(mod.alpha * mod.alpha + mod.beta * mod.beta);

    controller->state.valid = true;
    controller->state.saturated = false;

    if (vectorMag > maxModulation) {
        controller->state.saturated = true;
        if (controller->config.enableAutoScale) {
            scale = maxModulation / vectorMag;
            mod.alpha *= scale;
            mod.beta *= scale;
        } else {
            controller->state.valid = false;
        }
    }

    phase = foc_inv_clarke(mod);
    phaseMax = foc_svpwm_max3(phase.a, phase.b, phase.c);
    phaseMin = foc_svpwm_min3(phase.a, phase.b, phase.c);
    commonMode = -0.5f * (phaseMax + phaseMin);

    controller->state.dutyA = foc_clamp(0.5f + phase.a + commonMode, 0.0f, 1.0f);
    controller->state.dutyB = foc_clamp(0.5f + phase.b + commonMode, 0.0f, 1.0f);
    controller->state.dutyC = foc_clamp(0.5f + phase.c + commonMode, 0.0f, 1.0f);
    controller->state.scale = scale;

    if ((foc_svpwm_abs(controller->state.dutyA - 0.0f) <= 0.0f) ||
        (foc_svpwm_abs(controller->state.dutyA - 1.0f) <= 0.0f) ||
        (foc_svpwm_abs(controller->state.dutyB - 0.0f) <= 0.0f) ||
        (foc_svpwm_abs(controller->state.dutyB - 1.0f) <= 0.0f) ||
        (foc_svpwm_abs(controller->state.dutyC - 0.0f) <= 0.0f) ||
        (foc_svpwm_abs(controller->state.dutyC - 1.0f) <= 0.0f)) {
        controller->state.saturated = true;
    }

    return controller->state.valid;
}
