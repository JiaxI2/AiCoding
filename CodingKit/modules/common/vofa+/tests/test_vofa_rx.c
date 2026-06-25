#include "vofa.h"

#include <math.h>
#include <stdint.h>
#include <stdio.h>

static float g_pending_value;
static uint16_t g_pending_object;
static uint16_t g_pending_parameter;
static uint32_t g_pending_count;
static uint32_t g_process_count;
static uint32_t g_status_count;
static float g_sample_rate;
static bool g_stream;

static int32_t test_write(void *user, const vofa_octet_t *data, uint32_t length)
{
    (void)user;
    (void)data;
    return (int32_t)length;
}

void Vofa_AppInit(void) {}
void Vofa_AppProcess(void) { ++g_process_count; }
vofa_result_t Vofa_AppWriteParameterPending(uint16_t object_id, uint16_t parameter_id, float value)
{
    g_pending_object = object_id;
    g_pending_parameter = parameter_id;
    g_pending_value = value;
    ++g_pending_count;
    return VOFA_OK;
}
vofa_result_t Vofa_AppReadParameter(uint16_t object_id, uint16_t parameter_id, float *value)
{
    (void)object_id; (void)parameter_id; if (value != NULL) { *value = 2.0F; } return VOFA_OK;
}
vofa_result_t Vofa_AppSaveParameters(void) { return VOFA_OK; }
vofa_result_t Vofa_AppLoadDefaults(void) { return VOFA_OK; }
vofa_result_t Vofa_AppSetSampleRate(float sample_rate_hz) { g_sample_rate = sample_rate_hz; return VOFA_OK; }
uint32_t Vofa_AppGetStatus(void) { ++g_status_count; return 0x55UL; }
void Vofa_AppStartStream(void) { g_stream = true; }
void Vofa_AppStopStream(void) { g_stream = false; }

static uint16_t crc16(const vofa_octet_t *data, uint16_t length)
{
    uint16_t crc = 0xFFFFU;
    uint16_t i;

    for (i = 0U; i < length; ++i)
    {
        uint16_t bit;
        crc ^= (uint16_t)((data[i] & 0xFFU) << 8U);
        for (bit = 0U; bit < 8U; ++bit)
        {
            if ((crc & 0x8000U) != 0U)
            {
                crc = (uint16_t)((crc << 1U) ^ 0x1021U);
            }
            else
            {
                crc = (uint16_t)(crc << 1U);
            }
        }
    }
    return crc;
}

static void encode_float(float value, vofa_octet_t out[4])
{
    union { float f; uint32_t u; } bits;

    bits.f = value;
    out[0] = (vofa_octet_t)(bits.u & 0xFFU);
    out[1] = (vofa_octet_t)((bits.u >> 8U) & 0xFFU);
    out[2] = (vofa_octet_t)((bits.u >> 16U) & 0xFFU);
    out[3] = (vofa_octet_t)((bits.u >> 24U) & 0xFFU);
}

static uint16_t build_frame(vofa_octet_t *frame, uint8_t cmd, float value, bool bad_crc)
{
    vofa_octet_t payload[4];
    uint16_t index = 0U;
    uint16_t crc;

    encode_float(value, payload);
    frame[index++] = VOFA_NEW_HEADER0;
    frame[index++] = VOFA_NEW_HEADER1;
    frame[index++] = VOFA_NEW_VERSION;
    frame[index++] = cmd;
    frame[index++] = 0x11U;
    frame[index++] = 0x01U;
    frame[index++] = 0x00U;
    frame[index++] = 0x02U;
    frame[index++] = 0x00U;
    frame[index++] = (cmd == VOFA_CMD_GET_STATUS) ? VOFA_DATA_NONE : VOFA_DATA_FLOAT32;
    frame[index++] = (cmd == VOFA_CMD_GET_STATUS) ? 0U : 4U;
    if (cmd != VOFA_CMD_GET_STATUS)
    {
        frame[index++] = payload[0];
        frame[index++] = payload[1];
        frame[index++] = payload[2];
        frame[index++] = payload[3];
    }
    crc = crc16(frame, index);
    if (bad_crc)
    {
        crc ^= 1U;
    }
    frame[index++] = (vofa_octet_t)(crc & 0xFFU);
    frame[index++] = (vofa_octet_t)((crc >> 8U) & 0xFFU);
    return index;
}

int main(void)
{
    vofa_octet_t frame[32];
    vofa_octet_t legacy[8];
    uint16_t length;
    const vofa_statistics_t *stats;

    Vofa_Init(test_write, NULL, NULL);

    legacy[0] = 0xAAU;
    legacy[1] = 0xFFU;
    legacy[2] = 0x05U;
    legacy[3] = 0x06U;
    encode_float(1.0F, &legacy[4]);
    Vofa_RxFeed(legacy, 8U);
    if (g_pending_count != 0U)
    {
        printf("legacy executed before process\n");
        return 1;
    }
    Vofa_Process();
    if ((g_pending_count != 1U) || (g_pending_object != 0x05U) || (g_pending_parameter != 0x06U) ||
        (fabsf(g_pending_value - 1.0F) > 0.0001F))
    {
        printf("legacy process failed\n");
        return 1;
    }

    length = build_frame(frame, VOFA_CMD_WRITE_PARAMETER, 3.5F, false);
    Vofa_RxFeed(frame, length);
    if (g_pending_count != 1U)
    {
        printf("new executed before process\n");
        return 1;
    }
    Vofa_Process();
    if ((g_pending_count != 2U) || (g_pending_object != 0x0001U) || (g_pending_parameter != 0x0002U) ||
        (fabsf(g_pending_value - 3.5F) > 0.0001F))
    {
        printf("new process failed\n");
        return 1;
    }

    length = build_frame(frame, VOFA_CMD_WRITE_PARAMETER, 4.5F, true);
    Vofa_RxFeed(frame, length);
    Vofa_Process();
    if (g_pending_count != 2U)
    {
        printf("bad crc accepted\n");
        return 1;
    }

    length = build_frame(frame, VOFA_CMD_SET_SAMPLE_RATE, 1000.0F, false);
    Vofa_RxFeed(frame, length);
    Vofa_Process();
    if (fabsf(g_sample_rate - 1000.0F) > 0.0001F)
    {
        printf("sample rate failed\n");
        return 1;
    }

    length = build_frame(frame, VOFA_CMD_START_STREAM, 0.0F, false);
    frame[9] = VOFA_DATA_NONE;
    frame[10] = 0U;
    {
        uint16_t crc = crc16(frame, 11U);
        frame[11] = (vofa_octet_t)(crc & 0xFFU);
        frame[12] = (vofa_octet_t)((crc >> 8U) & 0xFFU);
        length = 13U;
    }
    Vofa_RxFeed(frame, length);
    Vofa_Process();
    if (!g_stream)
    {
        printf("stream start failed\n");
        return 1;
    }

    length = build_frame(frame, VOFA_CMD_GET_STATUS, 0.0F, false);
    Vofa_RxFeed(frame, length);
    Vofa_Process();
    if (g_status_count == 0U)
    {
        printf("status failed\n");
        return 1;
    }

    Vofa_RxFeed(frame, 1U);
    for (length = 0U; length < (VOFA_RX_TIMEOUT_TICKS + 2U); ++length)
    {
        Vofa_Process();
    }
    stats = Vofa_GetStatistics();
    if ((stats->crc_error_count == 0U) || (stats->rx_timeout_count == 0U) || (g_process_count == 0U))
    {
        printf("stats failed\n");
        return 1;
    }

    printf("test_vofa_rx passed\n");
    return 0;
}