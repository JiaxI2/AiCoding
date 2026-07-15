/**
 * @file state_machine.h
 * @brief 状态机、直接内存访问通知和受控并发访问的公开接口。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @version 1.2.0
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：声明核心状态机的数据类型、临界区适配接口和公开函数。
 * 主要功能：演示参数校验、状态迁移、数值边界、直接内存访问完成通知和快照读取。
 * 文件关系：由 state_machine.c 实现；advanced_test.c 只通过本文件公开的接口验证行为。
 */

#ifndef ADVANCED_STATE_MACHINE_H
#define ADVANCED_STATE_MACHINE_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 单批采样允许保存的元素数量。
 */
enum
{
    DEMO_SAMPLE_CAPACITY = 8
};

/**
 * @brief 核心模块运行状态。
 */
typedef enum
{
    DEMO_STATE_IDLE = 0,
    DEMO_STATE_STARTING,
    DEMO_STATE_RUNNING,
    DEMO_STATE_STOPPING,
    DEMO_STATE_FAULT
} demo_state_t;

/**
 * @brief 核心模块操作结果。
 */
typedef enum
{
    DEMO_RESULT_OK = 0,
    DEMO_RESULT_INVALID_ARGUMENT,
    DEMO_RESULT_INVALID_STATE,
    DEMO_RESULT_NO_DATA,
    DEMO_RESULT_INTERNAL_ERROR
} demo_result_t;

/**
 * @brief 进入或退出临界区的适配函数类型。
 *
 * @param[in,out] user_context 平台适配层上下文；其含义由回调实现约定。
 */
typedef void (*demo_critical_fn)(void *user_context);

/**
 * @brief 临界区适配接口。
 *
 * @details 两个回调必须成对提供，并由调用方保证不会阻塞中断服务程序。
 */
typedef struct
{
    demo_critical_fn enter;  /**< 进入临界区，屏蔽共享快照被并发修改。 */
    demo_critical_fn leave;  /**< 退出临界区，恢复平台原有并发状态。 */
    void *user_context;      /**< 原样传给 enter 和 leave 的平台上下文。 */
} demo_critical_section_t;

/**
 * @brief 核心模块可观察状态快照。
 */
typedef struct
{
    demo_state_t state;          /**< 读取快照时的状态机状态。 */
    demo_result_t last_result;   /**< 最近一次公开操作的确定结果。 */
    uint32_t dma_sequence;       /**< 最近处理的直接内存访问完成序号。 */
    size_t sample_count;         /**< 当前有效采样数量，范围为 0 到 8。 */
} demo_status_t;

/**
 * @brief 核心模块上下文。
 *
 * @details
 * 上下文由调用方静态分配。主执行流通过临界区访问中断共享字段；中断服务程序只写入
 * dma_sequence 和 dma_pending。禁止复制正在运行的上下文。
 */
typedef struct
{
    demo_state_t state;                         /**< 当前状态机状态。 */
    demo_result_t last_result;                  /**< 最近一次公开操作结果。 */
    uint16_t samples[DEMO_SAMPLE_CAPACITY];     /**< 当前批次的有界采样。 */
    size_t sample_count;                        /**< samples 中有效元素数量。 */
    volatile uint32_t dma_sequence;             /**< 中断写、主执行流受保护读取。 */
    volatile bool dma_pending;                  /**< 尚未消费的完成通知标志。 */
    demo_critical_section_t critical_section;   /**< 平台提供的并发保护接口。 */
} demo_context_t;

/**
 * @brief 初始化核心模块上下文。
 *
 * @param[out] context 待初始化上下文，不允许为 NULL。
 * @param[in] critical_section 成对的临界区回调，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 初始化成功；其他值表示参数无效。
 *
 * @note 性能为固定 8 次初始化，执行时间有界；函数不可重入，同一上下文须由一个调用方初始化。
 */
demo_result_t DEMO_Init(demo_context_t *context,
                        const demo_critical_section_t *critical_section);

/**
 * @brief 请求状态机从空闲状态启动。
 *
 * @param[in,out] context 已初始化上下文，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 已进入启动中状态；其他值表示参数或状态不允许启动。
 *
 * @note 性能为常量时间；调用方不得与同一上下文的其他主执行流操作并发调用。
 */
demo_result_t DEMO_Start(demo_context_t *context);

/**
 * @brief 请求状态机停止运行。
 *
 * @param[in,out] context 已初始化上下文，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 已进入停止中状态；其他值表示参数或状态不允许停止。
 *
 * @note 性能为常量时间；函数不等待硬件，且不可与同一上下文的主执行流操作并发调用。
 */
demo_result_t DEMO_Stop(demo_context_t *context);

/**
 * @brief 提交一批定长无符号采样。
 *
 * @param[in,out] context 处于运行状态的上下文，不允许为 NULL。
 * @param[in] samples 输入采样数组，不允许为 NULL。
 * @param[in] sample_count 输入元素数量，范围为 1 到 DEMO_SAMPLE_CAPACITY。
 *
 * @return DEMO_RESULT_OK 提交成功；其他值表示参数或状态错误。
 *
 * @note 性能与 sample_count 线性相关且上限为 8；并发访问由临界区保护且不执行阻塞操作。
 */
demo_result_t DEMO_SubmitSamples(demo_context_t *context,
                                 const uint16_t *samples,
                                 size_t sample_count);

/**
 * @brief 记录直接内存访问完成事件。
 *
 * @param[in,out] context 已初始化上下文；为 NULL 时忽略通知。
 * @param[in] sequence 硬件生成的单调序号，回绕语义由上层协议定义。
 *
 * @return 无。
 *
 * @note 性能为固定次数写操作；本函数供中断服务程序调用，不调用临界区或阻塞接口。
 */
void DEMO_NotifyDmaCompleteFromIsr(demo_context_t *context, uint32_t sequence);

/**
 * @brief 执行一次有界状态机周期并消费完成通知。
 *
 * @param[in,out] context 已初始化上下文，不允许为 NULL。
 * @param[out] average 成功处理采样时接收平均值，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 周期处理成功；其他值表示无数据、状态或参数错误。
 *
 * @note 性能为常量上界；函数不可重入，但可与指定中断服务程序按接口约定协作。
 */
demo_result_t DEMO_RunCycle(demo_context_t *context, uint32_t *average);

/**
 * @brief 获取一致的核心状态快照。
 *
 * @param[in] context 已初始化上下文，不允许为 NULL。
 * @param[out] status 接收状态快照，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 读取成功；其他值表示参数错误。
 *
 * @note 性能为常量时间；与中断并发读取时使用调用方提供的临界区回调。
 */
demo_result_t DEMO_GetStatus(const demo_context_t *context, demo_status_t *status);

#ifdef __cplusplus
}
#endif

#endif /* ADVANCED_STATE_MACHINE_H */
