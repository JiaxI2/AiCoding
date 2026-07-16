/**
 * @file demo_protocol.h
 * @brief 不可信帧、字符串和格式化输出的安全处理接口。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：声明固定网络字节序帧、受界字符串复制和固定格式输出接口。
 * 主要功能：展示通信字节序、二进制长度、空字符结尾、整数转换和格式串安全规则。
 * 文件关系：由 demo_protocol.c 实现；可独立于 demo.h 和 demo_pool.h 包含及编译。
 */

#ifndef DEMO_PROTOCOL_H
#define DEMO_PROTOCOL_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 演示协议固定字段值。
 */
enum
{
    DEMO_PROTOCOL_VERSION = 1,
    DEMO_PROTOCOL_FRAME_SIZE = 8
};

/**
 * @brief 协议解析结果。
 */
typedef enum
{
    DEMO_PROTOCOL_RESULT_OK = 0,
    DEMO_PROTOCOL_RESULT_INVALID_ARGUMENT,
    DEMO_PROTOCOL_RESULT_INVALID_LENGTH,
    DEMO_PROTOCOL_RESULT_INVALID_VERSION,
    DEMO_PROTOCOL_RESULT_INVALID_CHECKSUM,
    DEMO_PROTOCOL_RESULT_OUTPUT_TOO_SMALL
} demo_protocol_result_t;

/**
 * @brief 从固定网络字节序帧解析出的消息。
 */
typedef struct
{
    uint16_t command;  /**< 两字节无符号命令字。 */
    uint32_t value;    /**< 四字节无符号载荷。 */
} demo_protocol_message_t;

/**
 * @brief 解析固定长度的网络字节序二进制帧。
 *
 * @param[in] frame 可访问的二进制输入区，不允许为 NULL。
 * @param[in] frame_length 输入区字节数，必须等于 DEMO_PROTOCOL_FRAME_SIZE。
 * @param[out] message 接收已验证消息，不允许为 NULL。
 *
 * @return DEMO_PROTOCOL_RESULT_OK 解析成功；其他值说明失败原因。
 *
 * @note 性能为固定 8 字节扫描；函数不把二进制数据当字符串处理，可重入。
 */
demo_protocol_result_t DEMO_DecodeFrame(const uint8_t *frame,
                                        size_t frame_length,
                                        demo_protocol_message_t *message);

/**
 * @brief 从有界源区域复制一个以空字符结束的字符串。
 *
 * @param[in] source 可访问的源区域，不允许为 NULL。
 * @param[in] source_capacity 源区域可安全读取的字节数。
 * @param[out] destination 目标字符数组，不允许为 NULL。
 * @param[in] destination_capacity 目标数组容量，必须包含结尾空字符空间。
 *
 * @return DEMO_PROTOCOL_RESULT_OK 复制成功；其他值表示参数、结尾或容量错误。
 *
 * @note 性能与 source_capacity 线性相关且有明确上界；函数可重入，源和目标区域不得重叠。
 */
demo_protocol_result_t DEMO_CopyText(const char *source,
                                     size_t source_capacity,
                                     char *destination,
                                     size_t destination_capacity);

/**
 * @brief 使用固定格式串生成状态文本。
 *
 * @param[in] sequence 待格式化的无符号序号。
 * @param[out] destination 接收文本的字符数组，不允许为 NULL。
 * @param[in] destination_capacity 目标数组容量，必须大于 0。
 *
 * @return DEMO_PROTOCOL_RESULT_OK 格式化成功；其他值表示参数或容量错误。
 *
 * @note 性能受目标容量严格约束；格式串为编译期常量，用户数据不作格式说明，函数可重入。
 */
demo_protocol_result_t DEMO_FormatStatus(uint32_t sequence,
                                         char *destination,
                                         size_t destination_capacity);

#ifdef __cplusplus
}
#endif

#endif /* DEMO_PROTOCOL_H */
