/**
 * @file demo.h
 * @brief 计算有界采样平均值并给出等级判定的简单公开接口。
 * @copyright Copyright (c) 2026 HU JIAXUAN.
 * @version 1.2.0
 * @date 2026-07-15
 * @author HU JIAXUAN
 *
 * @details
 * 文件内容：声明采样阈值、计算结果和两个简单公开函数。
 * 主要功能：对最多 8 个无符号采样计算整数平均值，并判定低、正常或高等级。
 * 文件关系：由 demo.c 实现；advanced/ 提供独立的完整规则覆盖样例。
 */

#ifndef DEMO_H
#define DEMO_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 单次评估允许读取的最大采样数量。
 */
enum
{
    DEMO_SAMPLE_MAX_COUNT = 8
};

/**
 * @brief 采样评估操作结果。
 */
typedef enum
{
    DEMO_EVALUATE_RESULT_OK = 0,
    DEMO_EVALUATE_RESULT_INVALID_ARGUMENT
} demo_evaluate_result_t;

/**
 * @brief 平均值相对于阈值的等级。
 */
typedef enum
{
    DEMO_SAMPLE_LEVEL_LOW = 0,
    DEMO_SAMPLE_LEVEL_NORMAL,
    DEMO_SAMPLE_LEVEL_HIGH
} demo_sample_level_t;

/**
 * @brief 采样等级阈值。
 *
 * @details low 必须小于或等于 high；两个边界都属于正常等级。
 */
typedef struct
{
    uint16_t low;   /**< 平均值低于该值时判定为低等级。 */
    uint16_t high;  /**< 平均值高于该值时判定为高等级。 */
} demo_thresholds_t;

/**
 * @brief 一次采样评估的输出摘要。
 */
typedef struct
{
    uint16_t average;          /**< 向下取整后的整数平均值。 */
    demo_sample_level_t level; /**< 平均值对应的等级。 */
} demo_sample_summary_t;

/**
 * @brief 计算一组有界采样的平均值和等级。
 *
 * @param[in] samples 输入采样数组，不允许为 NULL。
 * @param[in] sample_count 输入元素数量，范围为 1 到 DEMO_SAMPLE_MAX_COUNT。
 * @param[in] thresholds 等级阈值，不允许为 NULL，且 low 不得大于 high。
 * @param[out] summary 成功时接收平均值和等级，不允许为 NULL。
 *
 * @return DEMO_EVALUATE_RESULT_OK 评估成功；其他值表示输入参数无效。
 *
 * @note 性能与 sample_count 线性相关且最多扫描 8 个元素；函数可重入、不共享可变状态，
 *       可由不同任务并发处理不同输入，但不应在中断中传入生命周期不确定的缓冲区。
 */
demo_evaluate_result_t DEMO_EvaluateSamples(const uint16_t *samples,
                                             size_t sample_count,
                                             const demo_thresholds_t *thresholds,
                                             demo_sample_summary_t *summary);

/**
 * @brief 返回采样等级的固定英文名称。
 *
 * @param[in] level 待转换的采样等级。
 *
 * @return 合法等级返回 "low"、"normal" 或 "high"；非法枚举值返回 "unknown"。
 *
 * @note 性能为常量时间；返回只读静态字符串，函数可重入且不得修改返回内容。
 */
const char *DEMO_GetLevelName(demo_sample_level_t level);

#ifdef __cplusplus
}
#endif

#endif /* DEMO_H */
