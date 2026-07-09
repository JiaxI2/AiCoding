#ifndef FOC_MATH_H
#define FOC_MATH_H

/**
 * @file foc_math.h
 * @brief FOC 坐标变换和基础数学工具。
 * @author HU JIAXUAN
 *
 * 本模块只提供 Clarke、Park、反 Park、反 Clarke、角度正余弦和限幅工具。
 * 本模块不读取 ADC、不写 PWM，也不绑定电机对象、驱动对象或控制环路状态。
 *
 */

#ifdef __cplusplus
extern "C" {
#endif

#define FOC_PI              (3.14159265358979323846f)
#define FOC_TWO_PI          (6.28318530717958647692f)
#define FOC_ONE_BY_SQRT3    (0.57735026918962576451f)
#define FOC_SQRT3_BY_2      (0.86602540378443864676f)
#define FOC_TWO_BY_SQRT3    (1.15470053837925152902f)

/** @brief 三相 abc 量。 */
typedef struct {
    float a; /* A 相分量，单位由调用场景决定，通常为 A、V 或 pu。 */
    float b; /* B 相分量，单位同 a。 */
    float c; /* C 相分量，单位同 a。 */
} FocPhase;

/** @brief 静止 alpha-beta 坐标系量。 */
typedef struct {
    float alpha; /* alpha 轴分量，单位由调用场景决定，通常为 A、V 或 pu。 */
    float beta; /* beta 轴分量，单位同 alpha。 */
} FocAb;

/** @brief 同步旋转 dq 坐标系量。 */
typedef struct {
    float d; /* d 轴分量，单位由调用场景决定，通常为 A、V 或 pu。 */
    float q; /* q 轴分量，单位同 d。 */
} FocDq;

/** @brief 同一角度的正弦和余弦值。 */
typedef struct {
    float sinValue; /* sin(angle)，无单位。 */
    float cosValue; /* cos(angle)，无单位。 */
} FocSinCos;

/**
 * @brief 根据角度计算正弦和余弦。
 * @param[in] angleRad 输入角度，单位 rad。
 * @return 正弦和余弦结果，无单位。
 */
FocSinCos foc_sincos(float angleRad);

/**
 * @brief 将三相 abc 量转换为 alpha-beta 静止坐标系量。
 * @param[in] phase 三相输入，单位由调用场景决定。
 * @return alpha-beta 输出，单位同输入。
 */
FocAb foc_clarke(FocPhase phase);

/**
 * @brief 将 alpha-beta 静止坐标系量转换为 dq 同步旋转坐标系量。
 * @param[in] ab alpha-beta 输入，单位由调用场景决定。
 * @param[in] sc 电角度正余弦，无单位。
 * @return dq 输出，单位同输入。
 */
FocDq foc_park(FocAb ab, FocSinCos sc);

/**
 * @brief 将 dq 同步旋转坐标系量转换为 alpha-beta 静止坐标系量。
 * @param[in] dq dq 输入，单位由调用场景决定。
 * @param[in] sc 电角度正余弦，无单位。
 * @return alpha-beta 输出，单位同输入。
 */
FocAb foc_inv_park(FocDq dq, FocSinCos sc);

/**
 * @brief 将 alpha-beta 静止坐标系量转换为三相 abc 量。
 * @param[in] ab alpha-beta 输入，单位由调用场景决定。
 * @return 三相输出，单位同输入。
 */
FocPhase foc_inv_clarke(FocAb ab);

/**
 * @brief 将输入值限制到指定范围。
 * @param[in] value 输入值，单位由调用场景决定。
 * @param[in] minValue 下限，单位同 value。
 * @param[in] maxValue 上限，单位同 value。
 * @return 限幅后的值；当 minValue 大于 maxValue 时返回原值。
 */
float foc_clamp(float value, float minValue, float maxValue);

#ifdef __cplusplus
}
#endif

#endif
