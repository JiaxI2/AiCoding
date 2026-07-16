/**
 * @file advanced_test.c
 * @brief 从公开接口验证黄金示例的行为、边界和故障注入路径。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：包含简单采样评估、状态机、协议处理和固定资源池的主机单元测试。
 * 主要功能：验证正常流程、容量边界、错误返回码、陈旧句柄和临界区配对。
 * 文件关系：只包含四个公开头文件，不访问任何静态函数或实现私有数据。
 */

#include "demo.h"
#include "fixed_pool.h"
#include "protocol.h"
#include "state_machine.h"

#include <assert.h>
#include <stddef.h>
#include <stdint.h>
#include <string.h>

typedef struct
{
    uint32_t enter_count;
    uint32_t leave_count;
} demo_test_lock_t;

static void DEMO_TestEnterCritical(void *user_context);
static void DEMO_TestLeaveCritical(void *user_context);
static void DEMO_TestSimpleEvaluation(void);
static void DEMO_TestCoreLifecycle(void);
static void DEMO_TestProtocolBoundaries(void);
static void DEMO_TestPoolLifecycle(void);
static void DEMO_TestFaultInjection(void);

/**
 * @brief 执行全部公开行为测试。
 *
 * @details
 * 测试按模块职责顺序运行。assert 只用于测试失败判定，不承担生产代码的运行时参数检查。
 * 每个测试自行构造上下文，避免用例间共享可变全局状态。
 *
 * @return 0 表示全部断言通过；断言失败时由测试运行时终止进程。
 */
int main(void)
{
    DEMO_TestSimpleEvaluation();
    DEMO_TestCoreLifecycle();
    DEMO_TestProtocolBoundaries();
    DEMO_TestPoolLifecycle();
    DEMO_TestFaultInjection();

    return 0;
}

/**
 * @brief 验证入门 demo 的平均值、等级和参数错误行为。
 *
 * @details
 * 使用四个固定采样得到可人工核对的平均值；随后覆盖空输入、反向阈值和非法枚举分支。
 * 测试只调用 demo.h 的公开接口，证明入门示例无需理解高级模块即可独立使用。
 *
 * @return 无。
 */
static void DEMO_TestSimpleEvaluation(void)
{
    const uint16_t samples[4] = {10U, 20U, 30U, 40U};
    const demo_thresholds_t thresholds = {15U, 35U};
    const demo_thresholds_t invalid_thresholds = {40U, 20U};
    demo_sample_summary_t summary = {0U, DEMO_SAMPLE_LEVEL_LOW};

    assert(DEMO_EvaluateSamples(samples, 4U, &thresholds, &summary) ==
           DEMO_EVALUATE_RESULT_OK);
    assert(summary.average == 25U);
    assert(summary.level == DEMO_SAMPLE_LEVEL_NORMAL);
    assert(strcmp(DEMO_GetLevelName(summary.level), "normal") == 0);
    assert(DEMO_EvaluateSamples(samples, 0U, &thresholds, &summary) ==
           DEMO_EVALUATE_RESULT_INVALID_ARGUMENT);
    assert(DEMO_EvaluateSamples(samples, 4U, &invalid_thresholds, &summary) ==
           DEMO_EVALUATE_RESULT_INVALID_ARGUMENT);
    assert(strcmp(DEMO_GetLevelName((demo_sample_level_t)99), "unknown") == 0);
}

/**
 * @brief 记录一次测试临界区进入操作。
 *
 * @details 将注入上下文转换为测试计数器并递增，用于验证生产接口的进入与退出严格配对。
 *
 * @param[in,out] user_context 指向 demo_test_lock_t 的测试上下文。
 *
 * @return 无。
 */
static void DEMO_TestEnterCritical(void *user_context)
{
    demo_test_lock_t *const lock = (demo_test_lock_t *)user_context;

    assert(lock != NULL);
    ++lock->enter_count;
}

/**
 * @brief 记录一次测试临界区退出操作。
 *
 * @details 与进入回调使用同一个计数器，使测试能够检测遗漏退出或多余退出。
 *
 * @param[in,out] user_context 指向 demo_test_lock_t 的测试上下文。
 *
 * @return 无。
 */
static void DEMO_TestLeaveCritical(void *user_context)
{
    demo_test_lock_t *const lock = (demo_test_lock_t *)user_context;

    assert(lock != NULL);
    ++lock->leave_count;
}

/**
 * @brief 验证核心状态机的正常生命周期和共享快照。
 *
 * @details
 * 用公开接口完成初始化、启动、提交、通知、计算、读取状态和停止；同时验证平均值及临界区配对。
 *
 * @return 无。
 */
static void DEMO_TestCoreLifecycle(void)
{
    demo_context_t context;
    demo_status_t status;
    demo_test_lock_t lock = {0U, 0U};
    const demo_critical_section_t critical_section =
    {
        DEMO_TestEnterCritical,
        DEMO_TestLeaveCritical,
        &lock
    };
    const uint16_t samples[4] = {10U, 20U, 30U, 40U};
    uint32_t average = 0U;

    assert(DEMO_Init(&context, &critical_section) == DEMO_RESULT_OK);
    assert(DEMO_Start(&context) == DEMO_RESULT_OK);
    assert(DEMO_RunCycle(&context, &average) == DEMO_RESULT_OK);
    assert(DEMO_SubmitSamples(&context, samples, 4U) == DEMO_RESULT_OK);
    DEMO_NotifyDmaCompleteFromIsr(&context, 7U);
    assert(DEMO_RunCycle(&context, &average) == DEMO_RESULT_OK);
    assert(average == 25U);
    assert(DEMO_GetStatus(&context, &status) == DEMO_RESULT_OK);
    assert(status.state == DEMO_STATE_RUNNING);
    assert(status.dma_sequence == 7U);
    assert(status.sample_count == 4U);
    assert(DEMO_Stop(&context) == DEMO_RESULT_OK);
    assert(DEMO_RunCycle(&context, &average) == DEMO_RESULT_OK);
    assert(context.state == DEMO_STATE_IDLE);
    assert(lock.enter_count == lock.leave_count);
}

/**
 * @brief 验证协议的字节序、校验、字符串边界和固定格式输出。
 *
 * @details
 * 构造一个完整帧并验证字段；再注入校验失败、目标过小和缺少空字符等边界错误。
 *
 * @return 无。
 */
static void DEMO_TestProtocolBoundaries(void)
{
    uint8_t frame[DEMO_PROTOCOL_FRAME_SIZE] = {1U, 0x12U, 0x34U, 0U, 0U, 0U, 5U, 0U};
    demo_protocol_message_t message;
    char text[8];
    char formatted[32];
    const char source[] = "safe";
    const char unterminated[3] = {'b', 'a', 'd'};
    size_t index = 0U;

    for (index = 0U; index < ((size_t)DEMO_PROTOCOL_FRAME_SIZE - 1U); ++index)
    {
        /* 测试按生产协议定义计算异或校验，避免依赖私有实现函数。 */
        frame[7] = (uint8_t)(frame[7] ^ frame[index]);
    }

    assert(DEMO_DecodeFrame(frame, sizeof(frame), &message) == DEMO_PROTOCOL_RESULT_OK);
    assert(message.command == 0x1234U);
    assert(message.value == 5U);
    frame[7] = (uint8_t)(frame[7] ^ 1U);
    assert(DEMO_DecodeFrame(frame, sizeof(frame), &message) ==
           DEMO_PROTOCOL_RESULT_INVALID_CHECKSUM);
    assert(DEMO_CopyText(source, sizeof(source), text, sizeof(text)) ==
           DEMO_PROTOCOL_RESULT_OK);
    assert(strcmp(text, source) == 0);
    assert(DEMO_CopyText(unterminated, sizeof(unterminated), text, sizeof(text)) ==
           DEMO_PROTOCOL_RESULT_INVALID_LENGTH);
    assert(DEMO_FormatStatus(42U, formatted, sizeof(formatted)) == DEMO_PROTOCOL_RESULT_OK);
    assert(strcmp(formatted, "sequence=42") == 0);
    assert(DEMO_FormatStatus(42U, formatted, 4U) ==
           DEMO_PROTOCOL_RESULT_OUTPUT_TOO_SMALL);
}

/**
 * @brief 验证固定资源池的申请、容量检查、释放和陈旧句柄拒绝。
 *
 * @details
 * 完成一次资源生命周期后保留旧句柄并重新申请，确认代际号使旧句柄不能读取新资源。
 *
 * @return 无。
 */
static void DEMO_TestPoolLifecycle(void)
{
    demo_pool_t pool;
    demo_pool_handle_t old_handle;
    demo_pool_handle_t new_handle;
    const uint8_t input[3] = {1U, 2U, 3U};
    uint8_t output[3] = {0U, 0U, 0U};
    size_t output_length = sizeof(output);
    size_t in_use_count = 0U;

    assert(DEMO_PoolInit(&pool) == DEMO_POOL_RESULT_OK);
    assert(DEMO_PoolAcquire(&pool, &old_handle) == DEMO_POOL_RESULT_OK);
    assert(DEMO_PoolWrite(&pool, old_handle, input, sizeof(input)) == DEMO_POOL_RESULT_OK);
    assert(DEMO_PoolRead(&pool, old_handle, output, &output_length) == DEMO_POOL_RESULT_OK);
    assert(output_length == sizeof(input));
    assert(memcmp(output, input, sizeof(input)) == 0);
    assert(DEMO_PoolRelease(&pool, old_handle) == DEMO_POOL_RESULT_OK);
    assert(DEMO_PoolAcquire(&pool, &new_handle) == DEMO_POOL_RESULT_OK);
    output_length = sizeof(output);
    assert(DEMO_PoolRead(&pool, old_handle, output, &output_length) ==
           DEMO_POOL_RESULT_INVALID_HANDLE);
    assert(new_handle.generation != old_handle.generation);
    assert(DEMO_PoolGetInUseCount(&pool, &in_use_count) == DEMO_POOL_RESULT_OK);
    assert(in_use_count == 1U);
}

/**
 * @brief 验证公开接口对运行时错误和错误注入的确定返回值。
 *
 * @details
 * 注入缺失临界区回调、空指针、错误状态、错误帧长度和超长资源数据，确认错误不会被断言替代。
 *
 * @return 无。
 */
static void DEMO_TestFaultInjection(void)
{
    demo_context_t context;
    demo_pool_t pool;
    demo_pool_handle_t handle;
    demo_test_lock_t lock = {0U, 0U};
    const demo_critical_section_t incomplete = {DEMO_TestEnterCritical, NULL, &lock};
    uint8_t frame[DEMO_PROTOCOL_FRAME_SIZE] = {0U};
    uint8_t oversized[DEMO_POOL_PAYLOAD_CAPACITY + 1] = {0U};

    assert(DEMO_Init(&context, &incomplete) == DEMO_RESULT_INVALID_ARGUMENT);
    assert(DEMO_Start(NULL) == DEMO_RESULT_INVALID_ARGUMENT);
    assert(DEMO_DecodeFrame(frame, 1U, NULL) == DEMO_PROTOCOL_RESULT_INVALID_ARGUMENT);
    assert(DEMO_PoolInit(&pool) == DEMO_POOL_RESULT_OK);
    assert(DEMO_PoolAcquire(&pool, &handle) == DEMO_POOL_RESULT_OK);
    assert(DEMO_PoolWrite(&pool, handle, oversized, sizeof(oversized)) ==
           DEMO_POOL_RESULT_INVALID_ARGUMENT);
    assert(DEMO_PoolRelease(&pool, handle) == DEMO_POOL_RESULT_OK);
    assert(DEMO_PoolRelease(&pool, handle) == DEMO_POOL_RESULT_INVALID_HANDLE);
}
