/**
 * @file foc.c
 * @brief 通用 FOC 控制单元实现。
 * @author HU JIAXUAN
 */

#include "foc.h"

#include <math.h>
#include <stdint.h>


/**
 * @brief 初始化 FOC 控制器对象。
 * @param[out] controller FOC 控制对象。
 * @return true 表示初始化成功，false 表示对象为空。
 *
 * 实现方法：清零整个对象，写入开环电压模式、默认控制频率、SVPWM 调制上限、
 * 积分衰减默认值和零电流偏置状态。该函数不读取硬件，也不替代工程参数配置。
 */
bool foc_init(Foc *controller)
{
    if (controller == 0) {
        return false;
    }

    *controller = (Foc){0};
    controller->config.controlMode = FOC_CONTROL_MODE_OPEN_VOLTAGE;
    controller->config.controlFreq = FOC_DEFAULT_CONTROL_FREQ_HZ;
    controller->config.modulationLimit = FOC_SVPWM_DEFAULT_MAX_MODULATION;
    controller->config.integratorDecay = 0.99f;
    controller->config.enableMotionControl = false;
    (void)foc_motion_init(&controller->motion);
    controller->motion.config.controlFreq = controller->config.controlFreq;
    controller->state.valid = false;
    controller->state.offsetValid = true;

    return true;
}

/**
 * @brief 清除三相零电流偏置。
 * @param[in,out] controller FOC 控制对象。
 * @return true 表示清除成功，false 表示对象为空。
 */
bool foc_current_offset_clear(Foc *controller)
{
    if (controller == 0) {
        return false;
    }

    controller->state.currentOffset = (FocPhase){0};
    controller->state.offsetSampleCount = 0U;
    controller->state.offsetValid = false;

    return true;
}

/**
 * @brief 用递推平均累计三相零电流偏置。
 * @param[in,out] controller FOC 控制对象。
 * @param[in] sample 零电流条件下读取到的三相电流样本，单位 A。
 * @return true 表示累计成功，false 表示对象为空。
 *
 * 实现方法：offset(k) = offset(k-1) + [sample - offset(k-1)] / k。
 * 该写法不保存历史数组，适合启动阶段多次采样平均。
 */
bool foc_current_offset_accumulate(Foc *controller, FocPhase sample)
{
    float sampleCount;

    if (controller == 0) {
        return false;
    }

    if (controller->state.offsetSampleCount < UINT32_MAX) {
        controller->state.offsetSampleCount++;
    }
    sampleCount = (float)controller->state.offsetSampleCount;

    controller->state.currentOffset.a += (sample.a - controller->state.currentOffset.a) / sampleCount;
    controller->state.currentOffset.b += (sample.b - controller->state.currentOffset.b) / sampleCount;
    controller->state.currentOffset.c += (sample.c - controller->state.currentOffset.c) / sampleCount;
    controller->state.offsetValid = true;

    return true;
}

/**
 * @brief 直接写入三相零电流偏置。
 * @param[in,out] controller FOC 控制对象。
 * @param[in] offset 三相零电流偏置，单位 A。
 * @return true 表示设置成功，false 表示对象为空。
 */
bool foc_current_offset_set(Foc *controller, FocPhase offset)
{
    if (controller == 0) {
        return false;
    }

    controller->state.currentOffset = offset;
    controller->state.offsetSampleCount = 1U;
    controller->state.offsetValid = true;

    return true;
}

/**
 * @brief 将控制频率转换为控制周期。
 * @param[in] controlFreq 控制频率，单位 Hz。
 * @return 控制周期，单位 s；频率非法时返回 0。
 */
static float foc_safe_control_period(float controlFreq)
{
    if (controlFreq <= 0.0f) {
        return 0.0f;
    }

    return 1.0f / controlFreq;
}

/**
 * @brief 检查母线电压是否可用于调制归一化。
 * @param[in] vbusVoltage 母线电压，单位 V。
 * @return 有效母线电压，单位 V；非法时返回 0。
 */
static float foc_safe_vbus(float vbusVoltage)
{
    if (vbusVoltage <= 0.0f) {
        return 0.0f;
    }

    return vbusVoltage;
}


/**
 * @brief 扣除三相零电流偏置。
 * @param[in] phaseCurrent 原始三相电流，单位 A。
 * @param[in] offset 三相零电流偏置，单位 A。
 * @return 扣除偏置后的三相电流，单位 A。
 *
 * 实现依据：三相电流采样链路通常存在 ADC 零点、运放偏置或换算偏差。
 * 在确认实际相电流为 0 时累计偏置，控制运行时先扣除偏置，再进入 Clarke/Park，
 * 可以避免零点误差直接变成 dq 直流误差。
 */
static FocPhase foc_apply_current_offset(FocPhase phaseCurrent, FocPhase offset)
{
    FocPhase result;

    result.a = phaseCurrent.a - offset.a;
    result.b = phaseCurrent.b - offset.b;
    result.c = phaseCurrent.c - offset.c;

    return result;
}

/**
 * @brief 处理积分饱和衰减系数。
 * @param[in] integratorDecay 积分衰减系数，无单位。
 * @return 限制到 [0, 1] 的衰减系数；配置非法时返回默认值 0.99。
 */
static float foc_safe_integrator_decay(float integratorDecay)
{
    float result = integratorDecay;

    if (result <= 0.0f) {
        result = 0.99f;
    }

    return foc_clamp(result, 0.0f, 1.0f);
}

/**
 * @brief 限制 dq 电压矢量幅值。
 * @param[in] voltage 输入 dq 电压，单位 V。
 * @param[in] maxVoltage 最大电压矢量幅值，单位 V；小于等于 0 时不限制。
 * @param[out] saturated 饱和标志；允许为空指针。
 * @return 限幅后的 dq 电压，单位 V。
 */
static FocDq foc_limit_voltage(FocDq voltage, float maxVoltage, bool *saturated)
{
    float mag;
    float scale;
    FocDq result = voltage;

    if (maxVoltage <= 0.0f) {
        return result;
    }

    mag = sqrtf(voltage.d * voltage.d + voltage.q * voltage.q);
    if (mag > maxVoltage) {
        scale = maxVoltage / mag;
        result.d *= scale;
        result.q *= scale;
        if (saturated != 0) {
            *saturated = true;
        }
    }

    return result;
}

/**
 * @brief 执行闭环 dq 电流 PI 控制。
 * @param[in,out] controller FOC 控制器实例，不能为空。
 * @param[in] period 控制周期，单位 s。
 * @return 无。结果写入 controller->state.voltageDq、currentError 和 saturated。
 *
 * 实现方法：用 currentSetpoint - currentDq 得到 d/q 电流误差，PI 输出 d/q 电压。
 * 电压矢量超出 maxVoltage 时按比例限幅，并衰减积分，避免饱和时积分继续增大。
 */
static void foc_update_closed_current(Foc *controller, float period)
{
    FocDq voltageRaw;
    bool saturated = false;
    float decay;

    controller->state.currentError.d = controller->input.currentSetpoint.d - controller->state.currentDq.d;
    controller->state.currentError.q = controller->input.currentSetpoint.q - controller->state.currentDq.q;

    voltageRaw.d = controller->input.voltageFeedforward.d + controller->state.integralD +
                   controller->config.currentKp * controller->state.currentError.d;
    voltageRaw.q = controller->input.voltageFeedforward.q + controller->state.integralQ +
                   controller->config.currentKp * controller->state.currentError.q;

    controller->state.voltageDq = foc_limit_voltage(voltageRaw, controller->config.maxVoltage, &saturated);
    controller->state.saturated = saturated;

    if (saturated) {
        decay = foc_safe_integrator_decay(controller->config.integratorDecay);
        controller->state.integralD *= decay;
        controller->state.integralQ *= decay;
    } else {
        controller->state.integralD += controller->config.currentKi * controller->state.currentError.d * period;
        controller->state.integralQ += controller->config.currentKi * controller->state.currentError.q * period;
    }
}

/**
 * @brief 执行开环 dq 电压模式。
 * @param[in,out] controller FOC 控制器实例，不能为空。
 * @return 无。结果写入 controller->state.voltageDq 和 saturated。
 *
 * 实现方法：直接把 voltageFeedforward 作为 dq 电压指令，并执行 maxVoltage 限幅。
 * 该模式不需要三相电流反馈，常用于开环启动、转子对齐、扫频测试或外部控制器
 * 已经计算好 vd/vq 的场景。
 */
static void foc_update_open_voltage(Foc *controller)
{
    bool saturated = false;

    controller->state.currentError.d = 0.0f;
    controller->state.currentError.q = 0.0f;
    controller->state.voltageDq = foc_limit_voltage(controller->input.voltageFeedforward,
                                                    controller->config.maxVoltage,
                                                    &saturated);
    controller->state.saturated = saturated;
}


/**
 * @brief 执行可选外环控制器，并把输出 dq 电流送入 FOC 电流环。
 * @param[in,out] controller FOC 控制器实例，不能为空。
 * @return true 表示外环输出有效，false 表示外环配置非法。
 *
 * 实现方法：同步外环控制频率，调用 foc_motion_update() 得到 dq 电流指令，
 * 然后写入 input.currentSetpoint。外环只生成电流指令，不直接参与 Park、SVPWM 或硬件输出。
 */
static bool foc_update_motion_current(Foc *controller)
{
    controller->motion.config.controlFreq = controller->config.controlFreq;
    if (!foc_motion_update(&controller->motion)) {
        controller->state.valid = false;
        return false;
    }

    controller->input.currentSetpoint = controller->motion.state.currentDq;
    return true;
}

/**
 * @brief 执行一次 FOC 主控制更新。
 * @param[in,out] controller FOC 控制器实例；输入来自 controller->input 和 controller->config。
 * @return true 表示 duty 输出有效；false 表示输入非法或调制失败。
 *
 * 主控制链：
 * 1. 检查控制频率和母线电压；
 * 2. 根据电角度计算 sin/cos；
 * 3. 三相电流先扣除零电流偏置，再经 Clarke/Park 得到 dq 电流；
 * 4. 外环级联模式先执行位置/速度/输入滤波/前馈，生成 dq 电流指令；
 * 5. 开环电压模式直接使用 dq 电压指令；闭环电流和外环级联模式执行 dq PI；
 * 6. dq 电压经反 Park 得到 alpha-beta 电压；
 * 7. alpha-beta 电压归一化为调制量；
 * 8. SVPWM 输出三相 duty。
 *
 * 控制依据：FOC 主单元只依赖电角度，不关心该角度来自开环积分、编码器、Hall、
 * resolver 或观测器。开环/闭环角度由 foc_angle 或上层提供；外环级联由 foc_motion 提供；本函数只完成一次
 * 坐标变换、电压生成和调制输出。
 */
bool foc(Foc *controller)
{
    float period;
    float vbus;
    FocSinCos sc;
    FocSvpwm svpwm;

    if (controller == 0) {
        return false;
    }

    period = foc_safe_control_period(controller->config.controlFreq);
    vbus = foc_safe_vbus(controller->input.vbusVoltage);
    if ((period <= 0.0f) || (vbus <= 0.0f)) {
        controller->state.valid = false;
        return false;
    }

    sc = foc_sincos(controller->input.electricalAngleRad);

    controller->state.phaseCurrentCorrected = foc_apply_current_offset(controller->input.phaseCurrent,
                                                                       controller->state.currentOffset);
    controller->state.currentAb = foc_clarke(controller->state.phaseCurrentCorrected);
    controller->state.currentDq = foc_park(controller->state.currentAb, sc);
    controller->state.saturated = false;

    if ((controller->config.controlMode == FOC_CONTROL_MODE_MOTION_CURRENT) ||
        controller->config.enableMotionControl) {
        if (!foc_update_motion_current(controller)) {
            return false;
        }
        foc_update_closed_current(controller, period);
    } else if (controller->config.controlMode == FOC_CONTROL_MODE_CLOSED_CURRENT) {
        foc_update_closed_current(controller, period);
    } else {
        foc_update_open_voltage(controller);
    }

    controller->state.voltageAb = foc_inv_park(controller->state.voltageDq, sc);

    controller->state.modulation.alpha = controller->state.voltageAb.alpha / vbus;
    controller->state.modulation.beta = controller->state.voltageAb.beta / vbus;

    (void)foc_svpwm_init(&svpwm);
    svpwm.config.maxModulation = controller->config.modulationLimit;
    svpwm.config.enableAutoScale = true;
    svpwm.input.modulation = controller->state.modulation;

    if (!foc_svpwm(&svpwm)) {
        controller->state.valid = false;
        return false;
    }

    controller->state.dutyA = svpwm.state.dutyA;
    controller->state.dutyB = svpwm.state.dutyB;
    controller->state.dutyC = svpwm.state.dutyC;
    controller->state.saturated = controller->state.saturated || svpwm.state.saturated;
    controller->state.valid = true;

    return controller->state.valid;
}
