/**
 * @file foc_motion.c
 * @brief FOC 外环级联控制、输入整形、前馈和齿槽补偿实现。
 * @author HU JIAXUAN
 */

#include "foc_motion.h"

#include <math.h>

/**
 * @brief 获取安全控制周期。
 * @param[in] controlFreq 控制频率，单位 Hz。
 * @return 控制周期，单位 s；参数非法时返回 0。
 */
static float foc_motion_period(float controlFreq)
{
    if (controlFreq <= 0.0f) {
        return 0.0f;
    }

    return 1.0f / controlFreq;
}

/**
 * @brief 获取绝对值。
 * @param[in] value 输入值。
 * @return 绝对值。
 */
static float foc_motion_abs(float value)
{
    return (value >= 0.0f) ? value : -value;
}

/**
 * @brief 按最大变化量逼近目标值。
 * @param[in] current 当前值。
 * @param[in] target 目标值。
 * @param[in] maxStep 单周期最大变化量。
 * @return 更新后的值。
 */
static float foc_motion_step_towards(float current, float target, float maxStep)
{
    float delta;

    if (maxStep <= 0.0f) {
        return target;
    }

    delta = target - current;
    if (delta > maxStep) {
        delta = maxStep;
    } else if (delta < -maxStep) {
        delta = -maxStep;
    }

    return current + delta;
}

/**
 * @brief 处理积分衰减系数。
 * @param[in] value 配置值。
 * @return 限制到 [0, 1] 的衰减系数。
 */
static float foc_motion_safe_decay(float value)
{
    if (value <= 0.0f) {
        value = 0.99f;
    }

    return foc_clamp(value, 0.0f, 1.0f);
}

/**
 * @brief 计算齿槽补偿电流。
 * @param[in] controller 外环控制对象。
 * @return 齿槽补偿电流，单位 A。
 *
 * 实现方法：用当前位置映射到补偿表索引，读取该位置需要的 q 轴电流前馈。
 * 补偿表由上层标定并传入，本模块不负责标定过程，也不分配表内存。
 */
static float foc_motion_lookup_anticogging(const FocMotion *controller)
{
    float scale;
    float tablePos;
    int32_t index;
    int32_t length;

    if ((controller == 0) || (!controller->config.enableAntiCogging) ||
        (controller->config.antiCoggingTable == 0) ||
        (controller->config.antiCoggingTableLength == 0U)) {
        return 0.0f;
    }

    length = (int32_t)controller->config.antiCoggingTableLength;
    scale = controller->config.antiCoggingPositionScale;
    if (scale <= 0.0f) {
        scale = (float)controller->config.antiCoggingTableLength;
    }

    tablePos = controller->input.positionFeedback * scale;
    index = (int32_t)floorf(tablePos);
    index %= length;
    if (index < 0) {
        index += length;
    }

    return controller->config.antiCoggingTable[index];
}

/**
 * @brief 按速度限制 q 轴电流。
 * @param[in] velLimit 速度限幅，单位 turn/s。
 * @param[in] velFeedback 当前速度，单位 turn/s。
 * @param[in] velGain 速度环比例增益，单位 A/(turn/s)。
 * @param[in] qCurrent q 轴电流指令，单位 A。
 * @return 速度限制后的 q 轴电流。
 */
static float foc_motion_limit_current_by_velocity(float velLimit,
                                                  float velFeedback,
                                                  float velGain,
                                                  float qCurrent)
{
    float currentMax;
    float currentMin;

    if ((velLimit <= 0.0f) || (velGain <= 0.0f)) {
        return qCurrent;
    }

    currentMax = (velLimit - velFeedback) * velGain;
    currentMin = (-velLimit - velFeedback) * velGain;

    return foc_clamp(qCurrent, currentMin, currentMax);
}

/**
 * @brief 限制 q 轴电流幅值。
 * @param[in] qCurrent 输入 q 轴电流，单位 A。
 * @param[in] currentLimit 电流限幅，单位 A。
 * @param[out] limited 限幅标志，可为空。
 * @return 限幅后的 q 轴电流。
 */
static float foc_motion_limit_current(float qCurrent, float currentLimit, bool *limited)
{
    float result = qCurrent;

    if (currentLimit <= 0.0f) {
        return result;
    }

    result = foc_clamp(qCurrent, -currentLimit, currentLimit);
    if ((result != qCurrent) && (limited != 0)) {
        *limited = true;
    }

    return result;
}

bool foc_motion_init(FocMotion *controller)
{
    if (controller == 0) {
        return false;
    }

    *controller = (FocMotion){0};
    controller->config.controlMode = FOC_MOTION_CONTROL_CURRENT;
    controller->config.inputMode = FOC_MOTION_INPUT_PASSTHROUGH;
    controller->config.controlFreq = FOC_MOTION_DEFAULT_CONTROL_FREQ_HZ;
    controller->config.inputFilterBandwidth = FOC_MOTION_DEFAULT_INPUT_FILTER_BW;
    controller->config.integratorDecay = 0.99f;
    controller->config.enableVelLimit = true;
    controller->config.enableCurrentLimit = true;
    controller->config.enableCurrentModeVelLimit = true;
    (void)foc_motion_update_filter(controller);

    return true;
}

bool foc_motion_reset(FocMotion *controller)
{
    FocMotionConfig config;
    float inputFilterKp;
    float inputFilterKi;

    if (controller == 0) {
        return false;
    }

    config = controller->config;
    inputFilterKp = controller->inputFilterKp;
    inputFilterKi = controller->inputFilterKi;
    controller->input = (FocMotionInput){0};
    controller->state = (FocMotionState){0};
    controller->config = config;
    controller->inputFilterKp = inputFilterKp;
    controller->inputFilterKi = inputFilterKi;

    return true;
}

bool foc_motion_update_filter(FocMotion *controller)
{
    float bandwidth;
    float maxBandwidth;
    float inputKi;

    if ((controller == 0) || (controller->config.controlFreq <= 0.0f)) {
        return false;
    }

    bandwidth = controller->config.inputFilterBandwidth;
    if (bandwidth <= 0.0f) {
        controller->inputFilterKp = 0.0f;
        controller->inputFilterKi = 0.0f;
        return true;
    }

    maxBandwidth = 0.25f * controller->config.controlFreq;
    if (bandwidth > maxBandwidth) {
        bandwidth = maxBandwidth;
    }

    inputKi = 2.0f * bandwidth;
    controller->inputFilterKi = inputKi;
    controller->inputFilterKp = 0.25f * inputKi * inputKi;

    return true;
}

/**
 * @brief 执行输入整形。
 * @param[in,out] controller 外环控制对象。
 * @param[in] period 控制周期，单位 s。
 * @return true 表示输入处理成功。
 */
static bool foc_motion_update_input(FocMotion *controller, float period)
{
    float maxStep;
    float oldVelocity;
    float deltaPosition;
    float deltaVelocity;
    float acceleration;

    switch (controller->config.inputMode) {
    case FOC_MOTION_INPUT_INACTIVE:
        break;

    case FOC_MOTION_INPUT_PASSTHROUGH:
        controller->state.positionSetpoint = controller->input.positionSetpoint;
        controller->state.velocitySetpoint = controller->input.velocitySetpoint;
        controller->state.qCurrentSetpoint = controller->input.qCurrentFeedforward;
        controller->state.accelerationFeedforwardCurrent = 0.0f;
        break;

    case FOC_MOTION_INPUT_VEL_RAMP:
        oldVelocity = controller->state.velocitySetpoint;
        maxStep = foc_motion_abs(controller->config.velRampRate * period);
        controller->state.velocitySetpoint = foc_motion_step_towards(oldVelocity,
                                                                     controller->input.velocitySetpoint,
                                                                     maxStep);
        acceleration = (controller->state.velocitySetpoint - oldVelocity) / period;
        controller->state.accelerationFeedforwardCurrent = acceleration * controller->config.inertiaFeedforwardGain;
        controller->state.qCurrentSetpoint = controller->input.qCurrentFeedforward +
                                             controller->state.accelerationFeedforwardCurrent;
        controller->state.positionSetpoint += controller->state.velocitySetpoint * period;
        break;

    case FOC_MOTION_INPUT_CURRENT_RAMP:
        maxStep = foc_motion_abs(controller->config.currentRampRate * period);
        controller->state.qCurrentSetpoint = foc_motion_step_towards(controller->state.qCurrentSetpoint,
                                                                     controller->input.qCurrentFeedforward,
                                                                     maxStep);
        controller->state.positionSetpoint = controller->input.positionSetpoint;
        controller->state.velocitySetpoint = controller->input.velocitySetpoint;
        controller->state.accelerationFeedforwardCurrent = 0.0f;
        break;

    case FOC_MOTION_INPUT_POS_FILTER:
        deltaPosition = controller->input.positionSetpoint - controller->state.positionSetpoint;
        deltaVelocity = controller->input.velocitySetpoint - controller->state.velocitySetpoint;
        acceleration = controller->inputFilterKp * deltaPosition + controller->inputFilterKi * deltaVelocity;
        controller->state.velocitySetpoint += period * acceleration;
        controller->state.positionSetpoint += period * controller->state.velocitySetpoint;
        controller->state.accelerationFeedforwardCurrent = acceleration * controller->config.inertiaFeedforwardGain;
        controller->state.qCurrentSetpoint = controller->input.qCurrentFeedforward +
                                             controller->state.accelerationFeedforwardCurrent;
        break;

    default:
        controller->state.valid = false;
        return false;
    }

    return true;
}

bool foc_motion_update(FocMotion *controller)
{
    float period;
    float velocityCommand;
    float qCurrent;
    float qCurrentBeforeLimit;
    bool limited = false;
    float decay;

    if (controller == 0) {
        return false;
    }

    period = foc_motion_period(controller->config.controlFreq);
    if (period <= 0.0f) {
        controller->state.valid = false;
        return false;
    }

    if (!foc_motion_update_input(controller, period)) {
        return false;
    }

    velocityCommand = controller->state.velocitySetpoint;
    controller->state.positionError = 0.0f;
    controller->state.velocityError = 0.0f;
    controller->state.antiCoggingCurrent = foc_motion_lookup_anticogging(controller);

    if (controller->config.controlMode >= FOC_MOTION_CONTROL_POSITION) {
        controller->state.positionError = controller->state.positionSetpoint - controller->input.positionFeedback;
        velocityCommand += controller->config.posGain * controller->state.positionError;
    }

    if (controller->config.enableVelLimit && (controller->config.velLimit > 0.0f)) {
        velocityCommand = foc_clamp(velocityCommand, -controller->config.velLimit, controller->config.velLimit);
    }

    qCurrent = controller->state.qCurrentSetpoint;
    if (controller->config.controlMode >= FOC_MOTION_CONTROL_VELOCITY) {
        controller->state.velocityError = velocityCommand - controller->input.velocityFeedback;
        qCurrent += controller->config.velGain * controller->state.velocityError;
        qCurrent += controller->state.velocityIntegratorCurrent;
    } else if (controller->config.enableCurrentModeVelLimit) {
        qCurrent = foc_motion_limit_current_by_velocity(controller->config.velLimit,
                                                        controller->input.velocityFeedback,
                                                        controller->config.velGain,
                                                        qCurrent);
    }

    qCurrent += controller->state.antiCoggingCurrent;
    qCurrentBeforeLimit = qCurrent;

    if (controller->config.enableCurrentLimit) {
        qCurrent = foc_motion_limit_current(qCurrent, controller->config.currentLimit, &limited);
    }

    if ((controller->config.controlMode >= FOC_MOTION_CONTROL_VELOCITY) && (controller->config.velIntegratorGain != 0.0f)) {
        if (limited || (qCurrent != qCurrentBeforeLimit)) {
            decay = foc_motion_safe_decay(controller->config.integratorDecay);
            controller->state.velocityIntegratorCurrent *= decay;
        } else {
            controller->state.velocityIntegratorCurrent += controller->config.velIntegratorGain *
                                                          controller->state.velocityError * period;
        }

        if (controller->config.velIntegratorLimit > 0.0f) {
            controller->state.velocityIntegratorCurrent = foc_clamp(controller->state.velocityIntegratorCurrent,
                                                                    -controller->config.velIntegratorLimit,
                                                                    controller->config.velIntegratorLimit);
        }
    } else {
        controller->state.velocityIntegratorCurrent = 0.0f;
    }

    controller->state.currentDq.d = controller->input.dCurrentFeedforward;
    controller->state.currentDq.q = qCurrent;
    controller->state.saturated = limited || (qCurrent != qCurrentBeforeLimit);
    controller->state.valid = true;

    return true;
}
