#ifndef FOC_MATH_H
#define FOC_MATH_H

/**
 * @file foc_math.h
 * @brief FOC 坐标变换数学模块。
 * @author HU JIAXUAN
 *
 * 本模块只做 Clarke、Park、反 Park、反 Clarke 和角度三角函数计算。
 * 不读取 ADC，不写 PWM，不绑定电机、编码器、驱动器或控制环路。
 *
 * 若使用标幺，所有输入输出量由上层按统一基值归一化。
 */

#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

#define FOC_PI              (3.14159265358979323846f)
#define FOC_TWO_PI          (6.28318530717958647692f)
#define FOC_ONE_BY_SQRT3    (0.57735026918962576451f)
#define FOC_SQRT3_BY_2      (0.86602540378443864676f)
#define FOC_TWO_BY_SQRT3    (1.15470053837925152902f)

typedef struct {
    float a; /* A 相量，单位同输入。 */
    float b; /* B 相量，单位同输入。 */
    float c; /* C 相量，单位同输入。 */
} FocPhase;

typedef struct {
    float alpha; /* 静止 alpha 轴分量，单位同输入。 */
    float beta;  /* 静止 beta 轴分量，单位同输入。 */
} FocAb;

typedef struct {
    float d; /* 旋转 d 轴分量，单位同输入。 */
    float q; /* 旋转 q 轴分量，单位同输入。 */
} FocDq;

typedef struct {
    float sinValue; /* sin(angle)，无单位。 */
    float cosValue; /* cos(angle)，无单位。 */
} FocSinCos;

/**
 * @brief 根据角度计算正弦和余弦。
 * @param angleRad 角度，单位 rad。
 * @return 正弦和余弦结果。
 */
FocSinCos foc_sincos(float angleRad);

/**
 * @brief 三相静止量转换为 alpha-beta 静止坐标量。
 * @param phase 三相输入，单位同实际物理量。
 * @return alpha-beta 静止坐标量。
 */
FocAb foc_clarke(FocPhase phase);

/**
 * @brief alpha-beta 静止坐标量转换为 dq 旋转坐标量。
 * @param ab alpha-beta 静止坐标量。
 * @param sc 电角度的正弦和余弦。
 * @return dq 旋转坐标量。
 */
FocDq foc_park(FocAb ab, FocSinCos sc);

/**
 * @brief dq 旋转坐标量转换为 alpha-beta 静止坐标量。
 * @param dq dq 旋转坐标量。
 * @param sc 电角度的正弦和余弦。
 * @return alpha-beta 静止坐标量。
 */
FocAb foc_inv_park(FocDq dq, FocSinCos sc);

/**
 * @brief alpha-beta 静止坐标量转换为三相量。
 * @param ab alpha-beta 静止坐标量。
 * @return 三相输出量。
 */
FocPhase foc_inv_clarke(FocAb ab);

/**
 * @brief 限幅到指定范围。
 * @param value 输入值。
 * @param minValue 最小值。
 * @param maxValue 最大值。
 * @return 限幅后的值。
 */
float foc_clamp(float value, float minValue, float maxValue);

#ifdef __cplusplus
}
#endif

#endif
