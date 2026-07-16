#ifndef FOC_H
#define FOC_H

/**
 * @file foc.h
 * @brief VF/IF FOC 控制器模板。
 * @author HU JIAXUAN
 *
 * 本模块只负责可复用 FOC 数学链路：相电流 offset 扣除 -> Clarke -> Park ->
 * VF/IF 电压生成 -> 反 Park -> SVPWM duty 生成。
 *
 * 本模块不绑定 ADC、PWM、编码器、Hall、观测器、状态机、故障保护、通信协议
 * 或硬件驱动对象；这些对象由上层应用或平台适配层负责。
 *
 */

#include <stdbool.h>
#include <stdint.h>

#include "foc_math.h"
#include "foc_svpwm.h"
#include "pid.h"

#ifdef __cplusplus
extern "C" {
#endif

#ifndef FOC_DEFAULT_CONTROL_FREQ_HZ
#define FOC_DEFAULT_CONTROL_FREQ_HZ (10000.0f)
#endif

/** @brief FOC 主控制模式。 */
typedef enum {
    FOC_MODE_VF = 0, /* V/f 或直接 dq 电压模式，输出电压经反 Park 和 SVPWM 生成 duty。 */
    FOC_MODE_IF = 1  /* I/f 或闭环电流模式，dq 电流命令经 PID 生成 dq 电压。 */
} FocMode;

/** @brief 电角度来源。 */
typedef enum {
    FOC_ANGLE_SENSOR = 0,    /* 上层提供真实电角度 theta_e，单位 rad。 */
    FOC_ANGLE_OPEN_LOOP = 1  /* 本模块按 open_loop_freq_hz 积分生成 theta_e。 */
} FocAngleMode;

/** @brief FOC 控制器对象。 */
typedef struct {
    FocMode mode; /* FOC 主控制模式，无单位。 */
    FocAngleMode angle_mode; /* 电角度来源，无单位。 */

    float control_freq; /* 控制频率，单位 Hz；必须大于 0。 */

    float vbus; /* 母线电压，单位 V；必须大于 0。 */
    float ia; /* A 相采样电流，单位 A。 */
    float ib; /* B 相采样电流，单位 A。 */
    float ic; /* C 相采样电流，单位 A。 */

    float theta_e; /* 电角度，单位 rad；SENSOR 模式由上层写入，OPEN_LOOP 模式由内部更新。 */
    float omega_e; /* 电角速度，单位 rad/s；OPEN_LOOP 模式由内部更新。 */
    float open_loop_freq_hz; /* 开环电频率命令，单位 Hz。 */
    float dir; /* 方向符号，无单位；正值为正向，负值为反向。 */

    float cmd_pos; /* 位置环目标，建议单位 turn；必须与 pos 使用同一单位。 */
    float pos; /* 位置反馈，建议单位 turn；必须与 cmd_pos 使用同一单位。 */
    float cmd_vel; /* 速度目标，建议单位 turn/s；必须与 vel 使用同一单位。 */
    float vel; /* 速度反馈，建议单位 turn/s；必须与 cmd_vel 使用同一单位。 */

    float cmd_id; /* d 轴电流目标，单位 A。 */
    float cmd_iq; /* q 轴电流目标，单位 A。 */

    float cmd_vd; /* d 轴电压前馈或 VF 电压命令，单位 V。 */
    float cmd_vq; /* q 轴电压前馈或 VF 电压命令，单位 V。 */

    float real_id; /* Park 变换后的 d 轴实际电流，单位 A。 */
    float real_iq; /* Park 变换后的 q 轴实际电流，单位 A。 */
    float real_ialpha; /* Clarke 变换后的 alpha 轴实际电流，单位 A。 */
    float real_ibeta; /* Clarke 变换后的 beta 轴实际电流，单位 A。 */

    float out_vd; /* 本周期输出 d 轴电压，单位 V。 */
    float out_vq; /* 本周期输出 q 轴电压，单位 V。 */
    float out_valpha; /* 反 Park 后 alpha 轴输出电压，单位 V。 */
    float out_vbeta; /* 反 Park 后 beta 轴输出电压，单位 V。 */

    float duty_a; /* A 相 PWM 占空比，无单位，范围 0~1。 */
    float duty_b; /* B 相 PWM 占空比，无单位，范围 0~1。 */
    float duty_c; /* C 相 PWM 占空比，无单位，范围 0~1。 */

    float max_voltage; /* dq 电压矢量限幅，单位 V；小于等于 0 表示不启用。 */
    float modulation_limit; /* SVPWM alpha-beta 调制矢量限幅，无单位，通常不超过 sqrt(3)/2。 */

    float vf_gain_v_per_hz; /* V/f 斜率，单位 V/Hz。 */
    float vf_boost_v; /* V/f 低频补偿电压，单位 V。 */
    float vf_min_v; /* V/f 自动电压下限，单位 V。 */
    float vf_max_v; /* V/f 自动电压上限，单位 V。 */

    bool enable_pos_loop; /* 位置环使能，无单位；输出写入 cmd_vel。 */
    bool enable_vel_loop; /* 速度环使能，无单位；输出写入 cmd_iq。 */
    bool enable_id_loop; /* d 轴电流环使能，无单位；未使能时 out_vd = cmd_vd。 */
    bool enable_iq_loop; /* q 轴电流环使能，无单位；未使能时 out_vq = cmd_vq。 */

    Pid pid_pos; /* 位置环 PID；输入单位同位置，输出单位同速度。 */
    Pid pid_vel; /* 速度环 PID；输入单位同速度，输出单位 A。 */
    Pid pid_id; /* d 轴电流环 PID；输入单位 A，输出单位 V。 */
    Pid pid_iq; /* q 轴电流环 PID；输入单位 A，输出单位 V。 */

    float ia_offset; /* A 相零电流 offset，单位 A。 */
    float ib_offset; /* B 相零电流 offset，单位 A。 */
    float ic_offset; /* C 相零电流 offset，单位 A。 */
    uint32_t offset_sample_count; /* offset 递推平均样本数，无单位。 */
    bool current_offset_valid; /* 电流 offset 是否有效，无单位。 */

    bool saturated; /* 本周期是否发生 PID、电压或 SVPWM 限幅，无单位。 */
    bool valid; /* 本周期 duty 输出是否有效，无单位。 */
} Foc;

/**
 * @brief 初始化 FOC 控制器对象。
 * @param[out] controller FOC 控制器对象指针；不能为空。
 * @return 初始化成功返回 true；controller 为空返回 false。
 */
bool foc_init(Foc *controller);

/**
 * @brief 清除三相电流 offset 估计值。
 * @param[in,out] controller FOC 控制器对象指针；不能为空。
 * @return 清除成功返回 true；controller 为空返回 false。
 */
bool foc_current_offset_clear(Foc *controller);

/**
 * @brief 将一组三相电流样本累加到 offset 递推平均值。
 * @param[in,out] controller FOC 控制器对象指针；不能为空。
 * @param[in] sample 三相电流样本，单位 A。
 * @return 累加成功返回 true；controller 为空返回 false。
 */
bool foc_current_offset_accumulate(Foc *controller, FocPhase sample);

/**
 * @brief 直接设置三相电流 offset。
 * @param[in,out] controller FOC 控制器对象指针；不能为空。
 * @param[in] offset 三相电流 offset，单位 A。
 * @return 设置成功返回 true；controller 为空返回 false。
 */
bool foc_current_offset_set(Foc *controller, FocPhase offset);

/**
 * @brief 执行一次 FOC 更新。
 *
 * 这是 VF/IF 控制器唯一执行入口。调用前，上层必须写入 mode、angle_mode、
 * control_freq、vbus、电流采样、角度来源和对应命令；调用后读取 duty_a/duty_b/duty_c。
 *
 * @param[in,out] controller FOC 控制器对象指针；不能为空。
 * @return duty 输出有效返回 true；输入非法、模式非法或 SVPWM 更新失败返回 false。
 */
bool foc_loop(Foc *controller);

#ifdef __cplusplus
}
#endif

#endif
