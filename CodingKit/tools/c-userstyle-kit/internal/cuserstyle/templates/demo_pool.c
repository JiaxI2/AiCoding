/**
 * @file demo_pool.c
 * @brief 实现固定容量资源池和带代际句柄的生命周期保护。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @version 1.0.0
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：实现资源池初始化、申请、读写、释放和占用统计。
 * 主要功能：避免内存泄漏和释放后引用，并展示固定资源复用策略。
 * 文件关系：实现 demo_pool.h；不依赖核心状态机或协议模块的内部符号。
 */

#include "demo_pool.h"

#include <assert.h>
#include <limits.h>

static uint16_t DEMO_NextGeneration(uint16_t current);
static bool DEMO_IsPoolHandleValid(const demo_pool_t *pool, demo_pool_handle_t handle);
static void DEMO_ClearPoolSlot(demo_pool_slot_t *slot);

/**
 * @brief 初始化固定资源池。
 *
 * @details
 * 逐槽清理全部载荷、长度和占用标志，并把初始代际号设为 0。首次申请会产生非零代际号。
 * 固定布局让内存占用在编译期可知，不存在动态分配失败和释放责任不清的问题。
 *
 * @param[out] pool 待初始化资源池，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 初始化成功；DEMO_POOL_RESULT_INVALID_ARGUMENT 表示参数无效。
 */
demo_pool_result_t DEMO_PoolInit(demo_pool_t *pool)
{
    demo_pool_result_t result = DEMO_POOL_RESULT_INVALID_ARGUMENT;

    /* 只有有效池对象才执行初始化；NULL 时保留参数错误且不写内存。 */
    if (pool != NULL)
    {
        size_t index = 0U;

        for (index = 0U; index < (size_t)DEMO_POOL_CAPACITY; ++index)
        {
            /* 初始化时保留 generation 为 0，首次申请统一递增到 1。 */
            DEMO_ClearPoolSlot(&pool->slots[index]);
            pool->slots[index].generation = 0U;
        }
        result = DEMO_POOL_RESULT_OK;
    }

    return result;
}

/**
 * @brief 申请一个空闲固定资源槽。
 *
 * @details
 * 按稳定索引顺序查找第一个空闲槽，更新非零代际号后再发布占用标志和句柄。
 * 搜索次数最多为 4，资源耗尽通过明确返回码报告。
 *
 * @param[in,out] pool 已初始化资源池，不允许为 NULL。
 * @param[out] handle 接收新句柄，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 申请成功；其他值表示参数错误或资源耗尽。
 */
demo_pool_result_t DEMO_PoolAcquire(demo_pool_t *pool, demo_pool_handle_t *handle)
{
    demo_pool_result_t result = DEMO_POOL_RESULT_INVALID_ARGUMENT;

    /* 参数无效时不扫描资源池，也不修改调用方的输出句柄。 */
    if ((pool != NULL) && (handle != NULL))
    {
        size_t index = 0U;

        result = DEMO_POOL_RESULT_NO_RESOURCE;
        for (index = 0U; index < (size_t)DEMO_POOL_CAPACITY; ++index)
        {
            /* 只在首个空闲槽发布新句柄；已占用槽保持原样并继续扫描。 */
            if (!pool->slots[index].in_use)
            {
                /* 代际号先更新，再把槽和句柄作为一个逻辑事务发布。 */
                pool->slots[index].generation =
                    DEMO_NextGeneration(pool->slots[index].generation);
                pool->slots[index].length = 0U;
                pool->slots[index].in_use = true;
                handle->slot = (uint16_t)index;
                handle->generation = pool->slots[index].generation;
                result = DEMO_POOL_RESULT_OK;
                break;
            }
        }
    }

    return result;
}

/**
 * @brief 向有效资源槽写入有界二进制数据。
 *
 * @details
 * 先验证外部指针、长度和代际句柄，再复制全部数据并最后发布有效长度。
 * 失败路径不会改变已有资源内容，防止调用方观察到半写入状态。
 *
 * @param[in,out] pool 已初始化资源池，不允许为 NULL。
 * @param[in] handle 当前有效句柄。
 * @param[in] data 输入二进制区，不允许为 NULL。
 * @param[in] data_length 输入字节数，不得超过 DEMO_POOL_PAYLOAD_CAPACITY。
 *
 * @return DEMO_POOL_RESULT_OK 写入成功；其他值表示参数或句柄错误。
 */
demo_pool_result_t DEMO_PoolWrite(demo_pool_t *pool,
                                  demo_pool_handle_t handle,
                                  const uint8_t *data,
                                  size_t data_length)
{
    demo_pool_result_t result = DEMO_POOL_RESULT_INVALID_ARGUMENT;

    /* 指针或长度无效时不校验句柄，也不改变已有槽内容。 */
    if ((pool != NULL) && (data != NULL) &&
        (data_length <= (size_t)DEMO_POOL_PAYLOAD_CAPACITY))
    {
        if (DEMO_IsPoolHandleValid(pool, handle))
        {
            size_t index = 0U;
            demo_pool_slot_t *const slot = &pool->slots[handle.slot];

            for (index = 0U; index < data_length; ++index)
            {
                /* 显式长度控制二进制复制，不使用字符串长度函数。 */
                slot->payload[index] = data[index];
            }
            slot->length = data_length;
            result = DEMO_POOL_RESULT_OK;
        }
        else
        {
            /* 释放后的旧代际句柄不得访问重新分配的同一索引。 */
            result = DEMO_POOL_RESULT_INVALID_HANDLE;
        }
    }

    return result;
}

/**
 * @brief 从有效资源槽读取有界二进制数据。
 *
 * @details
 * output_length 同时承载输入容量和成功输出长度。容量不足时返回所需长度但不复制部分数据，
 * 使调用方能明确重试。代际校验防止释放后读取。
 *
 * @param[in] pool 已初始化资源池，不允许为 NULL。
 * @param[in] handle 当前有效句柄。
 * @param[out] output 接收二进制数据，不允许为 NULL。
 * @param[in,out] output_length 输入目标容量，成功时输出实际字节数，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 读取成功；其他值表示参数、句柄或容量错误。
 */
demo_pool_result_t DEMO_PoolRead(const demo_pool_t *pool,
                                 demo_pool_handle_t handle,
                                 uint8_t *output,
                                 size_t *output_length)
{
    demo_pool_result_t result = DEMO_POOL_RESULT_INVALID_ARGUMENT;

    /* 指针全部有效后才查询句柄；入口失败时不泄漏长度或数据。 */
    if ((pool != NULL) && (output != NULL) && (output_length != NULL))
    {
        if (DEMO_IsPoolHandleValid(pool, handle))
        {
            const demo_pool_slot_t *const slot = &pool->slots[handle.slot];

            if (*output_length < slot->length)
            {
                /* 只报告所需长度，不复制无法完整容纳的数据。 */
                *output_length = slot->length;
                result = DEMO_POOL_RESULT_OUTPUT_TOO_SMALL;
            }
            else
            {
                size_t index = 0U;

                for (index = 0U; index < slot->length; ++index)
                {
                    /* 循环上界来自成功写入时验证过的固定容量长度。 */
                    output[index] = slot->payload[index];
                }
                *output_length = slot->length;
                result = DEMO_POOL_RESULT_OK;
            }
        }
        else
        {
            /* 无效句柄不泄漏槽长度或历史内容。 */
            result = DEMO_POOL_RESULT_INVALID_HANDLE;
        }
    }

    return result;
}

/**
 * @brief 释放一个有效资源句柄。
 *
 * @details
 * 验证索引、占用状态和代际号后清除全部载荷，再撤销占用标志。generation 保留到下次申请
 * 时递增，使旧句柄不能命中新资源。固定资源无需调用 free，也不存在泄漏责任转移。
 *
 * @param[in,out] pool 已初始化资源池，不允许为 NULL。
 * @param[in] handle 当前有效句柄。
 *
 * @return DEMO_POOL_RESULT_OK 释放成功；其他值表示参数或句柄已经失效。
 */
demo_pool_result_t DEMO_PoolRelease(demo_pool_t *pool, demo_pool_handle_t handle)
{
    demo_pool_result_t result = DEMO_POOL_RESULT_INVALID_ARGUMENT;

    /* NULL 池对象不进入生命周期操作，结果保持参数错误。 */
    if (pool != NULL)
    {
        if (DEMO_IsPoolHandleValid(pool, handle))
        {
            /* 清理数据后才发布空闲状态，避免残留内容被下一使用者观察。 */
            DEMO_ClearPoolSlot(&pool->slots[handle.slot]);
            result = DEMO_POOL_RESULT_OK;
        }
        else
        {
            /* 重复释放或陈旧句柄都通过同一个确定错误码报告。 */
            result = DEMO_POOL_RESULT_INVALID_HANDLE;
        }
    }

    return result;
}

/**
 * @brief 统计当前占用的资源槽数量。
 *
 * @details
 * 扫描固定 4 个槽并只读取 in_use 标志。共享场景下，调用方应在本函数外层提供互斥，
 * 以保证得到同一时间点的业务快照。
 *
 * @param[in] pool 已初始化资源池，不允许为 NULL。
 * @param[out] in_use_count 接收占用数量，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 统计成功；DEMO_POOL_RESULT_INVALID_ARGUMENT 表示参数无效。
 */
demo_pool_result_t DEMO_PoolGetInUseCount(const demo_pool_t *pool, size_t *in_use_count)
{
    demo_pool_result_t result = DEMO_POOL_RESULT_INVALID_ARGUMENT;

    /* 只有输入输出对象都有效时才扫描；失败时保留调用方计数。 */
    if ((pool != NULL) && (in_use_count != NULL))
    {
        size_t count = 0U;
        size_t index = 0U;

        for (index = 0U; index < (size_t)DEMO_POOL_CAPACITY; ++index)
        {
            /* 只累计占用槽，空闲槽保持不计数。 */
            if (pool->slots[index].in_use)
            {
                ++count;
            }
        }
        *in_use_count = count;
        result = DEMO_POOL_RESULT_OK;
    }

    return result;
}

/**
 * @brief 生成下一个非零资源代际号。
 *
 * @details
 * UINT16_MAX 回绕时显式跳到 1，永远保留 0 作为未分配标记。分支避免依赖无符号隐式回绕
 * 来表达业务语义，并使边界行为可直接测试。
 *
 * @param[in] current 当前代际号。
 *
 * @return 范围为 1 到 UINT16_MAX 的下一个代际号。
 */
static uint16_t DEMO_NextGeneration(uint16_t current)
{
    uint16_t next = 1U;

    /* 默认把最大代际回绕到 1；其他值按序递增，0 始终保留。 */
    if (current < UINT16_MAX)
    {
        next = (uint16_t)(current + 1U);
    }

    return next;
}

/**
 * @brief 验证资源句柄是否指向当前占用代际。
 *
 * @details
 * 检查顺序先验证槽索引，再访问数组成员；随后同时比较占用标志和代际号。
 * 该函数只读资源池，不改变生命周期状态。
 *
 * @param[in] pool 已由公开入口检查的资源池。
 * @param[in] handle 待验证句柄。
 *
 * @return true 表示句柄当前有效；false 表示索引、占用状态或代际不匹配。
 */
static bool DEMO_IsPoolHandleValid(const demo_pool_t *pool, demo_pool_handle_t handle)
{
    bool valid = false;

    assert(pool != NULL);

    /* 只有索引在范围内才访问槽；越界句柄直接保持无效。 */
    if ((size_t)handle.slot < (size_t)DEMO_POOL_CAPACITY)
    {
        const demo_pool_slot_t *const slot = &pool->slots[handle.slot];

        valid = slot->in_use && (slot->generation == handle.generation);
    }

    return valid;
}

/**
 * @brief 清除一个固定资源槽并撤销占用状态。
 *
 * @details
 * 覆盖全部固定载荷，随后清零长度并最后撤销占用标志。代际号有意保留供下一次申请递增。
 *
 * @param[in,out] slot 待清理的内部资源槽。
 *
 * @return 无。
 */
static void DEMO_ClearPoolSlot(demo_pool_slot_t *slot)
{
    size_t index = 0U;

    assert(slot != NULL);

    for (index = 0U; index < (size_t)DEMO_POOL_PAYLOAD_CAPACITY; ++index)
    {
        /* 显式覆盖每个字节，避免资源复用时暴露上一使用者数据。 */
        slot->payload[index] = 0U;
    }
    slot->length = 0U;
    slot->in_use = false;
}
