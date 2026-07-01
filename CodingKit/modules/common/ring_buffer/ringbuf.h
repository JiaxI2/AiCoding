#ifndef RINGBUF_H
#define RINGBUF_H

#include <stdbool.h>
#include <stdint.h>
/*
 * TI C28x 的最小寻址单元是 16 bit，C2000 stdint.h 不提供精确 8-bit
 * uint8_t。这里仅为 ringbuf 的“字节流接口”提供本地兼容类型，实际存储
 * 单元仍是 C28x 的 16-bit char/uint16_t，与 driverlib 的 hw_types.h 一致。
 */
#if defined(__TMS320C2000__) && !defined(HW_TYPES_H)
typedef uint16_t uint8_t;
#endif

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 简单字节环形缓冲区实例。
 *
 * buffer 由调用者提供，模块不做动态内存分配；C28x 下 buffer 可指向 uint16_t 存储区并由模块按 2 字节打包；capacity、head、tail、used
 * 均按“可存储的字节数”计数。第一版不提供覆盖旧数据、peek 或线程锁，
 * 上层需要在 ISR/双核共享场景中自行处理临界区。
 */
typedef struct
{
    uint8_t *buffer;
    uint16_t capacity;
    uint16_t head;
    uint16_t tail;
    uint16_t used;
} ringbuf_t;

/**
 * @brief 初始化环形缓冲区。
 * @param rb 环形缓冲区实例。
 * @param buffer 外部提供的存储区。
 * @param capacity 存储区容量，单位为字节。
 */
void RingBuf_Init(ringbuf_t *rb, uint8_t *buffer, uint16_t capacity);

/**
 * @brief 清空环形缓冲区，保留原 buffer 和 capacity。
 * @param rb 环形缓冲区实例。
 */
void RingBuf_Reset(ringbuf_t *rb);

/**
 * @brief 获取当前已缓存字节数。
 * @param rb 环形缓冲区实例。
 * @return 已缓存字节数；参数无效时返回 0。
 */
uint16_t RingBuf_Used(const ringbuf_t *rb);

/**
 * @brief 获取当前剩余可写字节数。
 * @param rb 环形缓冲区实例。
 * @return 剩余字节数；参数无效时返回 0。
 */
uint16_t RingBuf_Free(const ringbuf_t *rb);

/**
 * @brief 连续写入一段数据。
 * @param rb 环形缓冲区实例。
 * @param data 待写入数据。
 * @param length 写入长度，单位为字节。
 * @return true 写入成功；false 参数无效或剩余空间不足，缓冲区状态不变。
 */
bool RingBuf_Write(ringbuf_t *rb, const uint8_t *data, uint16_t length);

/**
 * @brief 连续读取一段数据。
 * @param rb 环形缓冲区实例。
 * @param data 读取输出缓冲区。
 * @param length 读取长度，单位为字节。
 * @return true 读取成功；false 参数无效或数据不足，缓冲区状态不变。
 */
bool RingBuf_Read(ringbuf_t *rb, uint8_t *data, uint16_t length);

/**
 * @brief 读取 1 个字节。
 * @param rb 环形缓冲区实例。
 * @param data 读取输出地址。
 * @return true 读取成功；false 参数无效或缓冲区为空。
 */
bool RingBuf_ReadByte(ringbuf_t *rb, uint8_t *data);

#ifdef __cplusplus
}
#endif

#endif /* RINGBUF_H */
