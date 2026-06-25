#include "vofa.h"

#include <math.h>
#include <stdint.h>
#include <stdio.h>

static vofa_octet_t g_tx[512];
static uint32_t g_tx_count;
static bool g_busy;

static int32_t test_write(void *user, const vofa_octet_t *data, uint32_t length)
{
    uint32_t i;

    (void)user;
    if (g_busy)
    {
        return 0;
    }
    if ((data == NULL) && (length != 0U))
    {
        return -1;
    }
    for (i = 0U; i < length; ++i)
    {
        g_tx[g_tx_count++] = (vofa_octet_t)(data[i] & 0xFFU);
    }
    return (int32_t)length;
}

void Vofa_AppInit(void) {}
void Vofa_AppProcess(void) {}
vofa_result_t Vofa_AppWriteParameterPending(uint16_t object_id, uint16_t parameter_id, float value)
{
    (void)object_id; (void)parameter_id; (void)value; return VOFA_OK;
}
vofa_result_t Vofa_AppReadParameter(uint16_t object_id, uint16_t parameter_id, float *value)
{
    (void)object_id; (void)parameter_id; if (value != NULL) { *value = 0.0F; } return VOFA_OK;
}
vofa_result_t Vofa_AppSaveParameters(void) { return VOFA_OK; }
vofa_result_t Vofa_AppLoadDefaults(void) { return VOFA_OK; }
vofa_result_t Vofa_AppSetSampleRate(float sample_rate_hz) { (void)sample_rate_hz; return VOFA_OK; }
uint32_t Vofa_AppGetStatus(void) { return 0UL; }
void Vofa_AppStartStream(void) {}
void Vofa_AppStopStream(void) {}

static int expect(uint32_t index, uint16_t expected)
{
    if ((g_tx[index] & 0xFFU) != expected)
    {
        printf("octet[%lu] expected 0x%02X got 0x%02X\n", (unsigned long)index, expected, g_tx[index] & 0xFFU);
        return 1;
    }
    return 0;
}

static void clear_tx(void)
{
    g_tx_count = 0U;
}

int main(void)
{
    float values[4];
    const vofa_statistics_t *stats;
    int failed = 0;

    values[0] = 0.0F;
    values[1] = 1.0F;
    values[2] = -1.0F;
    values[3] = INFINITY;

    Vofa_Init(test_write, NULL, NULL);
    if (Vofa_SendChannels(values, 4U) != VOFA_OK)
    {
        printf("blocking send failed\n");
        return 1;
    }

    if (g_tx_count != 20U)
    {
        printf("tx_count=%lu\n", (unsigned long)g_tx_count);
        return 1;
    }

    failed += expect(4U, 0x00U);
    failed += expect(5U, 0x00U);
    failed += expect(6U, 0x80U);
    failed += expect(7U, 0x3FU);
    failed += expect(8U, 0x00U);
    failed += expect(9U, 0x00U);
    failed += expect(10U, 0x80U);
    failed += expect(11U, 0xBFU);
    failed += expect(12U, 0x00U);
    failed += expect(13U, 0x00U);
    failed += expect(14U, 0x80U);
    failed += expect(15U, 0x7FU);
    failed += expect(16U, VOFA_JUSTFLOAT_TAIL0);
    failed += expect(17U, VOFA_JUSTFLOAT_TAIL1);
    failed += expect(18U, VOFA_JUSTFLOAT_TAIL2);
    failed += expect(19U, VOFA_JUSTFLOAT_TAIL3);

    clear_tx();
    if (Vofa_SendChannels(NULL, 0U) != VOFA_OK)
    {
        printf("zero channel failed\n");
        return 1;
    }
    if (g_tx_count != 4U)
    {
        printf("zero tx_count=%lu\n", (unsigned long)g_tx_count);
        return 1;
    }

    clear_tx();
    if (Vofa_SendChannelsEx(values, 3U, VOFA_TX_INTERRUPT) != VOFA_OK)
    {
        printf("interrupt start failed\n");
        return 1;
    }
    if (Vofa_SendChannelsEx(values, 3U, VOFA_TX_INTERRUPT) != VOFA_ERROR_BUSY)
    {
        printf("busy not reported\n");
        return 1;
    }
    while (Vofa_IsTxBusy())
    {
        Vofa_TxEventHandler();
    }
    if (g_tx_count != 16U)
    {
        printf("interrupt tx_count=%lu\n", (unsigned long)g_tx_count);
        return 1;
    }

    g_busy = true;
    if (Vofa_SendChannels(values, 1U) != VOFA_ERROR_TIMEOUT)
    {
        printf("timeout not reported\n");
        return 1;
    }
    g_busy = false;

    stats = Vofa_GetStatistics();
    if ((stats->tx_frame_count < 3U) || (stats->tx_busy_count == 0U) || (stats->tx_timeout_count == 0U))
    {
        printf("stats failed\n");
        return 1;
    }

    if (failed != 0)
    {
        return 1;
    }
    printf("test_vofa_encode passed\n");
    return 0;
}