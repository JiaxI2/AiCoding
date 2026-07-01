#ifndef VOFA_H
#define VOFA_H

/**
 * @file vofa.h
 * @brief VOFA+ 通用嵌入式通信模块。
 *
 * 最小使用方法：
 * 1. 实现一个发送函数：int32_t write(void *user, const vofa_octet_t *data, uint32_t len);
 * 2. 如果需要接收调参命令，再实现一个读取函数；不接收时传 NULL。
 * 3. 初始化：Vofa_Init(write, read, user);
 * 4. 发送波形：Vofa_SendChannels(channels, channel_count);
 * 5. 主循环或低优先级任务中周期调用：Vofa_Process();
 *
 * C28x 兼容说明：vofa_octet_t 用 uint16_t 承载线上 8 位 octet，只有低 8 位有效。
 * 大小端说明：CPU 可大端或小端，但 VOFA+ JustFloat 线上格式始终固定小端。
 *
 * 修改记录
 * 日期         作者        原因
 * 2026-06-23   HUJIAXUAN   简化初始化和配置，修复协议分流、命令队列和中断发送状态机。
 */

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* ========================================================================== */
/* 用户配置                                                       */
/* ========================================================================== */

#define VOFA_CPU_ENDIAN_LITTLE          (0U)
#define VOFA_CPU_ENDIAN_BIG             (1U)

#ifndef VOFA_CPU_ENDIAN
#define VOFA_CPU_ENDIAN                 VOFA_CPU_ENDIAN_LITTLE
#endif

#define VOFA_PORT_SCI_VALUE             (0U)
#define VOFA_PORT_ETHERCAT_VALUE        (1U)
#define VOFA_PORT_MEMORY_VALUE          (2U)
#define VOFA_PORT_USER_VALUE            (3U)

/* 通信介质类型。普通串口保持默认 SCI；Memory/SWD 调试可改成 VOFA_PORT_MEMORY_VALUE。 */
#ifndef VOFA_PORT_TYPE
#define VOFA_PORT_TYPE                  VOFA_PORT_SCI_VALUE
#endif

/* Vofa_SendChannels() 默认发送模式。最简单串口调试用阻塞；TX ISR 场景可改为中断。 */
#ifndef VOFA_DEFAULT_TX_MODE
#define VOFA_DEFAULT_TX_MODE            VOFA_TX_BLOCKING
#endif

/* 中断发送模式最多缓存多少个通道快照。阻塞发送不需要整帧缓存。 */
#ifndef VOFA_MAX_CHANNELS
#define VOFA_MAX_CHANNELS               (32U)
#endif

/* 接收命令队列深度。RX 中断只入队，Vofa_Process() 再执行命令。 */
#ifndef VOFA_COMMAND_QUEUE_SIZE
#define VOFA_COMMAND_QUEUE_SIZE         (8U)
#endif

/* 新协议接收缓冲大小。只影响控制命令，不影响 JustFloat 发送。 */
#ifndef VOFA_RX_BUFFER_SIZE
#define VOFA_RX_BUFFER_SIZE             (64U)
#endif

/* 接收超时计数。每次 Vofa_Process() 可认为时间前进 1 tick。 */
#ifndef VOFA_RX_TIMEOUT_TICKS
#define VOFA_RX_TIMEOUT_TICKS           (100U)
#endif

/* 阻塞发送等待底层写函数接受数据的轮询上限。 */
#ifndef VOFA_TX_TIMEOUT_COUNT
#define VOFA_TX_TIMEOUT_COUNT           (100000UL)
#endif

/* Memory/SWD Port 环形缓冲大小。只有 VOFA_PORT_TYPE 为 MEMORY 时使用。 */
#ifndef VOFA_MEMORY_BUFFER_SIZE
#define VOFA_MEMORY_BUFFER_SIZE         (512U)
#endif

/* 功能裁剪开关。关闭后对应代码尽量不参与编译。 */
#ifndef VOFA_ENABLE_JUSTFLOAT
#define VOFA_ENABLE_JUSTFLOAT           (1U)
#endif

#ifndef VOFA_ENABLE_FIREWATER
#define VOFA_ENABLE_FIREWATER           (0U)
#endif

#ifndef VOFA_ENABLE_PID_TUNING
#define VOFA_ENABLE_PID_TUNING          (1U)
#endif

#ifndef VOFA_ENABLE_LEGACY_COMMAND
#define VOFA_ENABLE_LEGACY_COMMAND      (1U)
#endif

#ifndef VOFA_ENABLE_NEW_COMMAND
#define VOFA_ENABLE_NEW_COMMAND         (1U)
#endif

#ifndef VOFA_ENABLE_STATISTICS
#define VOFA_ENABLE_STATISTICS          (1U)
#endif

/* C28x/ISR 并发保护钩子。需要保护共享队列或 TX 状态时，在工程配置中覆盖。 */
#ifndef VOFA_ENTER_CRITICAL
#define VOFA_ENTER_CRITICAL()           do { } while (0)
#endif

#ifndef VOFA_EXIT_CRITICAL
#define VOFA_EXIT_CRITICAL()            do { } while (0)
#endif

/* ========================================================================== */
/* 协议常量和合法性检查                                                       */
/* ========================================================================== */

#define VOFA_JUSTFLOAT_TAIL0            (0x00U)
#define VOFA_JUSTFLOAT_TAIL1            (0x00U)
#define VOFA_JUSTFLOAT_TAIL2            (0x80U)
#define VOFA_JUSTFLOAT_TAIL3            (0x7FU)
#define VOFA_FLOAT_OCTETS               (4U)
#define VOFA_LEGACY_FRAME_OCTETS        (8U)
#define VOFA_NEW_HEADER0                (0xA5U)
#define VOFA_NEW_HEADER1                (0x5AU)
#define VOFA_NEW_VERSION                (0x01U)
#define VOFA_NEW_FRAME_MIN_OCTETS       (13U)

#if (VOFA_CPU_ENDIAN != VOFA_CPU_ENDIAN_LITTLE) && (VOFA_CPU_ENDIAN != VOFA_CPU_ENDIAN_BIG)
#error "VOFA_CPU_ENDIAN must be VOFA_CPU_ENDIAN_LITTLE or VOFA_CPU_ENDIAN_BIG."
#endif

#if (VOFA_MAX_CHANNELS == 0U)
#error "VOFA_MAX_CHANNELS must be greater than zero."
#endif

#if (VOFA_COMMAND_QUEUE_SIZE == 0U)
#error "VOFA_COMMAND_QUEUE_SIZE must be greater than zero."
#endif

#if (VOFA_RX_BUFFER_SIZE < VOFA_NEW_FRAME_MIN_OCTETS)
#error "VOFA_RX_BUFFER_SIZE is too small."
#endif

#if (VOFA_MEMORY_BUFFER_SIZE == 0U)
#error "VOFA_MEMORY_BUFFER_SIZE must be greater than zero."
#endif

typedef uint16_t vofa_octet_t;

typedef int32_t (*vofa_write_fn)(void *user, const vofa_octet_t *data, uint32_t length);
typedef int32_t (*vofa_read_fn)(void *user, vofa_octet_t *data, uint32_t capacity);

typedef enum
{
    VOFA_OK = 0,
    VOFA_ERROR_INVALID_ARGUMENT = -1,
    VOFA_ERROR_BUSY = -2,
    VOFA_ERROR_TIMEOUT = -3,
    VOFA_ERROR_TRANSPORT = -4,
    VOFA_ERROR_BUFFER_TOO_SMALL = -5,
    VOFA_ERROR_CRC = -6,
    VOFA_ERROR_LENGTH = -7,
    VOFA_ERROR_VERSION = -8,
    VOFA_ERROR_TYPE = -9,
    VOFA_ERROR_UNKNOWN_COMMAND = -10,
    VOFA_ERROR_NOT_FOUND = -11,
    VOFA_ERROR_RANGE = -12,
    VOFA_ERROR_QUEUE_FULL = -13,
    VOFA_ERROR_DISABLED = -14
} vofa_result_t;

typedef enum
{
    VOFA_TX_BLOCKING = 0,
    VOFA_TX_INTERRUPT
} vofa_tx_mode_t;

typedef enum
{
    VOFA_CMD_READ_PARAMETER  = 0x01,
    VOFA_CMD_WRITE_PARAMETER = 0x02,
    VOFA_CMD_SAVE_PARAMETERS = 0x03,
    VOFA_CMD_LOAD_DEFAULTS   = 0x04,
    VOFA_CMD_START_STREAM    = 0x10,
    VOFA_CMD_STOP_STREAM     = 0x11,
    VOFA_CMD_SET_SAMPLE_RATE = 0x12,
    VOFA_CMD_GET_STATUS      = 0x20
} vofa_command_id_t;

typedef enum
{
    VOFA_DATA_NONE    = 0x00,
    VOFA_DATA_FLOAT32 = 0x01,
    VOFA_DATA_UINT32  = 0x02
} vofa_data_type_t;

typedef enum
{
    VOFA_WAVE_SINE = 0,
    VOFA_WAVE_SQUARE,
    VOFA_WAVE_TRIANGLE,
    VOFA_WAVE_SAWTOOTH,
    VOFA_WAVE_STEP,
    VOFA_WAVE_RAMP
} vofa_waveform_t;

typedef struct
{
    volatile uint32_t magic;
    volatile uint32_t version;
    volatile uint32_t write_index;
    volatile uint32_t read_index;
    volatile uint32_t sequence;
    volatile uint32_t overflow_count;
    vofa_octet_t buffer[VOFA_MEMORY_BUFFER_SIZE];
} vofa_memory_port_t;

typedef struct
{
    uint32_t tx_frame_count;
    uint32_t tx_octet_count;
    uint32_t rx_frame_count;
    uint32_t crc_error_count;
    uint32_t rx_timeout_count;
    uint32_t parameter_range_error_count;
    uint32_t queue_overflow_count;
    uint32_t tx_busy_count;
    uint32_t tx_timeout_count;
    uint32_t memory_overflow_count;
} vofa_statistics_t;

typedef struct
{
    uint16_t object_id;
    uint16_t parameter_id;
    float minimum;
    float maximum;
    uint16_t flags;
} vofa_parameter_descriptor_t;

#define VOFA_PARAMETER_FLAG_READABLE     (0x0001U)
#define VOFA_PARAMETER_FLAG_WRITABLE     (0x0002U)
#define VOFA_PARAMETER_FLAG_CLAMP        (0x0004U)

/**
 * @brief 初始化 VOFA 模块。
 * @param[in] write_fn 发送函数，最小发送必须提供。MEMORY Port 可传 NULL 使用内部环形缓冲。
 * @param[in] read_fn 读取函数，不接收命令时传 NULL。
 * @param[in] user 用户上下文，会传给 write_fn/read_fn。
 */
void Vofa_Init(vofa_write_fn write_fn, vofa_read_fn read_fn, void *user);

/** @brief 主循环调用：读取输入、处理队列命令、执行安全参数更新。 */
void Vofa_Process(void);

/** @brief 使用默认发送模式发送任意数量 JustFloat 通道。 */
vofa_result_t Vofa_SendChannels(const float *channels, uint16_t channel_count);

/** @brief 使用指定发送模式发送任意数量 JustFloat 通道。 */
vofa_result_t Vofa_SendChannelsEx(const float *channels, uint16_t channel_count, vofa_tx_mode_t mode);

/** @brief 发送原始 octet 数据，适合自定义桥接。 */
vofa_result_t Vofa_SendRaw(const vofa_octet_t *data, uint32_t length);

/** @brief 把 RX 中断、EtherCAT Mailbox 或用户队列收到的数据喂入状态机。 */
void Vofa_RxFeed(const vofa_octet_t *data, uint32_t length);

/** @brief TX ready 中断或事件中调用，推进中断发送。 */
void Vofa_TxEventHandler(void);

/** @brief 查询中断发送是否忙。 */
bool Vofa_IsTxBusy(void);

/** @brief 清空协议状态、发送状态、命令队列和统计。 */
void Vofa_Reset(void);

/** @brief 获取统计信息。 */
const vofa_statistics_t *Vofa_GetStatistics(void);

/** @brief 获取 Memory/SWD Port 缓冲区地址。只有 MEMORY Port 下用于调试器读取。 */
vofa_memory_port_t *Vofa_GetMemoryPort(void);

/* 应用层钩子：默认实现在 vofa_app.c，真实项目可替换。 */
void Vofa_AppInit(void);
void Vofa_AppProcess(void);
vofa_result_t Vofa_AppWriteParameterPending(uint16_t object_id, uint16_t parameter_id, float value);
vofa_result_t Vofa_AppReadParameter(uint16_t object_id, uint16_t parameter_id, float *value);
vofa_result_t Vofa_AppSaveParameters(void);
vofa_result_t Vofa_AppLoadDefaults(void);
vofa_result_t Vofa_AppSetSampleRate(float sample_rate_hz);
uint32_t Vofa_AppGetStatus(void);
void Vofa_AppStartStream(void);
void Vofa_AppStopStream(void);

float Vofa_AppWaveSine(float amplitude, float frequency, float sample_time);
float Vofa_AppWaveSquare(float amplitude, float frequency, float sample_time);
float Vofa_AppWaveTriangle(float amplitude, float frequency, float sample_time);
float Vofa_AppWaveSawtooth(float amplitude, float frequency, float sample_time);
float Vofa_AppWaveStep(float low, float high, uint32_t switch_after_samples);
float Vofa_AppWaveRamp(float start, float slope_per_sample, float minimum, float maximum);

#define VOFA_SEND_VALUES(...) \
    do { \
        const float vofa_values_[] = { __VA_ARGS__ }; \
        (void)Vofa_SendChannels(vofa_values_, (uint16_t)(sizeof(vofa_values_) / sizeof(vofa_values_[0]))); \
    } while (0)

#ifdef __cplusplus
}
#endif

#endif /* VOFA_H */