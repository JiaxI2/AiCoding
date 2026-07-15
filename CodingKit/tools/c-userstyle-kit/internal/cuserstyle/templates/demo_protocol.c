/**
 * @file demo_protocol.c
 * @brief 实现不可信二进制帧、字符串和固定格式输出的安全处理。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @version 1.0.0
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：实现网络字节序解码、校验和、有界空字符查找、复制和安全格式化。
 * 主要功能：把所有外部输入视为不可信数据，在写输出前完成长度和内容验证。
 * 文件关系：实现 demo_protocol.h；不访问核心状态机和固定资源池的内部数据。
 */

#include "demo_protocol.h"

#include <inttypes.h>
#include <stdio.h>

/**
 * @brief 当前实现接受的线协议版本。
 *
 * @details
 * 取值范围固定为 1，与公开枚举 DEMO_PROTOCOL_VERSION 一致。该文件级数据初始化后只读，
 * 只供本实现的解码入口访问，不需要互斥，也不得由外部模块通过 extern 绕过接口读取。
 */
static const uint8_t s_protocol_version = (uint8_t)DEMO_PROTOCOL_VERSION;

static uint16_t DEMO_ReadU16BigEndian(const uint8_t *bytes);
static uint32_t DEMO_ReadU32BigEndian(const uint8_t *bytes);
static uint8_t DEMO_CalculateChecksum(const uint8_t *bytes, size_t byte_count);

/**
 * @brief 解析固定长度的网络字节序二进制帧。
 *
 * @details
 * 1. 校验输入和输出地址以及精确帧长，证明固定偏移访问不会越界。
 * 2. 按顺序检查协议版本和异或校验，返回最先发现的确定错误。
 * 3. 全部检查通过后转换网络字节序并一次发布消息字段，失败路径不写部分结果。
 *
 * @param[in] frame 可访问的二进制输入区，不允许为 NULL。
 * @param[in] frame_length 输入区字节数，必须等于 DEMO_PROTOCOL_FRAME_SIZE。
 * @param[out] message 接收已验证消息，不允许为 NULL。
 *
 * @return DEMO_PROTOCOL_RESULT_OK 解析成功；其他值说明失败原因。
 */
demo_protocol_result_t DEMO_DecodeFrame(const uint8_t *frame,
                                        size_t frame_length,
                                        demo_protocol_message_t *message)
{
    demo_protocol_result_t result = DEMO_PROTOCOL_RESULT_INVALID_ARGUMENT;

    /* 指针有效后才进入逐层帧校验，任何失败分支都保持输出消息不变。 */
    if ((frame != NULL) && (message != NULL))
    {
        if (frame_length != (size_t)DEMO_PROTOCOL_FRAME_SIZE)
        {
            /* 长度不符时固定偏移尚未得到证明，禁止读取版本、载荷或校验字节。 */
            result = DEMO_PROTOCOL_RESULT_INVALID_LENGTH;
        }
        else if (frame[0] != s_protocol_version)
        {
            /* 未知版本可能采用不同字段布局，不能按当前协议继续解释。 */
            result = DEMO_PROTOCOL_RESULT_INVALID_VERSION;
        }
        else if (DEMO_CalculateChecksum(frame, frame_length - 1U) != frame[7])
        {
            /* 校验失败表示传输内容不可信，不发布任何已经读取的部分字段。 */
            result = DEMO_PROTOCOL_RESULT_INVALID_CHECKSUM;
        }
        else
        {
            /* 验证完成后再转换并发布，两个字段来自同一个已验证输入快照。 */
            message->command = DEMO_ReadU16BigEndian(&frame[1]);
            message->value = DEMO_ReadU32BigEndian(&frame[3]);
            result = DEMO_PROTOCOL_RESULT_OK;
        }
    }

    return result;
}

/**
 * @brief 从有界源区域复制一个以空字符结束的字符串。
 *
 * @details
 * 在 source_capacity 范围内查找第一个空字符，不调用无界字符串函数。只有确认目标容量
 * 能容纳内容和结尾后才复制。该顺序同时避免源越界、目标越界和未结束字符串。
 *
 * @param[in] source 可访问的源区域，不允许为 NULL。
 * @param[in] source_capacity 源区域可安全读取的字节数。
 * @param[out] destination 目标字符数组，不允许为 NULL。
 * @param[in] destination_capacity 目标数组容量，必须包含结尾空字符空间。
 *
 * @return DEMO_PROTOCOL_RESULT_OK 复制成功；其他值表示参数、结尾或容量错误。
 */
demo_protocol_result_t DEMO_CopyText(const char *source,
                                     size_t source_capacity,
                                     char *destination,
                                     size_t destination_capacity)
{
    demo_protocol_result_t result = DEMO_PROTOCOL_RESULT_INVALID_ARGUMENT;

    /* 只有源、目标及其容量都有效时才查找结尾；否则保持目标不变。 */
    if ((source != NULL) && (destination != NULL) &&
        (source_capacity > 0U) && (destination_capacity > 0U))
    {
        size_t text_length = 0U;

        while ((text_length < source_capacity) && (source[text_length] != '\0'))
        {
            /* 每次读取前检查上界，避免先访问后判断造成差一错误。 */
            ++text_length;
        }

        if (text_length == source_capacity)
        {
            /* 可访问范围内没有空字符，输入不是合格的有界字符串。 */
            result = DEMO_PROTOCOL_RESULT_INVALID_LENGTH;
        }
        else if (text_length >= destination_capacity)
        {
            /* 等于容量也失败，因为仍需保留一个结尾空字符。 */
            result = DEMO_PROTOCOL_RESULT_OUTPUT_TOO_SMALL;
        }
        else
        {
            size_t index = 0U;

            for (index = 0U; index < text_length; ++index)
            {
                /* 源和目标索引都由前述双重容量检查约束。 */
                destination[index] = source[index];
            }
            destination[text_length] = '\0';
            result = DEMO_PROTOCOL_RESULT_OK;
        }
    }

    return result;
}

/**
 * @brief 使用固定格式串生成状态文本。
 *
 * @details
 * 调用 C99 snprintf 时使用编译期固定格式，并用 PRIu32 保证格式与 uint32_t 匹配。
 * 同时检查负返回值和截断条件，失败时主动写入空字符串，避免调用方误用截断内容。
 *
 * @param[in] sequence 待格式化的无符号序号。
 * @param[out] destination 接收文本的字符数组，不允许为 NULL。
 * @param[in] destination_capacity 目标数组容量，必须大于 0。
 *
 * @return DEMO_PROTOCOL_RESULT_OK 格式化成功；其他值表示参数或容量错误。
 */
demo_protocol_result_t DEMO_FormatStatus(uint32_t sequence,
                                         char *destination,
                                         size_t destination_capacity)
{
    demo_protocol_result_t result = DEMO_PROTOCOL_RESULT_INVALID_ARGUMENT;

    /* 无有效输出区域时不调用格式化函数，结果保持参数错误。 */
    if ((destination != NULL) && (destination_capacity > 0U))
    {
        const int written = snprintf(destination,
                                     destination_capacity,
                                     "sequence=%" PRIu32,
                                     sequence);

        if (written < 0)
        {
            /* 编码错误时清空首字符，禁止传播不完整文本。 */
            destination[0] = '\0';
            result = DEMO_PROTOCOL_RESULT_INVALID_ARGUMENT;
        }
        else if ((size_t)written >= destination_capacity)
        {
            /* snprintf 已保证结尾，但截断文本不属于成功输出。 */
            destination[0] = '\0';
            result = DEMO_PROTOCOL_RESULT_OUTPUT_TOO_SMALL;
        }
        else
        {
            /* 返回长度严格小于容量，文本完整且以空字符结束。 */
            result = DEMO_PROTOCOL_RESULT_OK;
        }
    }

    return result;
}

/**
 * @brief 按网络字节序读取一个 16 位无符号整数。
 *
 * @details
 * 每个字节先提升到 uint16_t，再按固定移位组合，避免有符号扩展和主机端字节序依赖。
 * 调用方已经保证至少可读取两个字节。
 *
 * @param[in] bytes 指向两个有效输入字节。
 *
 * @return 解码后的主机整数值。
 */
static uint16_t DEMO_ReadU16BigEndian(const uint8_t *bytes)
{
    const uint16_t high = (uint16_t)((uint32_t)bytes[0] << 8U);
    const uint16_t low = (uint16_t)bytes[1];

    return (uint16_t)(high | low);
}

/**
 * @brief 按网络字节序读取一个 32 位无符号整数。
 *
 * @details
 * 所有移位均在 uint32_t 上进行，规避窄整数提升和符号错误。调用方保证四字节可访问。
 *
 * @param[in] bytes 指向四个有效输入字节。
 *
 * @return 解码后的主机整数值。
 */
static uint32_t DEMO_ReadU32BigEndian(const uint8_t *bytes)
{
    const uint32_t byte_0 = (uint32_t)bytes[0] << 24U;
    const uint32_t byte_1 = (uint32_t)bytes[1] << 16U;
    const uint32_t byte_2 = (uint32_t)bytes[2] << 8U;
    const uint32_t byte_3 = (uint32_t)bytes[3];

    return byte_0 | byte_1 | byte_2 | byte_3;
}

/**
 * @brief 计算有界二进制区域的异或校验值。
 *
 * @details
 * 使用显式 byte_count，不调用 strlen 之类的字符串函数。循环每次处理一个字节且有明确上界。
 *
 * @param[in] bytes 有效二进制输入区。
 * @param[in] byte_count 要处理的字节数。
 *
 * @return 全部输入字节的异或值。
 */
static uint8_t DEMO_CalculateChecksum(const uint8_t *bytes, size_t byte_count)
{
    uint8_t checksum = 0U;
    size_t index = 0U;

    for (index = 0U; index < byte_count; ++index)
    {
        /* 显式转回 uint8_t，说明校验算法只保留最低 8 位。 */
        checksum = (uint8_t)(checksum ^ bytes[index]);
    }

    return checksum;
}
