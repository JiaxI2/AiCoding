/**
 * @file foc_svpwm.c
 * @brief SVPWM 调制模块实现。
 * @author HU JIAXUAN
 */

#include "foc_svpwm.h"

#include <math.h>

static float foc_abs(float value)
{
    return (value >= 0.0f) ? value : -value;
}

static float foc_max3(float a, float b, float c)
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

static float foc_min3(float a, float b, float c)
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

static float foc_safe_max_modulation(float maxModulation)
{
    float result = maxModulation;

    /* 未配置时使用线性 SVPWM 常用上限。 */
    if (result <= 0.0f) {
        result = FOC_SQRT3_BY_2;
    }

    if (result > FOC_SQRT3_BY_2) {
        result = FOC_SQRT3_BY_2;
    }

    return result;
}


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
 * @param[in,out] controller SVPWM 对象；输入为 alpha-beta 调制量和调制配置。
 * @return true 表示 duty 输出有效；false 表示对象为空或调制量超限且不允许缩放。
 *
 * 实现方法：
 * 1. 读取 alpha-beta 调制量，并计算矢量幅值；
 * 2. 按 maxModulation 检查线性调制范围；
 * 3. 超限且 enableAutoScale 为 true 时，按比例缩放 alpha-beta 矢量；
 * 4. 通过反 Clarke 得到三相调制量；
 * 5. 计算 commonMode = -0.5 * (maxPhase + minPhase)，完成零序注入；
 * 6. duty = 0.5 + phase + commonMode，并限制到 0~1。
 *
 * 控制依据：零序注入形式和扇区时间形式都是 SVPWM 的实现方式。当前写法
 * 通过 common-mode 分量把三相 duty 居中，提升直流母线利用率，并保持输出
 * 与具体 PWM 外设解耦。
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
    maxModulation = foc_safe_max_modulation(controller->config.maxModulation);
    vectorMag = sqrtf(mod.alpha * mod.alpha + mod.beta * mod.beta);

    controller->state.valid = true;
    controller->state.saturated = false;

    /* 调制矢量超限时可选择自动缩放，避免 duty 溢出。 */
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
    phaseMax = foc_max3(phase.a, phase.b, phase.c);
    phaseMin = foc_min3(phase.a, phase.b, phase.c);
    commonMode = -0.5f * (phaseMax + phaseMin);

    /* 零序注入后映射到中心对齐 PWM 占空比。 */
    controller->state.dutyA = foc_clamp(0.5f + phase.a + commonMode, 0.0f, 1.0f);
    controller->state.dutyB = foc_clamp(0.5f + phase.b + commonMode, 0.0f, 1.0f);
    controller->state.dutyC = foc_clamp(0.5f + phase.c + commonMode, 0.0f, 1.0f);
    controller->state.scale = scale;

    if ((foc_abs(controller->state.dutyA - 0.0f) <= 0.0f) ||
        (foc_abs(controller->state.dutyA - 1.0f) <= 0.0f) ||
        (foc_abs(controller->state.dutyB - 0.0f) <= 0.0f) ||
        (foc_abs(controller->state.dutyB - 1.0f) <= 0.0f) ||
        (foc_abs(controller->state.dutyC - 0.0f) <= 0.0f) ||
        (foc_abs(controller->state.dutyC - 1.0f) <= 0.0f)) {
        controller->state.saturated = true;
    }

    return controller->state.valid;
}
