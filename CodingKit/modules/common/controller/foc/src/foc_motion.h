#ifndef FOC_MOTION_H
#define FOC_MOTION_H

/**
 * @file foc_motion.h
 * @brief FOC 外环级联控制、输入整形、前馈和齿槽补偿模块。
 * @author HU JIAXUAN
 *
 * 本模块位于 FOC 电流环之前，用于把位置/速度/电流指令转换为 dq 电流指令。
 * 它不读取编码器、不读取 ADC、不写 PWM，也不绑定电机对象或驱动对象。
 *
 * 推荐控制链：
 * 输入指令 -> 输入模式处理 -> 位置环 -> 速度环 -> 前馈/齿槽补偿 -> dq 电流指令 -> foc()。
 */

#include <stdbool.h>
#include <stdint.h>

#include "foc_math.h"

#ifdef __cplusplus
extern "C" {
#endif

#ifndef FOC_MOTION_DEFAULT_CONTROL_FREQ_HZ
#define FOC_MOTION_DEFAULT_CONTROL_FREQ_HZ (10000.0f)
#endif

#ifndef FOC_MOTION_DEFAULT_INPUT_FILTER_BW
#define FOC_MOTION_DEFAULT_INPUT_FILTER_BW (2.0f)
#endif

typedef enum {
    FOC_MOTION_CONTROL_CURRENT = 0,   /* 电流指令模式：直接输出 q 轴电流。 */
    FOC_MOTION_CONTROL_VELOCITY = 1,  /* 速度闭环：速度误差输出 q 轴电流。 */
    FOC_MOTION_CONTROL_POSITION = 2   /* 位置-速度级联：位置误差先生成速度指令。 */
} FocMotionControlMode;

typedef enum {
    FOC_MOTION_INPUT_INACTIVE = 0,       /* 不更新 setpoint。 */
    FOC_MOTION_INPUT_PASSTHROUGH = 1,    /* 输入直接进入 setpoint。 */
    FOC_MOTION_INPUT_POS_FILTER = 2,     /* 二阶位置输入滤波。 */
    FOC_MOTION_INPUT_VEL_RAMP = 3,       /* 速度指令斜坡。 */
    FOC_MOTION_INPUT_CURRENT_RAMP = 4    /* q 轴电流指令斜坡。 */
} FocMotionInputMode;

typedef struct {
    FocMotionControlMode controlMode; /* 外环控制模式。 */
    FocMotionInputMode inputMode; /* 输入整形模式。 */
    float controlFreq; /* 外环更新频率，单位 Hz。 */
    float posGain; /* 位置环比例增益，单位 (turn/s)/turn。 */
    float velGain; /* 速度环比例增益，单位 A/(turn/s)。 */
    float velIntegratorGain; /* 速度环积分增益，单位 A/((turn/s)*s)。 */
    float velIntegratorLimit; /* 速度积分限幅，单位 A；小于等于 0 表示不启用。 */
    float velLimit; /* 速度限幅，单位 turn/s；小于等于 0 表示不启用。 */
    float currentLimit; /* q 轴电流限幅，单位 A；小于等于 0 表示不启用。 */
    float velRampRate; /* 速度斜坡限制，单位 (turn/s)/s。 */
    float currentRampRate; /* q 轴电流斜坡限制，单位 A/s。 */
    float inertiaFeedforwardGain; /* 加速度前馈增益，单位 A/(turn/s^2)。 */
    float inputFilterBandwidth; /* 二阶输入滤波带宽，单位 1/s。 */
    float integratorDecay; /* 饱和时积分衰减系数，无单位，范围 0~1。 */
    bool enableVelLimit; /* 是否限制速度指令。 */
    bool enableCurrentLimit; /* 是否限制 q 轴电流。 */
    bool enableCurrentModeVelLimit; /* 电流模式下是否按速度限制 q 轴电流。 */
    bool enableAntiCogging; /* 是否启用齿槽电流前馈补偿。 */
    const float *antiCoggingTable; /* 齿槽补偿表，单位 A；由上层提供存储。 */
    uint32_t antiCoggingTableLength; /* 齿槽补偿表长度。 */
    float antiCoggingPositionScale; /* position turn 到表索引的比例；小于等于 0 时默认每转一张表。 */
} FocMotionConfig;

typedef struct {
    float positionSetpoint; /* 位置输入，单位 turn。 */
    float velocitySetpoint; /* 速度输入，单位 turn/s。 */
    float qCurrentFeedforward; /* q 轴电流前馈，单位 A。 */
    float dCurrentFeedforward; /* d 轴电流前馈，单位 A。 */
    float positionFeedback; /* 位置反馈，单位 turn。 */
    float velocityFeedback; /* 速度反馈，单位 turn/s。 */
} FocMotionInput;

typedef struct {
    float positionSetpoint; /* 输入整形后的内部位置指令，单位 turn。 */
    float velocitySetpoint; /* 输入整形后的内部速度指令，单位 turn/s。 */
    float qCurrentSetpoint; /* 输入整形后的 q 轴基础电流，单位 A。 */
    float positionError; /* 位置误差，单位 turn。 */
    float velocityError; /* 速度误差，单位 turn/s。 */
    float velocityIntegratorCurrent; /* 速度积分电流，单位 A。 */
    float accelerationFeedforwardCurrent; /* 加速度前馈电流，单位 A。 */
    float antiCoggingCurrent; /* 齿槽补偿电流，单位 A。 */
    FocDq currentDq; /* 输出 dq 电流指令，单位 A。 */
    bool saturated; /* 输出或积分是否发生限幅。 */
    bool valid; /* 输出是否有效。 */
} FocMotionState;

typedef struct {
    FocMotionConfig config; /* 外环配置。 */
    FocMotionInput input; /* 外环输入。 */
    FocMotionState state; /* 外环状态和输出。 */
    float inputFilterKp; /* 二阶输入滤波内部比例系数。 */
    float inputFilterKi; /* 二阶输入滤波内部阻尼系数。 */
} FocMotion;

/**
 * @brief 初始化外环控制器对象。
 * @param[out] controller 外环控制对象。
 * @return true 表示初始化成功，false 表示对象为空。
 */
bool foc_motion_init(FocMotion *controller);

/**
 * @brief 重置外环动态状态，保留配置。
 * @param[in,out] controller 外环控制对象。
 * @return true 表示重置成功，false 表示对象为空。
 */
bool foc_motion_reset(FocMotion *controller);

/**
 * @brief 根据输入滤波带宽更新二阶滤波系数。
 * @param[in,out] controller 外环控制对象。
 * @return true 表示更新成功，false 表示对象为空或控制频率非法。
 */
bool foc_motion_update_filter(FocMotion *controller);

/**
 * @brief 执行一次外环控制更新。
 * @param[in,out] controller 外环控制对象。
 * @return true 表示输出 dq 电流指令有效，false 表示参数非法。
 *
 * 控制链：
 * 1. 根据输入模式更新 position/velocity/current setpoint；
 * 2. 位置模式下用位置误差生成速度指令；
 * 3. 速度模式下用速度误差生成 q 轴电流；
 * 4. 叠加速度积分、加速度前馈、q 轴电流前馈和齿槽补偿；
 * 5. 执行速度/电流限幅和积分抗饱和；
 * 6. 输出 dq 电流指令给 FOC 电流环。
 */
bool foc_motion_update(FocMotion *controller);

#ifdef __cplusplus
}
#endif

#endif
