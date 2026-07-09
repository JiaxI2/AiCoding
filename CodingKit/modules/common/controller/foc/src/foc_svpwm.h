#ifndef FOC_SVPWM_H
#define FOC_SVPWM_H

/**
 * @file foc_svpwm.h
 * @brief SVPWM 调制模块。
 * @author HU JIAXUAN
 *
 * 本模块只把 alpha-beta 调制量转换为三相中心对齐 PWM 占空比。
 * 输入为归一化调制量，无单位；输出 duty 范围为 0~1。
 *
 */

#include <stdbool.h>

#include "foc_math.h"

#ifndef FOC_SVPWM_DEFAULT_MAX_MODULATION
#define FOC_SVPWM_DEFAULT_MAX_MODULATION (FOC_SQRT3_BY_2)
#endif

#ifdef __cplusplus
extern "C" {
#endif

/** @brief SVPWM 配置。 */
typedef struct {
    float maxModulation; /* alpha-beta 调制矢量幅值上限，无单位，通常不超过 sqrt(3)/2。 */
    bool enableAutoScale; /* 超限时是否自动缩放，无单位；true 缩放，false 标记无效。 */
} FocSvpwmConfig;

/** @brief SVPWM 本周期输入。 */
typedef struct {
    FocAb modulation; /* alpha-beta 归一化调制量，无单位。 */
} FocSvpwmInput;

/** @brief SVPWM 本周期状态和输出。 */
typedef struct {
    float dutyA; /* A 相占空比，无单位，范围 0~1。 */
    float dutyB; /* B 相占空比，无单位，范围 0~1。 */
    float dutyC; /* C 相占空比，无单位，范围 0~1。 */
    float scale; /* 调制矢量缩放系数，无单位。 */
    bool saturated; /* 调制量是否超限或 duty 是否到达 0/1 边界，无单位。 */
    bool valid; /* 本周期 duty 输出是否有效，无单位。 */
} FocSvpwmState;

/** @brief SVPWM 控制器对象。 */
typedef struct {
    FocSvpwmConfig config; /* SVPWM 配置区。 */
    FocSvpwmInput input; /* SVPWM 输入区。 */
    FocSvpwmState state; /* SVPWM 输出和状态区。 */
} FocSvpwm;

/**
 * @brief 初始化 SVPWM 控制器对象。
 * @param[out] controller SVPWM 控制器对象指针；不能为空。
 * @return 初始化成功返回 true；controller 为空返回 false。
 */
bool foc_svpwm_init(FocSvpwm *controller);

/**
 * @brief 执行一次 SVPWM 调制计算。
 * @param[in,out] controller SVPWM 控制器对象指针；不能为空。
 * @return duty 输出有效返回 true；输入为空或超限且不允许自动缩放时返回 false。
 */
bool foc_svpwm(FocSvpwm *controller);

#ifdef __cplusplus
}
#endif

#endif
