#include "vofa.h"

#include <float.h>
#include <limits.h>
#include <math.h>

/*
 * 修改记录
 * 日期         作者        原因
 * 2026-06-23   HUJIAXUAN   简化 Port 注册，统一 RX 分流，命令全部队列化并在 Vofa_Process 执行。
 */

typedef char vofa_float_must_be_32_bits[((sizeof(float) * CHAR_BIT) == 32U) ? 1 : -1];
typedef char vofa_uint32_must_be_32_bits[((sizeof(uint32_t) * CHAR_BIT) == 32U) ? 1 : -1];

#if (FLT_RADIX != 2) || (FLT_MANT_DIG != 24) || (FLT_MAX_EXP != 128)
#error "VOFA JustFloat requires IEEE-754 binary32 float format."
#endif

typedef union
{
    float value;
    uint32_t bits;
} vofa_float_bits_t;

typedef enum
{
    VOFA_RX_IDLE = 0,
    VOFA_RX_LEGACY_FF,
    VOFA_RX_LEGACY_TYPE,
    VOFA_RX_LEGACY_ID,
    VOFA_RX_LEGACY_VALUE,
    VOFA_RX_NEW_HEADER1,
    VOFA_RX_NEW_FIXED,
    VOFA_RX_NEW_PAYLOAD,
    VOFA_RX_NEW_CRC0,
    VOFA_RX_NEW_CRC1
} vofa_rx_state_t;

typedef struct
{
    uint8_t command;
    uint8_t sequence;
    uint16_t object_id;
    uint16_t parameter_id;
    uint8_t data_type;
    uint8_t length;
    vofa_octet_t payload[VOFA_FLOAT_OCTETS];
    bool from_new_protocol;
} vofa_command_t;

typedef struct
{
    bool initialized;
    vofa_write_fn write_fn;
    vofa_read_fn read_fn;
    void *user;

    volatile bool tx_busy;
    uint16_t tx_channel_count;
    uint32_t tx_octet_index;
    uint32_t tx_total_octets;
    float tx_channels[VOFA_MAX_CHANNELS];

    vofa_rx_state_t rx_state;
    uint16_t rx_index;
    uint16_t rx_payload_length;
    uint16_t rx_idle_ticks;
    vofa_octet_t rx_frame[VOFA_RX_BUFFER_SIZE];

    volatile uint16_t queue_head;
    volatile uint16_t queue_tail;
    vofa_command_t queue[VOFA_COMMAND_QUEUE_SIZE];

    vofa_statistics_t statistics;
    vofa_memory_port_t memory_port;
} vofa_context_t;

static vofa_context_t g_vofa;

/** @brief 统计计数饱和加 1。 */
static void vofa_stat_inc(uint32_t *value)
{
    if (*value < UINT32_MAX)
    {
        ++(*value);
    }
}

/** @brief 只保留线上 octet 的低 8 位，兼容 C28x 16-bit char。 */
static vofa_octet_t vofa_octet(vofa_octet_t value)
{
    return (vofa_octet_t)(value & 0x00FFU);
}

/** @brief float 到 JustFloat 小端 octet；不依赖 CPU 内存大小端。 */
static void vofa_encode_float_le(float value, vofa_octet_t out[VOFA_FLOAT_OCTETS])
{
    vofa_float_bits_t bits;

    bits.value = value;
    out[0] = (vofa_octet_t)(bits.bits & 0xFFU);
    out[1] = (vofa_octet_t)((bits.bits >> 8U) & 0xFFU);
    out[2] = (vofa_octet_t)((bits.bits >> 16U) & 0xFFU);
    out[3] = (vofa_octet_t)((bits.bits >> 24U) & 0xFFU);
}

/** @brief JustFloat 小端 octet 到 float；不依赖 CPU 内存大小端。 */
static float vofa_decode_float_le(const vofa_octet_t in[VOFA_FLOAT_OCTETS])
{
    vofa_float_bits_t bits;

    bits.bits =
        ((uint32_t)(in[0] & 0xFFU)) |
        ((uint32_t)(in[1] & 0xFFU) << 8U) |
        ((uint32_t)(in[2] & 0xFFU) << 16U) |
        ((uint32_t)(in[3] & 0xFFU) << 24U);

    return bits.value;
}

/** @brief CRC16-CCITT-FALSE，用于新控制协议帧校验。 */
static uint16_t vofa_crc16(const vofa_octet_t *data, uint16_t length)
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

/** @brief 生成 JustFloat 帧中的某一个 octet。 */
static vofa_result_t vofa_make_channel_octet(
    const float *channels,
    uint16_t channel_count,
    uint32_t index,
    vofa_octet_t *octet)
{
    uint32_t data_octets = (uint32_t)channel_count * VOFA_FLOAT_OCTETS;

    if (octet == NULL)
    {
        return VOFA_ERROR_INVALID_ARGUMENT;
    }

    if (index < data_octets)
    {
        uint16_t channel = (uint16_t)(index / VOFA_FLOAT_OCTETS);
        uint16_t byte_index = (uint16_t)(index % VOFA_FLOAT_OCTETS);
        vofa_octet_t encoded[VOFA_FLOAT_OCTETS];

        vofa_encode_float_le(channels[channel], encoded);
        *octet = encoded[byte_index];
        return VOFA_OK;
    }

    switch (index - data_octets)
    {
        case 0U: *octet = VOFA_JUSTFLOAT_TAIL0; return VOFA_OK;
        case 1U: *octet = VOFA_JUSTFLOAT_TAIL1; return VOFA_OK;
        case 2U: *octet = VOFA_JUSTFLOAT_TAIL2; return VOFA_OK;
        case 3U: *octet = VOFA_JUSTFLOAT_TAIL3; return VOFA_OK;
        default: break;
    }

    return VOFA_ERROR_RANGE;
}

#if (VOFA_PORT_TYPE == VOFA_PORT_MEMORY_VALUE)
/** @brief MEMORY Port 写内部环形缓冲，避免中断发送无人消费时卡死。 */
static int32_t vofa_memory_write(const vofa_octet_t *data, uint32_t length)
{
    uint32_t i;

    for (i = 0U; i < length; ++i)
    {
        uint32_t next = (g_vofa.memory_port.write_index + 1UL) % VOFA_MEMORY_BUFFER_SIZE;

        if (next == g_vofa.memory_port.read_index)
        {
            g_vofa.memory_port.read_index = (g_vofa.memory_port.read_index + 1UL) % VOFA_MEMORY_BUFFER_SIZE;
            g_vofa.memory_port.overflow_count++;
            vofa_stat_inc(&g_vofa.statistics.memory_overflow_count);
        }

        g_vofa.memory_port.buffer[g_vofa.memory_port.write_index] = vofa_octet(data[i]);
        g_vofa.memory_port.write_index = next;
        g_vofa.memory_port.sequence++;
    }

    return (int32_t)length;
}
#endif

/** @brief 编译期 Port 分派。SCI/User/EtherCAT 使用 Init 注册的写函数；Memory 使用内部缓冲。 */
static int32_t vofa_port_write(const vofa_octet_t *data, uint32_t length)
{
#if (VOFA_PORT_TYPE == VOFA_PORT_MEMORY_VALUE)
    return vofa_memory_write(data, length);
#elif (VOFA_PORT_TYPE == VOFA_PORT_SCI_VALUE) || \
      (VOFA_PORT_TYPE == VOFA_PORT_USER_VALUE) || \
      (VOFA_PORT_TYPE == VOFA_PORT_ETHERCAT_VALUE)
    if (g_vofa.write_fn == NULL)
    {
        return (int32_t)VOFA_ERROR_INVALID_ARGUMENT;
    }
    return g_vofa.write_fn(g_vofa.user, data, length);
#else
#error "Unsupported VOFA_PORT_TYPE."
#endif
}

/** @brief 编译期 Port 读分派。没有读函数时表示当前没有输入。 */
static int32_t vofa_port_read(vofa_octet_t *data, uint32_t capacity)
{
    if (g_vofa.read_fn == NULL)
    {
        (void)data;
        (void)capacity;
        return 0;
    }

    return g_vofa.read_fn(g_vofa.user, data, capacity);
}

/** @brief 阻塞写一个 octet，底层忙时等待到超时。 */
static vofa_result_t vofa_write_one_blocking(vofa_octet_t octet)
{
    uint32_t timeout = VOFA_TX_TIMEOUT_COUNT;
    vofa_octet_t value = vofa_octet(octet);

    for (;;)
    {
        int32_t rc = vofa_port_write(&value, 1U);

        if (rc > 0)
        {
            vofa_stat_inc(&g_vofa.statistics.tx_octet_count);
            return VOFA_OK;
        }

        if (rc < 0)
        {
            return VOFA_ERROR_TRANSPORT;
        }

        if (timeout == 0U)
        {
            vofa_stat_inc(&g_vofa.statistics.tx_timeout_count);
            return VOFA_ERROR_TIMEOUT;
        }
        --timeout;
    }
}

/** @brief 命令入队。RX ISR 只做解析和入队，不执行应用动作。 */
static vofa_result_t vofa_queue_push(const vofa_command_t *command)
{
    uint16_t next;

    if (command == NULL)
    {
        return VOFA_ERROR_INVALID_ARGUMENT;
    }

    VOFA_ENTER_CRITICAL();
    next = (uint16_t)((g_vofa.queue_head + 1U) % VOFA_COMMAND_QUEUE_SIZE);
    if (next == g_vofa.queue_tail)
    {
        VOFA_EXIT_CRITICAL();
        vofa_stat_inc(&g_vofa.statistics.queue_overflow_count);
        return VOFA_ERROR_QUEUE_FULL;
    }

    g_vofa.queue[g_vofa.queue_head] = *command;
    g_vofa.queue_head = next;
    VOFA_EXIT_CRITICAL();
    return VOFA_OK;
}

/** @brief 命令出队，只允许 Vofa_Process 调用。 */
static bool vofa_queue_pop(vofa_command_t *command)
{
    if (command == NULL)
    {
        return false;
    }

    VOFA_ENTER_CRITICAL();
    if (g_vofa.queue_head == g_vofa.queue_tail)
    {
        VOFA_EXIT_CRITICAL();
        return false;
    }

    *command = g_vofa.queue[g_vofa.queue_tail];
    g_vofa.queue_tail = (uint16_t)((g_vofa.queue_tail + 1U) % VOFA_COMMAND_QUEUE_SIZE);
    VOFA_EXIT_CRITICAL();
    return true;
}

/** @brief 接收状态复位。旧协议和新协议互斥分流，不做双解析。 */
static void vofa_rx_reset(void)
{
    g_vofa.rx_state = VOFA_RX_IDLE;
    g_vofa.rx_index = 0U;
    g_vofa.rx_payload_length = 0U;
    g_vofa.rx_idle_ticks = 0U;
}

/** @brief 新协议完整帧校验并入队。 */
static void vofa_handle_new_frame(uint16_t length)
{
#if (VOFA_ENABLE_NEW_COMMAND != 0U)
    vofa_command_t command;
    uint16_t received_crc;
    uint16_t computed_crc;

    if (length < VOFA_NEW_FRAME_MIN_OCTETS)
    {
        return;
    }

    if (g_vofa.rx_frame[2] != VOFA_NEW_VERSION)
    {
        return;
    }

    received_crc = (uint16_t)(g_vofa.rx_frame[length - 2U] |
                              (g_vofa.rx_frame[length - 1U] << 8U));
    computed_crc = vofa_crc16(g_vofa.rx_frame, (uint16_t)(length - 2U));
    if (received_crc != computed_crc)
    {
        vofa_stat_inc(&g_vofa.statistics.crc_error_count);
        return;
    }

    command.command = (uint8_t)(g_vofa.rx_frame[3] & 0xFFU);
    command.sequence = (uint8_t)(g_vofa.rx_frame[4] & 0xFFU);
    command.object_id = (uint16_t)(g_vofa.rx_frame[5] | (g_vofa.rx_frame[6] << 8U));
    command.parameter_id = (uint16_t)(g_vofa.rx_frame[7] | (g_vofa.rx_frame[8] << 8U));
    command.data_type = (uint8_t)(g_vofa.rx_frame[9] & 0xFFU);
    command.length = (uint8_t)(g_vofa.rx_frame[10] & 0xFFU);
    command.payload[0] = (command.length > 0U) ? g_vofa.rx_frame[11] : 0U;
    command.payload[1] = (command.length > 1U) ? g_vofa.rx_frame[12] : 0U;
    command.payload[2] = (command.length > 2U) ? g_vofa.rx_frame[13] : 0U;
    command.payload[3] = (command.length > 3U) ? g_vofa.rx_frame[14] : 0U;
    command.from_new_protocol = true;

    (void)vofa_queue_push(&command);
    vofa_stat_inc(&g_vofa.statistics.rx_frame_count);
#else
    (void)length;
#endif
}

/** @brief 处理一个 RX octet。入口只选择一个协议状态机，禁止新旧双解析。 */
static void vofa_rx_feed_one(vofa_octet_t octet)
{
    vofa_octet_t value = vofa_octet(octet);

    (void)value;
    g_vofa.rx_idle_ticks = 0U;

    switch (g_vofa.rx_state)
    {
        case VOFA_RX_IDLE:
#if (VOFA_ENABLE_LEGACY_COMMAND != 0U)
            if (value == 0xAAU)
            {
                g_vofa.rx_state = VOFA_RX_LEGACY_FF;
                g_vofa.rx_frame[0] = value;
                g_vofa.rx_index = 1U;
            }
            else
#endif
#if (VOFA_ENABLE_NEW_COMMAND != 0U)
            if (value == VOFA_NEW_HEADER0)
            {
                g_vofa.rx_state = VOFA_RX_NEW_HEADER1;
                g_vofa.rx_frame[0] = value;
                g_vofa.rx_index = 1U;
            }
            else
#endif
            {
                /* 未识别帧头，丢弃。 */
            }
            break;

#if (VOFA_ENABLE_LEGACY_COMMAND != 0U)
        case VOFA_RX_LEGACY_FF:
            if (value == 0xFFU)
            {
                g_vofa.rx_frame[g_vofa.rx_index++] = value;
                g_vofa.rx_state = VOFA_RX_LEGACY_TYPE;
            }
            else
            {
                vofa_rx_reset();
            }
            break;

        case VOFA_RX_LEGACY_TYPE:
            g_vofa.rx_frame[g_vofa.rx_index++] = value;
            g_vofa.rx_state = VOFA_RX_LEGACY_ID;
            break;

        case VOFA_RX_LEGACY_ID:
            g_vofa.rx_frame[g_vofa.rx_index++] = value;
            g_vofa.rx_state = VOFA_RX_LEGACY_VALUE;
            break;

        case VOFA_RX_LEGACY_VALUE:
            g_vofa.rx_frame[g_vofa.rx_index++] = value;
            if (g_vofa.rx_index >= VOFA_LEGACY_FRAME_OCTETS)
            {
                vofa_command_t command;

                command.command = VOFA_CMD_WRITE_PARAMETER;
                command.sequence = 0U;
                command.object_id = (uint16_t)(g_vofa.rx_frame[2] & 0xFFU);
                command.parameter_id = (uint16_t)(g_vofa.rx_frame[3] & 0xFFU);
                command.data_type = VOFA_DATA_FLOAT32;
                command.length = VOFA_FLOAT_OCTETS;
                command.payload[0] = g_vofa.rx_frame[4];
                command.payload[1] = g_vofa.rx_frame[5];
                command.payload[2] = g_vofa.rx_frame[6];
                command.payload[3] = g_vofa.rx_frame[7];
                command.from_new_protocol = false;
                (void)vofa_queue_push(&command);
                vofa_stat_inc(&g_vofa.statistics.rx_frame_count);
                vofa_rx_reset();
            }
            break;
#endif

#if (VOFA_ENABLE_NEW_COMMAND != 0U)
        case VOFA_RX_NEW_HEADER1:
            if (value == VOFA_NEW_HEADER1)
            {
                g_vofa.rx_frame[g_vofa.rx_index++] = value;
                g_vofa.rx_state = VOFA_RX_NEW_FIXED;
            }
            else
            {
                vofa_rx_reset();
            }
            break;

        case VOFA_RX_NEW_FIXED:
            g_vofa.rx_frame[g_vofa.rx_index++] = value;
            if (g_vofa.rx_index == 11U)
            {
                if (g_vofa.rx_frame[2] != VOFA_NEW_VERSION)
                {
                    vofa_rx_reset();
                }
                else
                {
                    g_vofa.rx_payload_length = (uint16_t)(g_vofa.rx_frame[10] & 0xFFU);
                    if ((g_vofa.rx_payload_length > VOFA_FLOAT_OCTETS) ||
                        ((uint16_t)(VOFA_NEW_FRAME_MIN_OCTETS + g_vofa.rx_payload_length) > VOFA_RX_BUFFER_SIZE))
                    {
                        vofa_rx_reset();
                    }
                    else if (g_vofa.rx_payload_length == 0U)
                    {
                        g_vofa.rx_state = VOFA_RX_NEW_CRC0;
                    }
                    else
                    {
                        g_vofa.rx_state = VOFA_RX_NEW_PAYLOAD;
                    }
                }
            }
            break;

        case VOFA_RX_NEW_PAYLOAD:
            g_vofa.rx_frame[g_vofa.rx_index++] = value;
            if (g_vofa.rx_index == (uint16_t)(11U + g_vofa.rx_payload_length))
            {
                g_vofa.rx_state = VOFA_RX_NEW_CRC0;
            }
            break;

        case VOFA_RX_NEW_CRC0:
            g_vofa.rx_frame[g_vofa.rx_index++] = value;
            g_vofa.rx_state = VOFA_RX_NEW_CRC1;
            break;

        case VOFA_RX_NEW_CRC1:
            g_vofa.rx_frame[g_vofa.rx_index++] = value;
            vofa_handle_new_frame(g_vofa.rx_index);
            vofa_rx_reset();
            break;
#endif

        default:
            vofa_rx_reset();
            break;
    }
}

/** @brief 处理队列命令。所有参数写入、stream 控制和状态查询都在此执行。 */
static void vofa_execute_command(const vofa_command_t *command)
{
    vofa_result_t result = VOFA_ERROR_UNKNOWN_COMMAND;

    if (command == NULL)
    {
        return;
    }

    switch ((vofa_command_id_t)command->command)
    {
        case VOFA_CMD_WRITE_PARAMETER:
#if (VOFA_ENABLE_PID_TUNING != 0U)
            if ((command->data_type == VOFA_DATA_FLOAT32) && (command->length == VOFA_FLOAT_OCTETS))
            {
                float value = vofa_decode_float_le(command->payload);
                if (isfinite(value) == 0)
                {
                    result = VOFA_ERROR_RANGE;
                }
                else
                {
                    result = Vofa_AppWriteParameterPending(command->object_id, command->parameter_id, value);
                }
            }
            else
            {
                result = VOFA_ERROR_TYPE;
            }
#else
            result = VOFA_ERROR_DISABLED;
#endif
            break;

        case VOFA_CMD_READ_PARAMETER:
#if (VOFA_ENABLE_PID_TUNING != 0U)
        {
            float value = 0.0F;
            result = Vofa_AppReadParameter(command->object_id, command->parameter_id, &value);
            (void)value;
            break;
        }
#else
            result = VOFA_ERROR_DISABLED;
            break;
#endif

        case VOFA_CMD_SAVE_PARAMETERS:
            result = Vofa_AppSaveParameters();
            break;

        case VOFA_CMD_LOAD_DEFAULTS:
            result = Vofa_AppLoadDefaults();
            break;

        case VOFA_CMD_START_STREAM:
            Vofa_AppStartStream();
            result = VOFA_OK;
            break;

        case VOFA_CMD_STOP_STREAM:
            Vofa_AppStopStream();
            result = VOFA_OK;
            break;

        case VOFA_CMD_SET_SAMPLE_RATE:
            if ((command->data_type == VOFA_DATA_FLOAT32) && (command->length == VOFA_FLOAT_OCTETS))
            {
                float sample_rate = vofa_decode_float_le(command->payload);
                if ((isfinite(sample_rate) == 0) || (sample_rate <= 0.0F))
                {
                    result = VOFA_ERROR_RANGE;
                }
                else
                {
                    result = Vofa_AppSetSampleRate(sample_rate);
                }
            }
            else
            {
                result = VOFA_ERROR_TYPE;
            }
            break;

        case VOFA_CMD_GET_STATUS:
        {
            uint32_t status = Vofa_AppGetStatus();
            if (Vofa_IsTxBusy())
            {
                status |= 0x00000001UL;
            }
            (void)status;
            result = VOFA_OK;
            break;
        }

        default:
            result = VOFA_ERROR_UNKNOWN_COMMAND;
            break;
    }

    if (result == VOFA_ERROR_RANGE)
    {
        vofa_stat_inc(&g_vofa.statistics.parameter_range_error_count);
    }
}

void Vofa_Init(vofa_write_fn write_fn, vofa_read_fn read_fn, void *user)
{
    Vofa_Reset();
    g_vofa.write_fn = write_fn;
    g_vofa.read_fn = read_fn;
    g_vofa.user = user;
    g_vofa.initialized = true;
    g_vofa.memory_port.magic = 0x564F4641UL;
    g_vofa.memory_port.version = 1UL;
    Vofa_AppInit();
}

void Vofa_Process(void)
{
    vofa_octet_t rx[VOFA_RX_BUFFER_SIZE];
    int32_t read_count;
    vofa_command_t command;

    if (!g_vofa.initialized)
    {
        return;
    }

    read_count = vofa_port_read(rx, VOFA_RX_BUFFER_SIZE);
    if (read_count > 0)
    {
        Vofa_RxFeed(rx, (uint32_t)read_count);
    }

    if (g_vofa.rx_state != VOFA_RX_IDLE)
    {
        ++g_vofa.rx_idle_ticks;
        if (g_vofa.rx_idle_ticks > VOFA_RX_TIMEOUT_TICKS)
        {
            vofa_stat_inc(&g_vofa.statistics.rx_timeout_count);
            vofa_rx_reset();
        }
    }

    while (vofa_queue_pop(&command))
    {
        vofa_execute_command(&command);
    }

    Vofa_AppProcess();
}

vofa_result_t Vofa_SendChannels(const float *channels, uint16_t channel_count)
{
    return Vofa_SendChannelsEx(channels, channel_count, (vofa_tx_mode_t)VOFA_DEFAULT_TX_MODE);
}

vofa_result_t Vofa_SendChannelsEx(const float *channels, uint16_t channel_count, vofa_tx_mode_t mode)
{
#if (VOFA_ENABLE_JUSTFLOAT == 0U)
    (void)channels;
    (void)channel_count;
    (void)mode;
    return VOFA_ERROR_DISABLED;
#else
    uint32_t total_octets;
    uint32_t i;

    if ((channels == NULL) && (channel_count != 0U))
    {
        return VOFA_ERROR_INVALID_ARGUMENT;
    }

    total_octets = ((uint32_t)channel_count * VOFA_FLOAT_OCTETS) + 4U;

    if (mode == VOFA_TX_BLOCKING)
    {
        for (i = 0U; i < total_octets; ++i)
        {
            vofa_octet_t octet;
            vofa_result_t result = vofa_make_channel_octet(channels, channel_count, i, &octet);
            if (result != VOFA_OK)
            {
                return result;
            }
            result = vofa_write_one_blocking(octet);
            if (result != VOFA_OK)
            {
                return result;
            }
        }
        vofa_stat_inc(&g_vofa.statistics.tx_frame_count);
        return VOFA_OK;
    }

    if (mode == VOFA_TX_INTERRUPT)
    {
        if (g_vofa.tx_busy)
        {
            vofa_stat_inc(&g_vofa.statistics.tx_busy_count);
            return VOFA_ERROR_BUSY;
        }

        if (channel_count > VOFA_MAX_CHANNELS)
        {
            return VOFA_ERROR_RANGE;
        }

        VOFA_ENTER_CRITICAL();
        for (i = 0U; i < channel_count; ++i)
        {
            g_vofa.tx_channels[i] = channels[i];
        }
        g_vofa.tx_channel_count = channel_count;
        g_vofa.tx_octet_index = 0U;
        g_vofa.tx_total_octets = total_octets;
        g_vofa.tx_busy = true;
        VOFA_EXIT_CRITICAL();

        Vofa_TxEventHandler();
        return VOFA_OK;
    }

    return VOFA_ERROR_INVALID_ARGUMENT;
#endif
}

vofa_result_t Vofa_SendRaw(const vofa_octet_t *data, uint32_t length)
{
    uint32_t i;

    if ((data == NULL) && (length != 0U))
    {
        return VOFA_ERROR_INVALID_ARGUMENT;
    }

    for (i = 0U; i < length; ++i)
    {
        vofa_result_t result = vofa_write_one_blocking(data[i]);
        if (result != VOFA_OK)
        {
            return result;
        }
    }

    return VOFA_OK;
}

void Vofa_RxFeed(const vofa_octet_t *data, uint32_t length)
{
    uint32_t i;

    if ((data == NULL) && (length != 0U))
    {
        return;
    }

    for (i = 0U; i < length; ++i)
    {
        vofa_rx_feed_one(data[i]);
    }
}

void Vofa_TxEventHandler(void)
{
    vofa_octet_t octet;
    int32_t rc;

    if (!g_vofa.tx_busy)
    {
        return;
    }

    if (vofa_make_channel_octet(g_vofa.tx_channels, g_vofa.tx_channel_count, g_vofa.tx_octet_index, &octet) != VOFA_OK)
    {
        g_vofa.tx_busy = false;
        return;
    }

    rc = vofa_port_write(&octet, 1U);
    if (rc > 0)
    {
        ++g_vofa.tx_octet_index;
        vofa_stat_inc(&g_vofa.statistics.tx_octet_count);
        if (g_vofa.tx_octet_index >= g_vofa.tx_total_octets)
        {
            g_vofa.tx_busy = false;
            vofa_stat_inc(&g_vofa.statistics.tx_frame_count);
        }
    }
    else if (rc < 0)
    {
        g_vofa.tx_busy = false;
    }
    else
    {
        /* 底层仍忙，等待下一次 TX ready 事件再次调用。 */
    }
}

bool Vofa_IsTxBusy(void)
{
    return g_vofa.tx_busy;
}

void Vofa_Reset(void)
{
    uint16_t i;

    g_vofa.initialized = false;
    g_vofa.write_fn = NULL;
    g_vofa.read_fn = NULL;
    g_vofa.user = NULL;
    g_vofa.tx_busy = false;
    g_vofa.tx_channel_count = 0U;
    g_vofa.tx_octet_index = 0U;
    g_vofa.tx_total_octets = 0U;
    for (i = 0U; i < VOFA_MAX_CHANNELS; ++i)
    {
        g_vofa.tx_channels[i] = 0.0F;
    }
    vofa_rx_reset();
    g_vofa.queue_head = 0U;
    g_vofa.queue_tail = 0U;
    g_vofa.statistics.tx_frame_count = 0U;
    g_vofa.statistics.tx_octet_count = 0U;
    g_vofa.statistics.rx_frame_count = 0U;
    g_vofa.statistics.crc_error_count = 0U;
    g_vofa.statistics.rx_timeout_count = 0U;
    g_vofa.statistics.parameter_range_error_count = 0U;
    g_vofa.statistics.queue_overflow_count = 0U;
    g_vofa.statistics.tx_busy_count = 0U;
    g_vofa.statistics.tx_timeout_count = 0U;
    g_vofa.statistics.memory_overflow_count = 0U;
    g_vofa.memory_port.magic = 0x564F4641UL;
    g_vofa.memory_port.version = 1UL;
    g_vofa.memory_port.write_index = 0UL;
    g_vofa.memory_port.read_index = 0UL;
    g_vofa.memory_port.sequence = 0UL;
    g_vofa.memory_port.overflow_count = 0UL;
}

const vofa_statistics_t *Vofa_GetStatistics(void)
{
    return &g_vofa.statistics;
}

vofa_memory_port_t *Vofa_GetMemoryPort(void)
{
    return &g_vofa.memory_port;
}