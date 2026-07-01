#ifndef FOC_SVPWM_H
#define FOC_SVPWM_H

/**
 * @file foc_svpwm.h
 * @brief SVPWM 调制模块。
 * @author HU JIAXUAN
 *
 * 本模块只把 alpha-beta 调制量转换为三相占空比。
 * 输入为归一化调制量，输出占空比范围为 0~1。
 * 若系统使用标幺，调制量同样由上层统一归一化。
 */

#include <stdbool.h>

#include "foc_math.h"

#ifndef FOC_SVPWM_DEFAULT_MAX_MODULATION
#define FOC_SVPWM_DEFAULT_MAX_MODULATION (FOC_SQRT3_BY_2)
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    float maxModulation; /* 最大调制矢量幅值，无单位；建议不超过 sqrt(3)/2。 */
    bool enableAutoScale; /* 超限时是否自动缩放，true 缩放，false 仅置位告警。 */
} FocSvpwmConfig;

typedef struct {
    FocAb modulation; /* alpha-beta 调制量，无单位。 */
} FocSvpwmInput;

typedef struct {
    float dutyA; /* A 相占空比，无单位，范围 0~1。 */
    float dutyB; /* B 相占空比，无单位，范围 0~1。 */
    float dutyC; /* C 相占空比，无单位，范围 0~1。 */
    float scale; /* 调制缩放系数，无单位。 */
    bool saturated; /* 调制量是否发生缩放或限幅。 */
    bool valid; /* 输出是否有效。 */
} FocSvpwmState;

typedef struct {
    FocSvpwmConfig config; /* SVPWM 配置。 */
    FocSvpwmInput input; /* SVPWM 输入。 */
    FocSvpwmState state; /* SVPWM 输出和状态。 */
} FocSvpwm;


/**
 * @brief 初始化 SVPWM 对象。
 * @param[out] controller SVPWM 对象。
 * @return true 表示初始化成功，false 表示对象为空。
 *
 * 初始化内容：清零输入和状态，默认启用自动缩放，调制上限为线性 SVPWM 常用上限。
 */
bool foc_svpwm_init(FocSvpwm *controller);

/**
 * @brief 执行一次 SVPWM 调制。
 *
 * 输入 alpha-beta 调制量，先按 maxModulation 做线性区检查，再通过
 * 反 Clarke 和 common-mode 零序注入生成中心对齐三相 duty。
 * @param controller SVPWM 控制对象。
 * @return true 表示输出有效，false 表示参数无效。
 */
bool foc_svpwm(FocSvpwm *controller);

#ifdef __cplusplus
}
#endif

#endif
