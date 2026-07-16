/**
 * @file foc.c
 * @brief VF/IF FOC 控制器实现。
 * @author HU JIAXUAN
 *
 */

#include "foc.h"
#include "pid.h"

#include <math.h>
#include <stdint.h>

static float foc_abs(float value);
static float foc_sign(float value);
static float foc_wrap_angle(float angle);
static void foc_sync_pid_freq(Foc *controller);
static float foc_pid_error(Pid *controller, float error);
static FocDq foc_limit_voltage(FocDq voltage, float maxVoltage, bool *saturated);
static void foc_update_open_loop_angle(Foc *controller);
static FocPhase foc_correct_phase_current(const Foc *controller);
static void foc_update_vf(Foc *controller);
static void foc_update_if(Foc *controller);
static bool foc_update_svpwm(Foc *controller);

/**
 * @brief 计算浮点绝对值。
 * @param[in] value 输入值，单位由调用方决定。
 * @return 非负绝对值，单位同输入。
 */
static float foc_abs(float value)
{
    if (value < 0.0f) {
        return -value;
    }

    return value;
}

/**
 * @brief 计算浮点符号。
 * @param[in] value 输入值，单位由调用方决定。
 * @return 正数返回 1，负数返回 -1，零返回 0；无单位。
 */
static float foc_sign(float value)
{
    if (value > 0.0f) {
        return 1.0f;
    }
    if (value < 0.0f) {
        return -1.0f;
    }

    return 0.0f;
}

/**
 * @brief 将电角度包装到 [0, 2*pi) 区间。
 * @param[in] angle 输入角度，单位 rad。
 * @return 包装后的角度，单位 rad。
 */
static float foc_wrap_angle(float angle)
{
    while (angle >= FOC_TWO_PI) {
        angle -= FOC_TWO_PI;
    }
    while (angle < 0.0f) {
        angle += FOC_TWO_PI;
    }

    return angle;
}

/**
 * @brief 将 FOC 控制频率同步到四个 PID 控制器。
 * @param[in,out] controller FOC 控制器对象；调用方保证非空。
 */
static void foc_sync_pid_freq(Foc *controller)
{
    controller->pid_pos.config.controlFreq = controller->control_freq;
    controller->pid_vel.config.controlFreq = controller->control_freq;
    controller->pid_id.config.controlFreq = controller->control_freq;
    controller->pid_iq.config.controlFreq = controller->control_freq;
}

/**
 * @brief 用 error-only 方式调用通用 PID 控制器。
 * @param[in,out] controller PID 控制器对象；为空时输出 0。
 * @param[in] error 控制误差，单位由具体环路决定。
 * @return PID 输出，单位由具体环路决定。
 */
static float foc_pid_error(Pid *controller, float error)
{
    if (controller == 0) {
        return 0.0f;
    }

    controller->input.setpoint = error;
    controller->input.feedback = 0.0f;
    controller->input.feedforward = 0.0f;

    return pid(controller);
}

/**
 * @brief 对 dq 电压矢量做幅值限制。
 * @param[in] voltage 输入 dq 电压，单位 V。
 * @param[in] maxVoltage 最大电压幅值，单位 V；小于等于 0 表示不限制。
 * @param[out] saturated 发生限幅时写 true；允许为空。
 * @return 限幅后的 dq 电压，单位 V。
 */
static FocDq foc_limit_voltage(FocDq voltage, float maxVoltage, bool *saturated)
{
    FocDq result = voltage;
    float magnitude;
    float scale;

    if (maxVoltage <= 0.0f) {
        return result;
    }

    magnitude = sqrtf((voltage.d * voltage.d) + (voltage.q * voltage.q));
    if (magnitude > maxVoltage) {
        scale = maxVoltage / magnitude;
        result.d *= scale;
        result.q *= scale;
        if (saturated != 0) {
            *saturated = true;
        }
    }

    return result;
}

/**
 * @brief 按开环频率积分更新电角度和电角速度。
 * @param[in,out] controller FOC 控制器对象；调用方保证非空且 control_freq 大于 0。
 */
static void foc_update_open_loop_angle(Foc *controller)
{
    const float angleStep = controller->dir * FOC_TWO_PI * controller->open_loop_freq_hz / controller->control_freq;

    controller->theta_e = foc_wrap_angle(controller->theta_e + angleStep);
    controller->omega_e = controller->dir * FOC_TWO_PI * controller->open_loop_freq_hz;
}

/**
 * @brief 扣除三相电流 offset。
 * @param[in] controller FOC 控制器对象；调用方保证非空。
 * @return offset 校正后的三相电流，单位 A。
 */
static FocPhase foc_correct_phase_current(const Foc *controller)
{
    FocPhase result;

    result.a = controller->ia - controller->ia_offset;
    result.b = controller->ib - controller->ib_offset;
    result.c = controller->ic - controller->ic_offset;

    return result;
}

/**
 * @brief 更新 VF 模式下的 dq 输出电压。
 * @param[in,out] controller FOC 控制器对象；调用方保证非空。
 */
static void foc_update_vf(Foc *controller)
{
    float vfVoltage = 0.0f;

    controller->out_vd = controller->cmd_vd;
    controller->out_vq = controller->cmd_vq;

    if ((controller->vf_gain_v_per_hz != 0.0f) || (controller->vf_boost_v != 0.0f)) {
        vfVoltage = controller->vf_boost_v +
                    (controller->vf_gain_v_per_hz * foc_abs(controller->open_loop_freq_hz));
        if (controller->vf_min_v <= controller->vf_max_v) {
            vfVoltage = foc_clamp(vfVoltage, controller->vf_min_v, controller->vf_max_v);
        }
        controller->out_vq += foc_sign(controller->dir) * vfVoltage;
    }
}

/**
 * @brief 更新 IF 模式下的位置、速度和电流环。
 * @param[in,out] controller FOC 控制器对象；调用方保证非空。
 */
static void foc_update_if(Foc *controller)
{
    bool saturated = false;

    if (controller->enable_pos_loop) {
        controller->cmd_vel = foc_pid_error(&controller->pid_pos, controller->cmd_pos - controller->pos);
        saturated = saturated || controller->pid_pos.state.saturated;
    }

    if (controller->enable_vel_loop) {
        controller->cmd_iq = foc_pid_error(&controller->pid_vel, controller->cmd_vel - controller->vel);
        saturated = saturated || controller->pid_vel.state.saturated;
    }

    if (controller->enable_id_loop) {
        controller->out_vd = foc_pid_error(&controller->pid_id, controller->cmd_id - controller->real_id);
        saturated = saturated || controller->pid_id.state.saturated;
    } else {
        controller->out_vd = controller->cmd_vd;
    }

    if (controller->enable_iq_loop) {
        controller->out_vq = foc_pid_error(&controller->pid_iq, controller->cmd_iq - controller->real_iq);
        saturated = saturated || controller->pid_iq.state.saturated;
    } else {
        controller->out_vq = controller->cmd_vq;
    }

    controller->saturated = controller->saturated || saturated;
}

/**
 * @brief 将 alpha-beta 输出电压转换为三相 SVPWM duty。
 * @param[in,out] controller FOC 控制器对象；调用方保证非空且 vbus 大于 0。
 * @return SVPWM 更新成功返回 true；初始化或计算失败返回 false。
 */
static bool foc_update_svpwm(Foc *controller)
{
    FocSvpwm svpwm;

    if (!foc_svpwm_init(&svpwm)) {
        return false;
    }

    svpwm.config.maxModulation = controller->modulation_limit;
    svpwm.config.enableAutoScale = true;
    svpwm.input.modulation.alpha = controller->out_valpha / controller->vbus;
    svpwm.input.modulation.beta = controller->out_vbeta / controller->vbus;

    if (!foc_svpwm(&svpwm)) {
        return false;
    }

    controller->duty_a = svpwm.state.dutyA;
    controller->duty_b = svpwm.state.dutyB;
    controller->duty_c = svpwm.state.dutyC;
    controller->saturated = controller->saturated || svpwm.state.saturated;

    return true;
}

/**
 * @brief 初始化 FOC 控制器对象。
 * @param[out] controller FOC 控制器对象；不能为空。
 * @return 初始化成功返回 true；controller 为空返回 false。
 */
bool foc_init(Foc *controller)
{
    if (controller == 0) {
        return false;
    }

    *controller = (Foc){0};
    controller->mode = FOC_MODE_IF;
    controller->angle_mode = FOC_ANGLE_SENSOR;
    controller->control_freq = FOC_DEFAULT_CONTROL_FREQ_HZ;
    controller->dir = 1.0f;
    controller->modulation_limit = FOC_SVPWM_DEFAULT_MAX_MODULATION;
    controller->enable_id_loop = true;
    controller->enable_iq_loop = true;
    controller->current_offset_valid = true;

    (void)pid_init(&controller->pid_pos);
    (void)pid_init(&controller->pid_vel);
    (void)pid_init(&controller->pid_id);
    (void)pid_init(&controller->pid_iq);
    foc_sync_pid_freq(controller);

    return true;
}

/**
 * @brief 清除三相电流 offset 估计值。
 * @param[in,out] controller FOC 控制器对象；不能为空。
 * @return 清除成功返回 true；controller 为空返回 false。
 */
bool foc_current_offset_clear(Foc *controller)
{
    if (controller == 0) {
        return false;
    }

    controller->ia_offset = 0.0f;
    controller->ib_offset = 0.0f;
    controller->ic_offset = 0.0f;
    controller->offset_sample_count = 0U;
    controller->current_offset_valid = false;

    return true;
}

/**
 * @brief 将一组三相电流样本累加到 offset 递推平均值。
 * @param[in,out] controller FOC 控制器对象；不能为空。
 * @param[in] sample 三相电流样本，单位 A。
 * @return 累加成功返回 true；controller 为空返回 false。
 */
bool foc_current_offset_accumulate(Foc *controller, FocPhase sample)
{
    float sampleCount;

    if (controller == 0) {
        return false;
    }

    if (controller->offset_sample_count < UINT32_MAX) {
        controller->offset_sample_count++;
    }

    sampleCount = (float)controller->offset_sample_count;
    controller->ia_offset += (sample.a - controller->ia_offset) / sampleCount;
    controller->ib_offset += (sample.b - controller->ib_offset) / sampleCount;
    controller->ic_offset += (sample.c - controller->ic_offset) / sampleCount;
    controller->current_offset_valid = true;

    return true;
}

/**
 * @brief 直接设置三相电流 offset。
 * @param[in,out] controller FOC 控制器对象；不能为空。
 * @param[in] offset 三相电流 offset，单位 A。
 * @return 设置成功返回 true；controller 为空返回 false。
 */
bool foc_current_offset_set(Foc *controller, FocPhase offset)
{
    if (controller == 0) {
        return false;
    }

    controller->ia_offset = offset.a;
    controller->ib_offset = offset.b;
    controller->ic_offset = offset.c;
    controller->offset_sample_count = 1U;
    controller->current_offset_valid = true;

    return true;
}

/**
 * @brief 执行一次 FOC 控制循环。
 * @param[in,out] controller FOC 控制器对象；不能为空。
 * @return duty 输出有效返回 true；输入非法、模式非法或 SVPWM 更新失败返回 false。
 */
bool foc_loop(Foc *controller)
{
    FocPhase correctedCurrent;
    FocAb currentAb;
    FocDq voltageDq;
    FocSinCos sc;
    bool limited = false;

    if (controller == 0) {
        return false;
    }

    controller->valid = false;
    controller->saturated = false;

    if ((controller->control_freq <= 0.0f) || (controller->vbus <= 0.0f)) {
        return false;
    }

    foc_sync_pid_freq(controller);

    if (controller->angle_mode == FOC_ANGLE_OPEN_LOOP) {
        foc_update_open_loop_angle(controller);
    } else if (controller->angle_mode != FOC_ANGLE_SENSOR) {
        return false;
    }

    sc = foc_sincos(controller->theta_e);

    correctedCurrent = foc_correct_phase_current(controller);
    currentAb = foc_clarke(correctedCurrent);
    controller->real_ialpha = currentAb.alpha;
    controller->real_ibeta = currentAb.beta;

    voltageDq = foc_park(currentAb, sc);
    controller->real_id = voltageDq.d;
    controller->real_iq = voltageDq.q;

    if (controller->mode == FOC_MODE_VF) {
        foc_update_vf(controller);
    } else if (controller->mode == FOC_MODE_IF) {
        foc_update_if(controller);
    } else {
        return false;
    }

    voltageDq.d = controller->out_vd;
    voltageDq.q = controller->out_vq;
    voltageDq = foc_limit_voltage(voltageDq, controller->max_voltage, &limited);
    controller->out_vd = voltageDq.d;
    controller->out_vq = voltageDq.q;
    controller->saturated = controller->saturated || limited;

    currentAb = foc_inv_park(voltageDq, sc);
    controller->out_valpha = currentAb.alpha;
    controller->out_vbeta = currentAb.beta;

    if (!foc_update_svpwm(controller)) {
        return false;
    }

    controller->valid = true;
    return true;
}
