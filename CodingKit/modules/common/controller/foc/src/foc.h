#ifndef FOC_H
#define FOC_H

/**
 * @file foc.h
 * @brief 通用 FOC 控制单元。
 * @author HU JIAXUAN
 *
 * 本模块把 dq 电压/电流指令、电角度、母线电压和可选三相电流转换为三相占空比。
 * Park、Clarke、SVPWM、电角度计算均拆成独立模块，可单独复用。
 * 本模块不绑定 ADC、PWM、编码器、电机对象、驱动对象或通信协议。
 *
 * 开环/闭环被拆成两层：
 * 1. 电角度开环或闭环由 foc_angle 模块负责；
 * 2. FOC 主单元只区分开环电压模式和闭环电流模式。
 *
 * 若使用标幺，电流、电压、调制量和增益必须由上层按统一基值归一化。
 * 三相零电流偏置由 foc_current_offset_* 接口维护，foc() 内部会先扣除偏置再做 Clarke。
 * 可选外环由 foc_motion 模块实现，包含位置环、速度环、输入滤波、前馈和齿槽补偿。
 */

#include <stdbool.h>
#include <stdint.h>

#include "foc_angle.h"
#include "foc_math.h"
#include "foc_motion.h"
#include "foc_svpwm.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    FOC_CONTROL_MODE_OPEN_VOLTAGE = 0,   /* 开环电压：直接使用 dq 电压指令。 */
    FOC_CONTROL_MODE_CLOSED_CURRENT = 1, /* 闭环电流：使用三相电流反馈执行 dq PI。 */
    FOC_CONTROL_MODE_MOTION_CURRENT = 2  /* 外环级联：位置/速度/输入整形输出 dq 电流，再进入电流环。 */
} FocControlMode;

#ifndef FOC_DEFAULT_CONTROL_FREQ_HZ
#define FOC_DEFAULT_CONTROL_FREQ_HZ (10000.0f)
#endif

typedef struct {
    FocControlMode controlMode; /* FOC 控制模式。 */
    float currentKp; /* 电流环比例增益，单位 V/A。 */
    float currentKi; /* 电流环积分增益，单位 V/(A*s)。 */
    float controlFreq; /* 控制频率，单位 Hz。 */
    float maxVoltage; /* dq 输出电压限幅，单位 V；不用时填 0。 */
    float modulationLimit; /* 调制矢量限幅，无单位；建议不超过 sqrt(3)/2。 */
    float integratorDecay; /* 饱和时积分衰减系数，无单位，范围 0~1。 */
    bool enableMotionControl; /* 是否启用内置外环控制器。 */
} FocConfig;

typedef struct {
    float vbusVoltage; /* 母线电压，单位 V。 */
    FocPhase phaseCurrent; /* 三相电流，单位 A；闭环电流模式使用。 */
    float electricalAngleRad; /* 电角度，单位 rad。 */
    FocDq currentSetpoint; /* dq 电流指令，单位 A；闭环电流模式使用。 */
    FocDq voltageFeedforward; /* dq 电压指令或前馈，单位 V。 */
} FocInput;

typedef struct {
    FocPhase phaseCurrentCorrected; /* 扣除零电流偏置后的三相电流，单位 A。 */
    FocPhase currentOffset; /* 三相零电流偏置，单位 A。 */
    uint32_t offsetSampleCount; /* 偏置累计样本数，无单位。 */
    bool offsetValid; /* 零电流偏置是否有效。 */
    FocAb currentAb; /* alpha-beta 电流，单位 A。 */
    FocDq currentDq; /* dq 电流，单位 A。 */
    FocDq currentError; /* dq 电流误差，单位 A。 */
    FocDq voltageDq; /* dq 输出电压，单位 V。 */
    FocAb voltageAb; /* alpha-beta 输出电压，单位 V。 */
    FocAb modulation; /* alpha-beta 调制量，无单位。 */
    float dutyA; /* A 相占空比，无单位，范围 0~1。 */
    float dutyB; /* B 相占空比，无单位，范围 0~1。 */
    float dutyC; /* C 相占空比，无单位，范围 0~1。 */
    float integralD; /* d 轴电流积分电压，单位 V。 */
    float integralQ; /* q 轴电流积分电压，单位 V。 */
    bool saturated; /* 输出是否发生限幅或调制饱和。 */
    bool valid; /* 输出是否有效。 */
} FocState;

typedef struct {
    FocConfig config; /* FOC 配置。 */
    FocInput input; /* FOC 输入。 */
    FocMotion motion; /* 可选外环控制器，用于位置环、速度环、输入滤波、前馈和齿槽补偿。 */
    FocState state; /* FOC 状态和输出。 */
} Foc;


/**
 * @brief 初始化 FOC 控制器对象。
 * @param[out] controller FOC 控制对象。
 * @return true 表示初始化成功，false 表示对象为空。
 *
 * 初始化内容：清零输入和状态，默认开环电压模式，设置安全默认控制频率、
 * 调制上限和积分衰减系数，三相零电流偏置默认为 0。实际电流环参数和母线
 * 电压仍应由工程按硬件覆盖。
 */
bool foc_init(Foc *controller);

/**
 * @brief 清除三相零电流偏置累计结果。
 * @param[in,out] controller FOC 控制对象。
 * @return true 表示清除成功，false 表示对象为空。
 */
bool foc_current_offset_clear(Foc *controller);

/**
 * @brief 累计一组三相零电流采样，用于估计电流采样偏置。
 * @param[in,out] controller FOC 控制对象。
 * @param[in] sample 零电流条件下读取到的三相电流样本，单位 A。
 * @return true 表示累计成功，false 表示对象为空。
 *
 * 使用要求：调用期间功率输出应关闭或确认实际相电流为 0。函数使用递推平均，
 * 不分配内存，不绑定 ADC，由上层提供已经换算成 A 的采样值。
 */
bool foc_current_offset_accumulate(Foc *controller, FocPhase sample);

/**
 * @brief 直接设置三相零电流偏置。
 * @param[in,out] controller FOC 控制对象。
 * @param[in] offset 三相零电流偏置，单位 A。
 * @return true 表示设置成功，false 表示对象为空。
 */
bool foc_current_offset_set(Foc *controller, FocPhase offset);

/**
 * @brief 执行一次 FOC 控制更新。
 *
 * 开环电压模式：dq 电压指令 -> 反 Park -> SVPWM，适合开环启动、对齐或外部控制器输出 vd/vq。
 * 闭环电流模式：三相电流 -> Clarke/Park -> dq PI -> 反 Park -> SVPWM，适合电流闭环。
 * 外环级联模式：foc_motion 先生成 dq 电流指令，再进入闭环电流模式。
 * @param controller FOC 控制对象。
 * @return true 表示输出有效，false 表示参数无效。
 */
bool foc(Foc *controller);

#ifdef __cplusplus
}
#endif

#endif
