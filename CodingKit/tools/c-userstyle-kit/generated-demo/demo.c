/**
 * @file demo.c
 * @brief 实现有界采样平均值计算和等级判定。
 * @copyright Copyright (c) 2026 HU JIAXUAN.
 * @date 2026-07-15
 * @author HU JIAXUAN
 *
 * @details
 * 文件内容：实现参数校验、有界累加、平均值计算和等级名称转换。
 * 主要功能：提供一个无需硬件依赖、能够直接阅读和运行的 C99 示例。
 * 文件关系：实现 demo.h；高级状态机、协议和资源池样例位于 advanced/。
 */

#include "demo.h"

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

static bool DEMO_IsInputValid(const uint16_t *samples,
                              size_t sample_count,
                              const demo_thresholds_t *thresholds,
                              const demo_sample_summary_t *summary);
static uint16_t DEMO_CalculateSampleAverage(const uint16_t *samples, size_t sample_count);
static demo_sample_level_t DEMO_ClassifyAverage(uint16_t average,
                                                 const demo_thresholds_t *thresholds);

/**
 * @brief 计算一组有界采样的平均值和等级。
 *
 * @details
 * 实现先一次性验证全部外部输入，再用 32 位累加器扫描最多 8 个 16 位采样。
 * 所有输出在计算完成后统一发布，因此失败路径不会给调用方留下部分更新的摘要。
 *
 * @param[in] samples 输入采样数组，不允许为 NULL。
 * @param[in] sample_count 输入元素数量，范围为 1 到 DEMO_SAMPLE_MAX_COUNT。
 * @param[in] thresholds 等级阈值，不允许为 NULL，且 low 不得大于 high。
 * @param[out] summary 成功时接收平均值和等级，不允许为 NULL。
 *
 * @return DEMO_EVALUATE_RESULT_OK 评估成功；其他值表示输入参数无效。
 */
demo_evaluate_result_t DEMO_EvaluateSamples(const uint16_t *samples,
                                             size_t sample_count,
                                             const demo_thresholds_t *thresholds,
                                             demo_sample_summary_t *summary)
{
    demo_evaluate_result_t result = DEMO_EVALUATE_RESULT_INVALID_ARGUMENT;

    /* 参数验证失败时不进入计算，summary 保持调用方原值。 */
    if (DEMO_IsInputValid(samples, sample_count, thresholds, summary))
    {
        const uint16_t average = DEMO_CalculateSampleAverage(samples, sample_count);
        const demo_sample_level_t level = DEMO_ClassifyAverage(average, thresholds);

        /* 两个结果来自同一次输入快照，完成计算后再一并发布。 */
        summary->average = average;
        summary->level = level;
        result = DEMO_EVALUATE_RESULT_OK;
    }

    return result;
}

/**
 * @brief 返回采样等级的固定英文名称。
 *
 * @details
 * 使用完整 switch 明确处理每个公开枚举值；default 将损坏或越界枚举收敛为 unknown。
 * 返回值均指向只读字符串常量，不需要调用方释放。
 *
 * @param[in] level 待转换的采样等级。
 *
 * @return 合法等级的英文名称；非法枚举值返回 "unknown"。
 */
const char *DEMO_GetLevelName(demo_sample_level_t level)
{
    const char *name = "unknown";

    switch (level)
    {
        case DEMO_SAMPLE_LEVEL_LOW:
            /* 低等级用于表示平均值尚未达到正常下界。 */
            name = "low";
            break;

        case DEMO_SAMPLE_LEVEL_NORMAL:
            /* 正常等级包含 low 和 high 两个阈值边界。 */
            name = "normal";
            break;

        case DEMO_SAMPLE_LEVEL_HIGH:
            /* 高等级用于表示平均值已经超过正常上界。 */
            name = "high";
            break;

        default:
            /* 非法枚举值保留统一的 unknown 文本，避免返回 NULL。 */
            break;
    }

    return name;
}

/**
 * @brief 检查采样评估所需的全部外部输入。
 *
 * @details
 * 指针、数量和阈值在任何解引用前集中验证，使主流程只处理合法数据。
 * 数量上限同时证明后续数组访问和 32 位累加均不会越界或溢出。
 *
 * @param[in] samples 输入采样数组。
 * @param[in] sample_count 输入元素数量。
 * @param[in] thresholds 等级阈值。
 * @param[in] summary 输出摘要。
 *
 * @return true 表示全部输入满足契约；false 表示至少一项无效。
 */
static bool DEMO_IsInputValid(const uint16_t *samples,
                              size_t sample_count,
                              const demo_thresholds_t *thresholds,
                              const demo_sample_summary_t *summary)
{
    bool valid = false;

    /* 只有全部指针和数量有效时才读取阈值；否则 valid 保持 false。 */
    if ((samples != NULL) && (thresholds != NULL) && (summary != NULL) &&
        (sample_count > 0U) && (sample_count <= (size_t)DEMO_SAMPLE_MAX_COUNT))
    {
        /* 反向阈值没有区间语义，只有有序边界才接受本次评估。 */
        if (thresholds->low <= thresholds->high)
        {
            valid = true;
        }
    }

    return valid;
}

/**
 * @brief 计算已经验证的采样数组整数平均值。
 *
 * @details
 * 8 个 uint16_t 最大值之和不超过 uint32_t。先累加后除法只发生一次截断，
 * 比逐项除法更准确；调用方已经证明指针和数量合法。
 *
 * @param[in] samples 已验证的采样数组。
 * @param[in] sample_count 已验证的元素数量。
 *
 * @return 向下取整且仍处于 uint16_t 范围内的平均值。
 */
static uint16_t DEMO_CalculateSampleAverage(const uint16_t *samples, size_t sample_count)
{
    uint32_t sum = 0U;
    size_t index = 0U;

    for (index = 0U; index < sample_count; ++index)
    {
        /* 显式提升后累加，避免窄类型算术和隐式转换掩盖范围。 */
        sum += (uint32_t)samples[index];
    }

    return (uint16_t)(sum / (uint32_t)sample_count);
}

/**
 * @brief 按已验证阈值判定平均值等级。
 *
 * @details
 * 判断顺序直接表达三个互斥区间；low 和 high 边界均归入正常等级。
 * 输入均为无符号 16 位值，不存在符号转换或算术溢出。
 *
 * @param[in] average 已计算的平均值。
 * @param[in] thresholds 已验证的等级阈值。
 *
 * @return 平均值对应的低、正常或高等级。
 */
static demo_sample_level_t DEMO_ClassifyAverage(uint16_t average,
                                                 const demo_thresholds_t *thresholds)
{
    demo_sample_level_t level = DEMO_SAMPLE_LEVEL_NORMAL;

    /* 默认覆盖闭区间 [low, high]，两个显式分支只处理区间外值。 */
    if (average < thresholds->low)
    {
        level = DEMO_SAMPLE_LEVEL_LOW;
    }
    else if (average > thresholds->high)
    {
        level = DEMO_SAMPLE_LEVEL_HIGH;
    }

    return level;
}
