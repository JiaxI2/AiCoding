#include "ai_debug_target_shim.h"

/*
 * Modification record
 * Date         Author      Reason
 * 2026-06-25   HUJIAXUAN   Add fixed-capacity drop-newest target shim.
 */

#if (AI_DEBUG_ENABLE != 0U)

typedef struct
{
    bool initialized;
    uint32_t sequence;
    uint16_t write_index;
    uint16_t read_index;
    AiDebugSample32 samples[AI_DEBUG_SAMPLE_CAPACITY];
    AiDebugStats stats;
} AiDebugContext;

static AiDebugContext g_aiDebug;

/** @brief Return the next circular index for a bounded queue. */
static uint16_t ai_debug_next_index(uint16_t index)
{
    ++index;
    if (index >= AI_DEBUG_SAMPLE_CAPACITY)
    {
        index = 0U;
    }
    return index;
}

AiDebugStatus AiDebug_Init(const AiDebugConfig *config)
{
    if (config == NULL)
    {
        return AI_DEBUG_ERR_CONFIG;
    }

    if ((config->max_channels == 0U) || (config->max_sample_rate_hz == 0UL))
    {
        return AI_DEBUG_ERR_CONFIG;
    }

    g_aiDebug.initialized = true;
    g_aiDebug.sequence = 0UL;
    g_aiDebug.write_index = 0U;
    g_aiDebug.read_index = 0U;
    g_aiDebug.stats.pushed_count = 0UL;
    g_aiDebug.stats.popped_count = 0UL;
    g_aiDebug.stats.dropped_count = 0UL;
    g_aiDebug.stats.snapshot_count = 0UL;

    return AI_DEBUG_OK;
}

AiDebugStatus AiDebug_PushSampleU32(uint16_t signal_id, uint32_t timestamp_ticks, uint32_t value)
{
    uint16_t next;
    AiDebugSample32 *sample;

    if (!g_aiDebug.initialized)
    {
        return AI_DEBUG_ERR_STATE;
    }

    next = ai_debug_next_index(g_aiDebug.write_index);
    if (next == g_aiDebug.read_index)
    {
        ++g_aiDebug.stats.dropped_count;
        return AI_DEBUG_ERR_FULL;
    }

    sample = &g_aiDebug.samples[g_aiDebug.write_index];
    sample->sequence = g_aiDebug.sequence;
    sample->timestamp_ticks = timestamp_ticks;
    sample->signal_id = signal_id;
    sample->flags = 0U;
    sample->raw_value = value;

    ++g_aiDebug.sequence;
    ++g_aiDebug.stats.pushed_count;
    g_aiDebug.write_index = next;

    return AI_DEBUG_OK;
}

AiDebugStatus AiDebug_PublishSnapshot(uint16_t snapshot_id, const void *data, uint16_t octet_length)
{
    (void)data;

    if (!g_aiDebug.initialized)
    {
        return AI_DEBUG_ERR_STATE;
    }

    if ((snapshot_id == 0U) || (octet_length == 0U))
    {
        return AI_DEBUG_ERR_RANGE;
    }

    ++g_aiDebug.stats.snapshot_count;
    return AI_DEBUG_OK;
}

AiDebugStatus AiDebug_Service(void)
{
    if (!g_aiDebug.initialized)
    {
        return AI_DEBUG_ERR_STATE;
    }

    if (g_aiDebug.read_index == g_aiDebug.write_index)
    {
        return AI_DEBUG_OK;
    }

    g_aiDebug.read_index = ai_debug_next_index(g_aiDebug.read_index);
    ++g_aiDebug.stats.popped_count;
    return AI_DEBUG_OK;
}

AiDebugStatus AiDebug_GetStats(AiDebugStats *stats)
{
    if (stats == NULL)
    {
        return AI_DEBUG_ERR_RANGE;
    }

    *stats = g_aiDebug.stats;
    return AI_DEBUG_OK;
}

#else

AiDebugStatus AiDebug_Init(const AiDebugConfig *config)
{
    (void)config;
    return AI_DEBUG_OK;
}

AiDebugStatus AiDebug_PublishSnapshot(uint16_t snapshot_id, const void *data, uint16_t octet_length)
{
    (void)snapshot_id;
    (void)data;
    (void)octet_length;
    return AI_DEBUG_OK;
}

AiDebugStatus AiDebug_Service(void)
{
    return AI_DEBUG_OK;
}

AiDebugStatus AiDebug_GetStats(AiDebugStats *stats)
{
    if (stats != NULL)
    {
        stats->pushed_count = 0UL;
        stats->popped_count = 0UL;
        stats->dropped_count = 0UL;
        stats->snapshot_count = 0UL;
    }
    return AI_DEBUG_OK;
}

#endif
