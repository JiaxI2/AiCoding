#include "pid.h"

/**
 * @file pid.c
 * @brief 通用 PID 控制单元实现。
 * @author HU JIAXUAN
 *
 * 本文件实现目标限幅、目标变化率限制、误差调节、微分滤波、
 * 输出限幅、输出死区和积分抗饱和。
 */

/**
 * @brief 计算 float 绝对值，避免在强实时路径中额外依赖 libm。
 * @param[in] value 输入值。
 * @return 绝对值。
 */
static float pid_abs(float value)
{
    /* 只用于死区判断，不改变控制器状态。 */
    if (value < 0.0f) {
        return -value;
    }

    return value;
}

/**
 * @brief 检查限幅参数是否有效。
 * @param[in] limit 限幅参数。
 * @return 参数有效返回 true，否则返回 false。
 */
static bool pid_limit_valid(PidLimit limit)
{
    /* 未启用的限幅不检查上下界，便于用户只配置必要项。 */
    if (!limit.enable) {
        return true;
    }

    return limit.min <= limit.max;
}

/**
 * @brief 根据限幅参数裁剪输入值。
 * @param[in] value 输入值。
 * @param[in] limit 限幅参数。
 * @return 裁剪后的值。
 */
static float pid_clamp(float value, PidLimit limit)
{
    /* 同一个内部函数覆盖目标、反馈、误差、积分和输出限幅。 */
    if (!limit.enable) {
        return value;
    }
    if (value > limit.max) {
        return limit.max;
    }
    if (value < limit.min) {
        return limit.min;
    }

    return value;
}

/**
 * @brief 将微分滤波系数限制在 [0, 1]。
 * @param[in] coef 原始滤波系数。
 * @return 清理后的滤波系数。
 */
static float pid_clean_filter_coef(float coef)
{
    /* 错误滤波系数不应导致控制器状态发散。 */
    if (coef < 0.0f) {
        return 0.0f;
    }
    if (coef > 1.0f) {
        return 1.0f;
    }

    return coef;
}

/**
 * @brief 检查配置是否可用于当前控制更新。
 * @param[in] config 控制器配置。
 * @return PID_OK 表示有效，否则返回错误码。
 */
static PidStatus pid_check_config(const PidConfig *config)
{
    if (config->controlFreq <= PID_EPSILON) {
        return PID_ERR_FREQ;
    }
    if (!pid_limit_valid(config->setpointLimit) ||
        !pid_limit_valid(config->feedbackLimit) ||
        !pid_limit_valid(config->errorLimit) ||
        !pid_limit_valid(config->integralLimit) ||
        !pid_limit_valid(config->outputLimit)) {
        return PID_ERR_LIMIT;
    }
    if (config->setpointRateEnable && (config->setpointRate < 0.0f)) {
        return PID_ERR_LIMIT;
    }
    if (config->antiWindupGain < 0.0f) {
        return PID_ERR_LIMIT;
    }

    return PID_OK;
}

/**
 * @brief 根据控制频率计算控制周期。
 * @param[in] config 控制器配置。
 * @return 控制周期，单位 s。
 */
static float pid_get_period(const PidConfig *config)
{
    /* 用户配置控制频率，内部使用控制周期完成离散积分和微分。 */
    return 1.0f / config->controlFreq;
}

/**
 * @brief 对目标值执行限幅和斜率限制。
 * @param[in] config 控制器配置。
 * @param[in] state 控制器状态。
 * @param[in] period 控制周期，单位 s。
 * @param[in] setpoint 原始目标值，单位同被控量；标幺时为 pu。
 * @return 处理后的目标值，单位同被控量；标幺时为 pu。
 */
static float pid_prepare_setpoint(const PidConfig *config,
                                  const PidState *state,
                                  float period,
                                  float setpoint)
{
    float setpointLimited;

    /* 先做绝对范围限制，再做每拍变化量限制。 */
    setpointLimited = pid_clamp(setpoint, config->setpointLimit);
    if (config->setpointRateEnable && state->initialized) {
        const float maxStep = config->setpointRate * period;
        const float delta = setpointLimited - state->setpointLimited;

        if (delta > maxStep) {
            setpointLimited = state->setpointLimited + maxStep;
        } else if (delta < -maxStep) {
            setpointLimited = state->setpointLimited - maxStep;
        }
    }

    return setpointLimited;
}

/**
 * @brief 根据输出死区处理控制量。
 * @param[in] value 原始输出，单位同输出；标幺时为 pu。
 * @param[in] deadband 死区宽度，单位同输出；标幺时为 pu。
 * @return 死区处理后的输出，单位同输出；标幺时为 pu。
 */
static float pid_apply_deadband(float value, float deadband)
{
    /* 死区放在通道求和之后，避免隐藏 P/I/D 内部状态。 */
    if (deadband <= 0.0f) {
        return value;
    }
    if (pid_abs(value) < deadband) {
        return 0.0f;
    }

    return value;
}

/**
 * @brief 更新可观测状态字段。
 * @param[in,out] state 控制器状态。
 * @param[in] input 控制器输入。
 * @param[in] setpointLimited 限幅后的目标值。
 * @param[in] feedbackLimited 限幅后的反馈值。
 * @param[in] error 当前误差。
 * @param[in] derivative 当前误差微分。
 * @param[in] rawOutput 限幅前输出。
 * @param[in] limitedOutput 限幅后输出。
 */
static void pid_update_state(PidState *state,
                             PidInput input,
                             float setpointLimited,
                             float feedbackLimited,
                             float error,
                             float derivative,
                             float rawOutput,
                             float limitedOutput)
{
    /* 统一在控制周期尾部更新观测量，避免中间状态被外部误读。 */
    state->setpoint = input.setpoint;
    state->setpointLimited = setpointLimited;
    state->feedback = input.feedback;
    state->feedbackLimited = feedbackLimited;
    state->previousError = state->error;
    state->error = error;
    state->derivative = derivative;
    state->rawOutput = rawOutput;
    state->output = limitedOutput;
    state->saturated = (limitedOutput != rawOutput);
    state->initialized = true;
    state->status = PID_OK;
}

/**
 * @brief 初始化 PID 控制器对象。
 * @param[out] controller 控制器对象。
 * @return true 表示初始化成功，false 表示对象为空。
 *
 * 实现方法：清零整个对象，写入默认控制频率、微分滤波系数、抗饱和增益和状态码。
 * 该函数不替代工程参数配置，实际 kp、ki、kd 和限幅仍应由上层设置。
 */
bool pid_init(Pid *controller)
{
    if (controller == (Pid *)0) {
        return false;
    }

    *controller = (Pid){0};
    controller->config.controlFreq = PID_DEFAULT_CONTROL_FREQ_HZ;
    controller->config.derivativeFilterCoef = 0.0f;
    controller->config.antiWindupGain = 1.0f;
    controller->state.status = PID_OK;

    return true;
}

float pid(Pid *controller)
{
    const PidConfig *config;
    PidState *state;
    PidInput input;
    PidStatus status;
    float period;
    float setpointLimited;
    float feedbackLimited;
    float error;
    float derivative;
    float filterCoef;
    float rawOutput;
    float limitedOutput;
    float integralNext;

    if (controller == (Pid *)0) {
        return 0.0f;
    }

    config = &controller->config;
    state = &controller->state;
    input = controller->input;

    status = pid_check_config(config);
    if (status != PID_OK) {
        /* 配置非法时保留上一拍输出，避免故障分支突然给执行器一个新命令。 */
        state->status = status;
        return state->output;
    }

    period = pid_get_period(config);

    /* 目标值、反馈值和误差都在控制器内部统一限幅。 */
    setpointLimited = pid_prepare_setpoint(config, state, period, input.setpoint);
    feedbackLimited = pid_clamp(input.feedback, config->feedbackLimit);
    error = pid_clamp(setpointLimited - feedbackLimited, config->errorLimit);

    /* 第一拍没有历史误差，因此微分项置 0，避免启动尖峰。 */
    derivative = 0.0f;
    if (state->initialized) {
        derivative = (error - state->error) / period;
    }
    filterCoef = pid_clean_filter_coef(config->derivativeFilterCoef);
    state->derivativeFiltered = (filterCoef * state->derivativeFiltered) + ((1.0f - filterCoef) * derivative);

    /* kp、ki、kd 为 0.0f 时，对应通道自然关闭。 */
    state->proportional = config->kp * error;
    state->derivativeTerm = config->kd * state->derivativeFiltered;
    state->feedforward = input.feedforward;

    rawOutput = state->proportional + state->integral + state->derivativeTerm + state->feedforward;
    rawOutput = pid_apply_deadband(rawOutput, config->deadband);
    limitedOutput = pid_clamp(rawOutput, config->outputLimit);

    if (config->ki != 0.0f) {
        /*
         * Back-calculation 积分抗饱和：
         * I(k+1) = I(k) + Ki * [e(k) + Kaw * (u_sat(k) - u_raw(k))] * Ts
         *
         * 等价拓扑：
         * 1. u_raw 为限幅前控制输出；
         * 2. u_sat 为输出限幅后的实际控制输出；
         * 3. u_sat - u_raw 为饱和误差；
         * 4. Kaw 将饱和误差反馈到积分输入。
         *
         * 该结构对应参考模型中的“限幅前输出 uc、限幅后输出 u、uc-u 经增益后从
         * 积分输入中扣除”的反算式抗饱和。PI 和 PID 只要 ki 非 0 都执行；
         * P 和 PD 没有积分项，自动跳过。
         */
        const float antiWindupError = limitedOutput - rawOutput;
        integralNext = state->integral +
                       (config->ki * (error + (config->antiWindupGain * antiWindupError)) * period);
        state->integral = pid_clamp(integralNext, config->integralLimit);
    } else {
        state->integral = 0.0f;
    }

    pid_update_state(state,
                     input,
                     setpointLimited,
                     feedbackLimited,
                     error,
                     derivative,
                     rawOutput,
                     limitedOutput);

    return limitedOutput;
}
