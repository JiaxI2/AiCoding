/**
 * @file fixed_pool.h
 * @brief 固定容量资源池及带代际句柄的生命周期接口。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @version 1.2.0
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：声明静态资源池、资源句柄和申请、读写、释放接口。
 * 主要功能：用固定存储替代动态分配，并通过代际号拒绝释放后的旧句柄。
 * 文件关系：由 fixed_pool.c 实现；可独立于 state_machine.h 和 protocol.h 包含及编译。
 */

#ifndef ADVANCED_FIXED_POOL_H
#define ADVANCED_FIXED_POOL_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 资源池固定布局参数。
 */
enum
{
    DEMO_POOL_CAPACITY = 4,
    DEMO_POOL_PAYLOAD_CAPACITY = 16
};

/**
 * @brief 资源池操作结果。
 */
typedef enum
{
    DEMO_POOL_RESULT_OK = 0,
    DEMO_POOL_RESULT_INVALID_ARGUMENT,
    DEMO_POOL_RESULT_NO_RESOURCE,
    DEMO_POOL_RESULT_INVALID_HANDLE,
    DEMO_POOL_RESULT_OUTPUT_TOO_SMALL
} demo_pool_result_t;

/**
 * @brief 标识一次具体资源占用的代际句柄。
 */
typedef struct
{
    uint16_t slot;        /**< 固定槽索引，范围为 0 到 3。 */
    uint16_t generation;  /**< 每次重新申请时变化，0 永远不是有效代际号。 */
} demo_pool_handle_t;

/**
 * @brief 单个固定资源槽。
 *
 * @details 槽只由资源池接口修改；调用方不得直接缓存 payload 指针。
 */
typedef struct
{
    uint8_t payload[DEMO_POOL_PAYLOAD_CAPACITY];  /**< 槽内固定二进制存储。 */
    size_t length;                                /**< 当前有效字节数。 */
    uint16_t generation;                         /**< 当前占用的代际号。 */
    bool in_use;                                 /**< true 表示槽已经分配。 */
} demo_pool_slot_t;

/**
 * @brief 固定容量资源池上下文。
 *
 * @details 调用方静态分配并通过公开接口访问；模块内部不申请堆内存。
 */
typedef struct
{
    demo_pool_slot_t slots[DEMO_POOL_CAPACITY];  /**< 全部固定资源槽。 */
} demo_pool_t;

/**
 * @brief 初始化固定资源池。
 *
 * @param[out] pool 待初始化资源池，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 初始化成功；其他值表示参数无效。
 *
 * @note 性能为固定 4 乘 16 次清理；同一资源池初始化期间不可并发访问。
 */
demo_pool_result_t DEMO_PoolInit(demo_pool_t *pool);

/**
 * @brief 申请一个空闲固定资源槽。
 *
 * @param[in,out] pool 已初始化资源池，不允许为 NULL。
 * @param[out] handle 接收新句柄，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 申请成功；其他值表示参数错误或资源耗尽。
 *
 * @note 性能为最多扫描 4 个槽且不动态分配；调用方负责为共享资源池提供互斥。
 */
demo_pool_result_t DEMO_PoolAcquire(demo_pool_t *pool, demo_pool_handle_t *handle);

/**
 * @brief 向有效资源槽写入有界二进制数据。
 *
 * @param[in,out] pool 已初始化资源池，不允许为 NULL。
 * @param[in] handle 当前有效句柄。
 * @param[in] data 输入二进制区，不允许为 NULL。
 * @param[in] data_length 输入字节数，不得超过 DEMO_POOL_PAYLOAD_CAPACITY。
 *
 * @return DEMO_POOL_RESULT_OK 写入成功；其他值表示参数或句柄错误。
 *
 * @note 性能为最多复制 16 字节；共享资源池由调用方互斥，二进制数据不作字符串处理。
 */
demo_pool_result_t DEMO_PoolWrite(demo_pool_t *pool,
                                  demo_pool_handle_t handle,
                                  const uint8_t *data,
                                  size_t data_length);

/**
 * @brief 从有效资源槽读取有界二进制数据。
 *
 * @param[in] pool 已初始化资源池，不允许为 NULL。
 * @param[in] handle 当前有效句柄。
 * @param[out] output 接收二进制数据，不允许为 NULL。
 * @param[in,out] output_length 输入目标容量，成功时输出实际字节数，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 读取成功；其他值表示参数、句柄或容量错误。
 *
 * @note 性能为最多复制 16 字节；共享资源池由调用方互斥，失败时不复制部分数据。
 */
demo_pool_result_t DEMO_PoolRead(const demo_pool_t *pool,
                                 demo_pool_handle_t handle,
                                 uint8_t *output,
                                 size_t *output_length);

/**
 * @brief 释放一个有效资源句柄。
 *
 * @param[in,out] pool 已初始化资源池，不允许为 NULL。
 * @param[in] handle 当前有效句柄。
 *
 * @return DEMO_POOL_RESULT_OK 释放成功；其他值表示参数或句柄已经失效。
 *
 * @note 性能为固定清除 16 字节；共享资源池由调用方互斥，旧句柄不能再次访问资源。
 */
demo_pool_result_t DEMO_PoolRelease(demo_pool_t *pool, demo_pool_handle_t handle);

/**
 * @brief 统计当前占用的资源槽数量。
 *
 * @param[in] pool 已初始化资源池，不允许为 NULL。
 * @param[out] in_use_count 接收占用数量，不允许为 NULL。
 *
 * @return DEMO_POOL_RESULT_OK 统计成功；其他值表示参数无效。
 *
 * @note 性能为固定扫描 4 个槽；共享资源池由调用方在本次快照外层提供互斥。
 */
demo_pool_result_t DEMO_PoolGetInUseCount(const demo_pool_t *pool, size_t *in_use_count);

#ifdef __cplusplus
}
#endif

#endif /* ADVANCED_FIXED_POOL_H */
