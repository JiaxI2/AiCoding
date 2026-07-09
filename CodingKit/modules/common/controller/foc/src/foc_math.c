/**
 * @file foc_math.c
 * @brief FOC 坐标变换和基础数学工具实现。
 * @author HU JIAXUAN
 *
 */

#include "foc_math.h"

#include <math.h>

/**
 * @brief 根据角度计算正弦和余弦。
 * @param[in] angleRad 输入角度，单位 rad。
 * @return 正弦和余弦结果，无单位。
 */
FocSinCos foc_sincos(float angleRad)
{
    FocSinCos result;

    result.sinValue = sinf(angleRad);
    result.cosValue = cosf(angleRad);

    return result;
}

/**
 * @brief 将三相 abc 量转换为 alpha-beta 静止坐标系量。
 * @param[in] phase 三相输入，单位由调用场景决定。
 * @return alpha-beta 输出，单位同输入。
 */
FocAb foc_clarke(FocPhase phase)
{
    FocAb result;

    result.alpha = phase.a;
    result.beta = FOC_ONE_BY_SQRT3 * (phase.b - phase.c);

    return result;
}

/**
 * @brief 将 alpha-beta 静止坐标系量转换为 dq 同步旋转坐标系量。
 * @param[in] ab alpha-beta 输入，单位由调用场景决定。
 * @param[in] sc 电角度正余弦，无单位。
 * @return dq 输出，单位同输入。
 */
FocDq foc_park(FocAb ab, FocSinCos sc)
{
    FocDq result;

    result.d = sc.cosValue * ab.alpha + sc.sinValue * ab.beta;
    result.q = sc.cosValue * ab.beta - sc.sinValue * ab.alpha;

    return result;
}

/**
 * @brief 将 dq 同步旋转坐标系量转换为 alpha-beta 静止坐标系量。
 * @param[in] dq dq 输入，单位由调用场景决定。
 * @param[in] sc 电角度正余弦，无单位。
 * @return alpha-beta 输出，单位同输入。
 */
FocAb foc_inv_park(FocDq dq, FocSinCos sc)
{
    FocAb result;

    result.alpha = sc.cosValue * dq.d - sc.sinValue * dq.q;
    result.beta = sc.sinValue * dq.d + sc.cosValue * dq.q;

    return result;
}

/**
 * @brief 将 alpha-beta 静止坐标系量转换为三相 abc 量。
 * @param[in] ab alpha-beta 输入，单位由调用场景决定。
 * @return 三相输出，单位同输入。
 */
FocPhase foc_inv_clarke(FocAb ab)
{
    FocPhase result;

    result.a = ab.alpha;
    result.b = -0.5f * ab.alpha + FOC_SQRT3_BY_2 * ab.beta;
    result.c = -0.5f * ab.alpha - FOC_SQRT3_BY_2 * ab.beta;

    return result;
}

/**
 * @brief 将输入值限制到指定范围。
 * @param[in] value 输入值，单位由调用场景决定。
 * @param[in] minValue 下限，单位同 value。
 * @param[in] maxValue 上限，单位同 value。
 * @return 限幅后的值；当 minValue 大于 maxValue 时返回原值。
 */
float foc_clamp(float value, float minValue, float maxValue)
{
    float result = value;

    if (minValue > maxValue) {
        return result;
    }

    if (result < minValue) {
        result = minValue;
    } else if (result > maxValue) {
        result = maxValue;
    } else {
        /* 保持原值。 */
    }

    return result;
}
