/**
 * @file state_machine.c
 * @brief 实现状态机、直接内存访问通知和受控并发访问。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @version 1.2.0
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：实现核心状态迁移、采样边界检查、平均值计算和中断事件消费。
 * 主要功能：展示公开接口防御性检查与内部断言的职责边界。
 * 文件关系：实现 state_machine.h；由 advanced_test.c 从公开接口验证，不依赖另外两个实现文件的内部符号。
 */

#include "state_machine.h"

#include <assert.h>

static void DEMO_EnterCritical(const demo_context_t *context);
static void DEMO_LeaveCritical(const demo_context_t *context);
static bool DEMO_IsTransitionAllowed(demo_state_t current, demo_state_t requested);
static demo_result_t DEMO_ChangeState(demo_context_t *context, demo_state_t requested);
static uint32_t DEMO_CalculateAverage(const uint16_t *samples, size_t sample_count);

/**
 * @brief 初始化核心模块上下文。
 *
 * @details
 * 实现步骤：先验证上下文及成对回调，再清空固定容量采样区，最后一次性发布空闲状态。
 * 采用先校验后写入是为了保证失败时不留下部分初始化的数据。上下文初始化期间不可并发访问。
 *
 * @param[out] context 待初始化上下文，不允许为 NULL。
 * @param[in] critical_section 成对的临界区回调，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 初始化成功；DEMO_RESULT_INVALID_ARGUMENT 表示参数无效。
 */
demo_result_t DEMO_Init(demo_context_t *context,
                        const demo_critical_section_t *critical_section)
{
    demo_result_t result = DEMO_RESULT_INVALID_ARGUMENT;

    /* 只有上下文和成对回调都有效时才写入，失败保持对象原值。 */
    if ((context != NULL) &&
        (critical_section != NULL) &&
        (critical_section->enter != NULL) &&
        (critical_section->leave != NULL))
    {
        size_t index = 0U;

        for (index = 0U; index < (size_t)DEMO_SAMPLE_CAPACITY; ++index)
        {
            /* 显式清零固定数组，避免首次使用读取未初始化数据。 */
            context->samples[index] = 0U;
        }

        context->state = DEMO_STATE_IDLE;
        context->last_result = DEMO_RESULT_OK;
        context->sample_count = 0U;
        context->dma_sequence = 0U;
        context->dma_pending = false;
        context->critical_section = *critical_section;
        result = DEMO_RESULT_OK;
    }

    return result;
}

/**
 * @brief 请求状态机从空闲状态启动。
 *
 * @details
 * 通过统一迁移函数检查当前状态并写入启动中状态，避免多个公开接口复制状态判断逻辑。
 * 本函数不轮询硬件，因此执行时间为常量上界。
 *
 * @param[in,out] context 已初始化上下文，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 请求成功；其他值表示参数或当前状态不允许启动。
 */
demo_result_t DEMO_Start(demo_context_t *context)
{
    demo_result_t result = DEMO_RESULT_INVALID_ARGUMENT;

    /* NULL 上下文无处记录错误状态，仅向调用方返回参数错误。 */
    if (context != NULL)
    {
        result = DEMO_ChangeState(context, DEMO_STATE_STARTING);
        context->last_result = result;
    }

    return result;
}

/**
 * @brief 请求状态机停止运行。
 *
 * @details
 * 只登记停止请求，实际收尾由下一个周期完成。该分阶段设计避免在接口内等待外设。
 *
 * @param[in,out] context 已初始化上下文，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 请求成功；其他值表示参数或当前状态不允许停止。
 */
demo_result_t DEMO_Stop(demo_context_t *context)
{
    demo_result_t result = DEMO_RESULT_INVALID_ARGUMENT;

    /* NULL 上下文无处记录错误状态，仅向调用方返回参数错误。 */
    if (context != NULL)
    {
        result = DEMO_ChangeState(context, DEMO_STATE_STOPPING);
        context->last_result = result;
    }

    return result;
}

/**
 * @brief 提交一批定长无符号采样。
 *
 * @details
 * 在进入临界区前完成全部外部参数检查；临界区内只复制最多 8 个元素并发布长度。
 * 先写内容后写长度，使读取方不会看到长度已更新但内容尚未完成的快照。
 *
 * @param[in,out] context 处于运行状态的上下文，不允许为 NULL。
 * @param[in] samples 输入采样数组，不允许为 NULL。
 * @param[in] sample_count 输入元素数量，范围为 1 到 DEMO_SAMPLE_CAPACITY。
 *
 * @return DEMO_RESULT_OK 提交成功；其他值表示参数或状态错误。
 */
demo_result_t DEMO_SubmitSamples(demo_context_t *context,
                                 const uint16_t *samples,
                                 size_t sample_count)
{
    demo_result_t result = DEMO_RESULT_INVALID_ARGUMENT;

    /* 入口参数失败时不解引用指针，也不覆盖上一批有效采样。 */
    if ((context != NULL) && (samples != NULL) &&
        (sample_count > 0U) && (sample_count <= (size_t)DEMO_SAMPLE_CAPACITY))
    {
        if (context->state == DEMO_STATE_RUNNING)
        {
            size_t index = 0U;

            DEMO_EnterCritical(context);
            for (index = 0U; index < sample_count; ++index)
            {
                /* 有界索引由 sample_count 的入口检查保证不会越过数组末端。 */
                context->samples[index] = samples[index];
            }
            context->sample_count = sample_count;
            DEMO_LeaveCritical(context);
            result = DEMO_RESULT_OK;
        }
        else
        {
            /* 非运行状态不得覆盖上一批有效采样。 */
            result = DEMO_RESULT_INVALID_STATE;
        }

        context->last_result = result;
    }

    return result;
}

/**
 * @brief 记录直接内存访问完成事件。
 *
 * @details
 * 本函数是中断服务程序（Interrupt Service Routine，ISR）入口。直接内存访问
 * （Direct Memory Access，DMA）序号先写入，pending 标志最后发布；主执行流在临界区消费。
 * 函数不调用任何回调，避免在中断上下文引入阻塞或不可控执行时间。
 *
 * @param[in,out] context 已初始化上下文；为 NULL 时忽略通知。
 * @param[in] sequence 硬件生成的完成序号。
 *
 * @return 无。
 */
void DEMO_NotifyDmaCompleteFromIsr(demo_context_t *context, uint32_t sequence)
{
    /* NULL 上下文无法上报，中断路径按策略忽略本次通知。 */
    if (context != NULL)
    {
        /* 先写数据再置位，保证消费方看到标志时序号已经更新。 */
        context->dma_sequence = sequence;
        context->dma_pending = true;
    }
}

/**
 * @brief 执行一次有界状态机周期并消费完成通知。
 *
 * @details
 * 1. 校验上下文和输出地址，失败时保持业务状态及调用方输出不变。
 * 2. 按当前状态只执行一个有界分支，不在周期函数内轮询或等待硬件。
 * 3. 运行状态在临界区内确认完整批次，成功后计算平均值并消费通知。
 * 4. 将本周期结果统一写回上下文；非法状态收敛到故障状态。
 *
 * @param[in,out] context 已初始化上下文，不允许为 NULL。
 * @param[out] average 成功处理采样时接收平均值，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 周期处理成功；其他值表示无数据、状态或参数错误。
 */
demo_result_t DEMO_RunCycle(demo_context_t *context, uint32_t *average)
{
    demo_result_t result = DEMO_RESULT_INVALID_ARGUMENT;

    /* 只有输入和输出位置都有效时才允许状态迁移或消费通知。 */
    if ((context != NULL) && (average != NULL))
    {
        /* 每个周期只处理当前状态对应的一个确定分支，保证执行上界可预测。 */
        switch (context->state)
        {
            case DEMO_STATE_IDLE:
                /* 空闲状态没有周期工作，保持状态并报告无有效运行数据。 */
                result = DEMO_RESULT_INVALID_STATE;
                break;

            case DEMO_STATE_STARTING:
                /* 启动准备在一个周期内完成，下一周期即可处理采样。 */
                context->state = DEMO_STATE_RUNNING;
                result = DEMO_RESULT_OK;
                break;

            case DEMO_STATE_RUNNING:
                /* 运行状态以临界区保护“通知 + 采样长度”这一致快照。 */
                DEMO_EnterCritical(context);

                /* 只有完整批次才能发布平均值，通知在成功消费后才清除。 */
                if (context->dma_pending && (context->sample_count > 0U))
                {
                    *average = DEMO_CalculateAverage(context->samples, context->sample_count);
                    context->dma_pending = false;
                    result = DEMO_RESULT_OK;
                }
                else
                {
                    /* 批次不完整时保留调用方输出和待处理通知，供后续周期继续判断。 */
                    result = DEMO_RESULT_NO_DATA;
                }

                DEMO_LeaveCritical(context);
                break;

            case DEMO_STATE_STOPPING:
                /* 停止收尾不释放动态资源，只清除批次并回到空闲状态。 */
                context->sample_count = 0U;
                context->dma_pending = false;
                context->state = DEMO_STATE_IDLE;
                result = DEMO_RESULT_OK;
                break;

            case DEMO_STATE_FAULT:
                /* 故障状态必须由更高层重新初始化，周期函数不自行恢复。 */
                result = DEMO_RESULT_INVALID_STATE;
                break;

            default:
                /* 非法枚举值表明上下文已损坏，立即收敛到故障状态。 */
                context->state = DEMO_STATE_FAULT;
                result = DEMO_RESULT_INTERNAL_ERROR;
                break;
        }

        /* 所有状态分支使用同一出口发布结果，状态快照不会遗漏本周期结论。 */
        context->last_result = result;
    }

    return result;
}

/**
 * @brief 获取一致的核心状态快照。
 *
 * @details
 * 在同一个临界区内复制全部共享字段，避免状态、序号和长度来自不同时间点。
 * 本函数不修改业务状态，因此适合监控任务周期读取。
 *
 * @param[in] context 已初始化上下文，不允许为 NULL。
 * @param[out] status 接收状态快照，不允许为 NULL。
 *
 * @return DEMO_RESULT_OK 读取成功；DEMO_RESULT_INVALID_ARGUMENT 表示参数无效。
 */
demo_result_t DEMO_GetStatus(const demo_context_t *context, demo_status_t *status)
{
    demo_result_t result = DEMO_RESULT_INVALID_ARGUMENT;

    /* 参数无效时不进入临界区，并保持调用方快照原值。 */
    if ((context != NULL) && (status != NULL))
    {
        DEMO_EnterCritical(context);
        status->state = context->state;
        status->last_result = context->last_result;
        status->dma_sequence = context->dma_sequence;
        status->sample_count = context->sample_count;
        DEMO_LeaveCritical(context);
        result = DEMO_RESULT_OK;
    }

    return result;
}

/**
 * @brief 进入调用方定义的临界区。
 *
 * @details
 * 公开入口已经验证回调完整性；这里用断言记录内部不变量，而不替代运行时参数检查。
 * 回调耗时由平台实现约束，核心模块不在临界区内调用其他外部服务。
 *
 * @param[in] context 已初始化的内部上下文。
 *
 * @return 无。
 */
static void DEMO_EnterCritical(const demo_context_t *context)
{
    assert(context != NULL);
    assert(context->critical_section.enter != NULL);
    context->critical_section.enter(context->critical_section.user_context);
}

/**
 * @brief 退出调用方定义的临界区。
 *
 * @details
 * 与 DEMO_EnterCritical 成对使用。断言用于暴露内部契约破坏，不处理外部运行时错误。
 *
 * @param[in] context 已初始化的内部上下文。
 *
 * @return 无。
 */
static void DEMO_LeaveCritical(const demo_context_t *context)
{
    assert(context != NULL);
    assert(context->critical_section.leave != NULL);
    context->critical_section.leave(context->critical_section.user_context);
}

/**
 * @brief 判断指定状态迁移是否被状态机允许。
 *
 * @details
 * 1. 按当前状态选择允许的请求：空闲只接受启动，运行只接受有序停止。
 * 2. 对过渡态、故障态和非法状态保持拒绝结果，统一返回明确的布尔判定。
 *
 * @param[in] current 当前状态。
 * @param[in] requested 请求进入的状态。
 *
 * @return true 表示允许迁移；false 表示拒绝迁移。
 */
static bool DEMO_IsTransitionAllowed(demo_state_t current, demo_state_t requested)
{
    bool allowed = false;

    switch (current)
    {
        case DEMO_STATE_IDLE:
            /* 空闲状态只接受启动请求。 */
            allowed = (requested == DEMO_STATE_STARTING);
            break;

        case DEMO_STATE_STARTING:
            /* 启动中状态由周期函数推进，公开接口不直接改变它。 */
            allowed = false;
            break;

        case DEMO_STATE_RUNNING:
            /* 运行状态只接受有序停止请求。 */
            allowed = (requested == DEMO_STATE_STOPPING);
            break;

        case DEMO_STATE_STOPPING:
            /* 停止中状态由周期函数收尾，拒绝新的迁移请求。 */
            allowed = false;
            break;

        case DEMO_STATE_FAULT:
            /* 故障状态只能通过重新初始化恢复。 */
            allowed = false;
            break;

        default:
            /* 非法当前状态不允许任何迁移。 */
            allowed = false;
            break;
    }

    return allowed;
}

/**
 * @brief 在检查迁移矩阵后更新状态。
 *
 * @details
 * 该函数只负责状态写入和错误归一化，不执行硬件动作，保证职责单一且便于复用。
 *
 * @param[in,out] context 已由公开入口验证的上下文。
 * @param[in] requested 请求进入的状态。
 *
 * @return DEMO_RESULT_OK 迁移成功；DEMO_RESULT_INVALID_STATE 表示迁移不允许。
 */
static demo_result_t DEMO_ChangeState(demo_context_t *context, demo_state_t requested)
{
    demo_result_t result = DEMO_RESULT_INVALID_STATE;

    assert(context != NULL);

    /* 不允许的请求保留当前状态，调用方可据返回码安全重试或上报。 */
    if (DEMO_IsTransitionAllowed(context->state, requested))
    {
        context->state = requested;
        result = DEMO_RESULT_OK;
    }

    return result;
}

/**
 * @brief 计算固定上限采样数组的整数平均值。
 *
 * @details
 * uint16_t 最大值乘以 8 不会超过 uint32_t。先累加再除法避免逐项截断误差；
 * 入口由调用方保证非空且数量合法，断言用于记录这一内部假设。
 *
 * @param[in] samples 有效采样数组。
 * @param[in] sample_count 元素数量，范围为 1 到 DEMO_SAMPLE_CAPACITY。
 *
 * @return 向下取整的无符号平均值。
 */
static uint32_t DEMO_CalculateAverage(const uint16_t *samples, size_t sample_count)
{
    uint32_t sum = 0U;
    size_t index = 0U;

    assert(samples != NULL);
    assert(sample_count > 0U);
    assert(sample_count <= (size_t)DEMO_SAMPLE_CAPACITY);

    for (index = 0U; index < sample_count; ++index)
    {
        /* 显式扩展为 uint32_t，使累加类型和上溢分析一目了然。 */
        sum += (uint32_t)samples[index];
    }

    return sum / (uint32_t)sample_count;
}
