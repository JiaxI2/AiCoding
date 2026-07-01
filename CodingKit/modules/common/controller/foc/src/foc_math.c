/**
 * @file foc_math.c
 * @brief FOC 坐标变换数学模块实现。
 * @author HU JIAXUAN
 */

#include "foc_math.h"

#include <math.h>

FocSinCos foc_sincos(float angleRad)
{
    FocSinCos result;

    /* 三角函数集中在这里，便于后续替换为查表或平台快速数学库。 */
    result.sinValue = sinf(angleRad);
    result.cosValue = cosf(angleRad);

    return result;
}

FocAb foc_clarke(FocPhase phase)
{
    FocAb result;

    /* 三相平衡系统的幅值一致 Clarke 变换。 */
    result.alpha = phase.a;
    result.beta = FOC_ONE_BY_SQRT3 * (phase.b - phase.c);

    return result;
}

FocDq foc_park(FocAb ab, FocSinCos sc)
{
    FocDq result;

    /* 将静止坐标量投影到电角度同步旋转坐标系。 */
    result.d = sc.cosValue * ab.alpha + sc.sinValue * ab.beta;
    result.q = sc.cosValue * ab.beta - sc.sinValue * ab.alpha;

    return result;
}

FocAb foc_inv_park(FocDq dq, FocSinCos sc)
{
    FocAb result;

    /* 将 dq 电压或电流指令还原到静止坐标系。 */
    result.alpha = sc.cosValue * dq.d - sc.sinValue * dq.q;
    result.beta = sc.sinValue * dq.d + sc.cosValue * dq.q;

    return result;
}

FocPhase foc_inv_clarke(FocAb ab)
{
    FocPhase result;

    /* 用于从 alpha-beta 指令恢复三相等效指令。 */
    result.a = ab.alpha;
    result.b = -0.5f * ab.alpha + FOC_SQRT3_BY_2 * ab.beta;
    result.c = -0.5f * ab.alpha - FOC_SQRT3_BY_2 * ab.beta;

    return result;
}

float foc_clamp(float value, float minValue, float maxValue)
{
    float result = value;

    /* 异常范围按原值返回，避免隐藏配置错误。 */
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
