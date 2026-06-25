#ifndef AI_DEBUG_TARGET_SHIM_H
#define AI_DEBUG_TARGET_SHIM_H

/**
 * @file ai_debug_target_shim.h
 * @brief Minimal fixed-resource target telemetry shim for AI Debug Kit.
 *
 * Modification record
 * Date         Author      Reason
 * 2026-06-25   HUJIAXUAN   Add non-blocking target shim template.
 */

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#ifndef AI_DEBUG_ENABLE
#define AI_DEBUG_ENABLE                 (1U)
#endif

#ifndef AI_DEBUG_SAMPLE_CAPACITY
#define AI_DEBUG_SAMPLE_CAPACITY        (64U)
#endif

#if (AI_DEBUG_SAMPLE_CAPACITY == 0U)
#error "AI_DEBUG_SAMPLE_CAPACITY must be greater than zero."
#endif

typedef struct
{
    uint32_t sequence;
    uint32_t timestamp_ticks;
    uint16_t signal_id;
    uint16_t flags;
    uint32_t raw_value;
} AiDebugSample32;

typedef struct
{
    uint32_t buffer_address;
    uint32_t buffer_size;
    uint32_t max_sample_rate_hz;
    uint16_t max_channels;
    uint16_t flags;
} AiDebugConfig;

typedef enum
{
    AI_DEBUG_OK = 0,
    AI_DEBUG_ERR_CONFIG = -1,
    AI_DEBUG_ERR_FULL = -2,
    AI_DEBUG_ERR_RANGE = -3,
    AI_DEBUG_ERR_STATE = -4
} AiDebugStatus;

typedef struct
{
    uint32_t pushed_count;
    uint32_t popped_count;
    uint32_t dropped_count;
    uint32_t snapshot_count;
} AiDebugStats;

/** @brief Initialize the fixed target shim state. */
AiDebugStatus AiDebug_Init(const AiDebugConfig *config);

/** @brief Push one 32-bit sample from ISR or task context without transport work. */
AiDebugStatus AiDebug_PushSampleU32(uint16_t signal_id, uint32_t timestamp_ticks, uint32_t value);

/** @brief Publish a bounded snapshot metadata record. */
AiDebugStatus AiDebug_PublishSnapshot(uint16_t snapshot_id, const void *data, uint16_t octet_length);

/** @brief Drain one pending sample into application-owned storage when provided. */
AiDebugStatus AiDebug_Service(void);

/** @brief Copy current shim counters to caller-owned storage. */
AiDebugStatus AiDebug_GetStats(AiDebugStats *stats);

#if (AI_DEBUG_ENABLE == 0U)
#define AiDebug_PushSampleU32(signal_id, timestamp_ticks, value) (AI_DEBUG_OK)
#endif

#ifdef __cplusplus
}
#endif

#endif /* AI_DEBUG_TARGET_SHIM_H */
