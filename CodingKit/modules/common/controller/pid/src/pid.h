#ifndef PID_CONTROLLER_H
#define PID_CONTROLLER_H

/**
 * @file pid.h
 * @brief 通用 PID 控制单元。
 * @author HU JIAXUAN
 *
 * 本模块不是裸 PID 公式，而是面向嵌入式实时控制的 PID 控制单元。
 * 单次 pid() 调用会完成目标整形、误差调节、微分滤波、输出保护和积分抗饱和。
 *
 * 使用流程：
 * 1. 调用 pid_init(&controller) 完成默认初始化；
 * 2. 按工程实际覆盖 config 与 input；
 * 3. 按 controlFreq 周期调用 pid(&controller)；
 * 4. 使用返回值或 state.output 作为控制输出。
 *
 * 单周期控制链：
 * 1. 检查 controlFreq 和限幅配置；
 * 2. 对 setpoint 做限幅和变化率限制；
 * 3. 对 feedback 做可选限幅；
 * 4. 计算并可选限幅 error；
 * 5. 计算 P/I/D 和 feedforward；
 * 6. 对 D 项微分做一阶低通滤波；
 * 7. 对 output 做限幅、deadband 和 back-calculation 抗积分饱和；
 * 8. 更新 state 观测量。
 *
 * 单位约定：
 * - setpoint、feedback、error 使用同一被控量单位；若用标幺，记为 pu；
 * - output、feedforward、P/I/D 分量使用同一输出单位；若用标幺，记为 pu；
 * - 增益单位由“输出单位/被控量单位”自动决定。
 *
 * 设计边界：
 * - 不绑定具体上层业务和硬件对象；
 * - P/PI/PD/PID 由 kp、ki、kd 是否为 0 决定；
 * - ki 非 0 时启用 back-calculation 积分抗饱和；
 * - 抗饱和拓扑与随包 Simulink 参考模型保持一致：u_raw 经输出限幅得到 u_sat，
 *   再用 Kaw * (u_sat - u_raw) 反馈到积分输入；
 * - 输入整形、限幅、死区和微分滤波均为可选控制工具，默认关闭或不影响裸 PID。
 */

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* 控制频率合法性判断阈值，单位 Hz。 */
#ifndef PID_EPSILON
#define PID_EPSILON (1.0e-12f)
#endif

/* 默认控制频率，单位 Hz；工程移植时应按实际中断频率覆盖。 */
#ifndef PID_DEFAULT_CONTROL_FREQ_HZ
#define PID_DEFAULT_CONTROL_FREQ_HZ (1000.0f)
#endif

/** @brief PID 运行状态码。 */
typedef enum {
    PID_OK = 0,          /* 计算正常。 */
    PID_ERR_NULL = -1,   /* 控制器指针为空。 */
    PID_ERR_FREQ = -2,   /* 控制频率非法。 */
    PID_ERR_LIMIT = -3   /* 限幅配置非法。 */
} PidStatus;

/** @brief 通用上下限，单位由具体字段决定。 */
typedef struct {
    bool enable; /* 启用开关，无单位。 */
    float min;   /* 下限，单位同被限幅字段；标幺时为 pu。 */
    float max;   /* 上限，单位同被限幅字段；标幺时为 pu。 */
} PidLimit;

/** @brief 本周期输入。 */
typedef struct {
    float setpoint;    /* 目标值，单位同被控量；标幺时为 pu。 */
    float feedback;    /* 反馈值，单位同被控量；标幺时为 pu。 */
    float feedforward; /* 前馈量，单位同输出；标幺时为 pu。 */
} PidInput;

/** @brief 控制器配置。 */
typedef struct {
    /* PID 参数。 */
    float kp;          /* 比例增益，输出单位/被控量单位；标幺时为 pu/pu。 */
    float ki;          /* 积分增益，输出单位/(被控量单位*s)；标幺时为 pu/(pu*s)。 */
    float kd;          /* 微分增益，输出单位*s/被控量单位；标幺时为 pu*s/pu。 */
    float controlFreq; /* 控制频率，单位 Hz。 */

    /* 输入整形与误差保护。 */
    PidLimit setpointLimit; /* 目标限幅，单位同被控量；标幺时为 pu。 */
    PidLimit feedbackLimit; /* 反馈限幅，单位同被控量；标幺时为 pu。 */
    PidLimit errorLimit;    /* 误差限幅，单位同被控量；标幺时为 pu。 */
    bool setpointRateEnable; /* 目标斜率限幅开关，无单位。 */
    float setpointRate;      /* 目标最大变化率，单位为被控量单位/s；标幺时为 pu/s。 */

    /* 微分滤波。 */
    float derivativeFilterCoef; /* 微分滤波系数，无单位，范围 [0, 1]。 */

    /* 积分和输出保护。 */
    PidLimit integralLimit; /* 积分限幅，单位同输出；标幺时为 pu。 */
    PidLimit outputLimit;   /* 输出限幅，单位同输出；标幺时为 pu。 */
    float antiWindupGain;  /* 抗饱和增益，单位为被控量单位/输出单位；标幺时为 pu/pu。 */
    float deadband;        /* 输出死区，单位同输出；标幺时为 pu。 */
} PidConfig;

/** @brief 控制器状态与观测量。 */
typedef struct {
    float setpoint;        /* 原始目标，单位同被控量；标幺时为 pu。 */
    float setpointLimited; /* 限幅目标，单位同被控量；标幺时为 pu。 */
    float feedback;        /* 原始反馈，单位同被控量；标幺时为 pu。 */
    float feedbackLimited; /* 限幅反馈，单位同被控量；标幺时为 pu。 */
    float error;           /* 当前误差，单位同被控量；标幺时为 pu。 */
    float previousError;   /* 上一拍误差，单位同被控量；标幺时为 pu。 */

    float derivative;         /* 原始微分，单位为被控量单位/s；标幺时为 pu/s。 */
    float derivativeFiltered; /* 滤波微分，单位为被控量单位/s；标幺时为 pu/s。 */

    float proportional;   /* P 分量，单位同输出；标幺时为 pu。 */
    float integral;       /* I 分量，单位同输出；标幺时为 pu。 */
    float derivativeTerm; /* D 分量，单位同输出；标幺时为 pu。 */
    float feedforward;    /* 前馈分量，单位同输出；标幺时为 pu。 */

    float rawOutput; /* 限幅前输出，单位同输出；标幺时为 pu。 */
    float output;    /* 最终输出，单位同输出；标幺时为 pu。 */

    bool saturated;   /* 输出饱和标志，无单位。 */
    bool initialized; /* 历史状态有效标志，无单位。 */
    PidStatus status; /* 最近状态码，无单位。 */
} PidState;

/** @brief PID 控制器对象。 */
typedef struct {
    PidConfig config; /* 配置区。 */
    PidInput input;   /* 输入区。 */
    PidState state;   /* 状态区。 */
} Pid;


/**
 * @brief 初始化 PID 控制器对象。
 * @param[out] controller 控制器对象。
 * @return true 表示初始化成功，false 表示对象为空。
 *
 * 初始化内容：清零输入和状态，关闭可选限幅/斜率/死区工具，设置默认控制频率。
 * 默认值只保证对象处于确定状态；实际 kp、ki、kd 和 controlFreq 仍应由工程覆盖。
 */
bool pid_init(Pid *controller);

/**
 * @brief 执行一次 PID 更新。
 * @param[in,out] controller 控制器对象；首次使用前清零。
 * @return 当前输出，单位同输出；等同于 controller->state.output。
 */
float pid(Pid *controller);

#ifdef __cplusplus
}
#endif

#endif /* PID_CONTROLLER_H */
