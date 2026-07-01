#ifndef FOC_ANGLE_H
#define FOC_ANGLE_H

/**
 * @file foc_angle.h
 * @brief 电角度计算、校准和开环角度模块。
 * @author HU JIAXUAN
 *
 * 本模块只处理 FOC 所需电角度，不读取编码器，不运行观测器，不写硬件。
 * 闭环角度模式使用上层给出的机械角度和机械角速度；开环角度模式使用
 * 上层给出的电角速度积分生成电角度；固定角度模式用于对齐、锁定或测试。
 */

#include <stdbool.h>
#include <stdint.h>

#include "foc_math.h"

#ifndef FOC_ANGLE_DEFAULT_CONTROL_FREQ_HZ
#define FOC_ANGLE_DEFAULT_CONTROL_FREQ_HZ (10000.0f)
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    FOC_ANGLE_MODE_SENSOR = 0,      /* 闭环角度：机械角度来自传感器或观测器。 */
    FOC_ANGLE_MODE_OPEN_LOOP = 1,   /* 开环角度：按电角速度积分生成角度。 */
    FOC_ANGLE_MODE_FIXED = 2        /* 固定角度：直接使用指定电角度。 */
} FocAngleMode;

typedef struct {
    FocAngleMode mode; /* 电角度模式。 */
    uint16_t polePairs; /* 极对数，无单位。 */
    int8_t direction; /* 方向，1 正向，-1 反向。 */
    float controlFreq; /* 角度更新频率，单位 Hz；开环角度模式必须配置。 */
    float offsetRad; /* 电角度零位偏置，单位 rad。 */
    float phaseCompTime; /* 相位延时补偿时间，单位 s；不用时填 0。 */
} FocAngleConfig;

typedef struct {
    float mechanicalAngleRad; /* 机械角度，单位 rad。 */
    float mechanicalSpeedRadPerSec; /* 机械角速度，单位 rad/s。 */
    float openLoopElectricalSpeedRadPerSec; /* 开环电角速度，单位 rad/s。 */
    float fixedElectricalAngleRad; /* 固定电角度，单位 rad。 */
    float alignElectricalRad; /* 校准时指定的电角度，单位 rad。 */
} FocAngleInput;

typedef struct {
    float electricalAngleRad; /* 电角度，范围 0~2pi，单位 rad。 */
    float electricalSpeedRadPerSec; /* 电角速度，单位 rad/s。 */
    float openLoopElectricalAngleRad; /* 开环积分电角度，范围 0~2pi，单位 rad。 */
    FocSinCos sincos; /* 电角度正弦和余弦。 */
    bool calibrated; /* 是否已执行零位校准。 */
    bool valid; /* 输出是否有效。 */
} FocAngleState;

typedef struct {
    FocAngleConfig config; /* 电角度配置。 */
    FocAngleInput input; /* 电角度输入。 */
    FocAngleState state; /* 电角度状态和输出。 */
} FocAngle;


/**
 * @brief 初始化电角度对象。
 * @param[out] angle 电角度对象。
 * @return true 表示初始化成功，false 表示对象为空。
 *
 * 初始化内容：清零输入和状态，默认 SENSOR 模式，极对数为 1，方向为正向，
 * 角度更新频率使用 FOC_ANGLE_DEFAULT_CONTROL_FREQ_HZ。
 */
bool foc_angle_init(FocAngle *angle);

/**
 * @brief 包装到 0~2pi。
 * @param angleRad 输入角度，单位 rad。
 * @return 包装后的角度，单位 rad。
 */
float foc_wrap_0_2pi(float angleRad);

/**
 * @brief 包装到 -pi~pi。
 * @param angleRad 输入角度，单位 rad。
 * @return 包装后的角度，单位 rad。
 */
float foc_wrap_pm_pi(float angleRad);

/**
 * @brief 设置开环积分角度。
 * @param angle 电角度对象。
 * @param electricalAngleRad 需要写入的电角度，单位 rad。
 * @return true 表示设置成功，false 表示参数无效。
 */
bool foc_angle_set_open_loop_phase(FocAngle *angle, float electricalAngleRad);

/**
 * @brief 根据当前机械角度计算电角度零位偏置。
 * @param angle 电角度对象。
 * @return true 表示校准成功，false 表示参数无效。
 */
bool foc_angle_calibrate(FocAngle *angle);

/**
 * @brief 更新一次电角度。
 *
 * 闭环角度模式：机械角度 -> 电角度；开环角度模式：电角速度积分 -> 电角度；
 * 固定角度模式：直接使用固定电角度。三种模式都会输出 0~2pi 电角度和 sin/cos。
 * @param angle 电角度对象。
 * @return true 表示输出有效，false 表示参数无效。
 */
bool foc_angle_update(FocAngle *angle);

#ifdef __cplusplus
}
#endif

#endif
