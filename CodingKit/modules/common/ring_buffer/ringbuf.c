#include "ringbuf.h"

#ifdef __TI_COMPILER_VERSION__
#pragma CODE_SECTION(RingBuf_Init, ".TI.ramfunc");
#pragma CODE_SECTION(RingBuf_Reset, ".TI.ramfunc");
#pragma CODE_SECTION(RingBuf_Used, ".TI.ramfunc");
#pragma CODE_SECTION(RingBuf_Free, ".TI.ramfunc");
#pragma CODE_SECTION(RingBuf_Write, ".TI.ramfunc");
#pragma CODE_SECTION(RingBuf_Read, ".TI.ramfunc");
#pragma CODE_SECTION(RingBuf_ReadByte, ".TI.ramfunc");
#endif

static void RingBuf_StoreByte(ringbuf_t *rb, uint16_t index, uint8_t data)
{
#if defined(__TMS320C2000__)
    uint16_t wordIndex = (uint16_t)(index >> 1U);
    uint16_t word = rb->buffer[wordIndex];

    if((index & 1U) == 0U)
    {
        word = (word & 0xFF00U) | ((uint16_t)data & 0x00FFU);
    }
    else
    {
        word = (word & 0x00FFU) | (((uint16_t)data & 0x00FFU) << 8U);
    }
    rb->buffer[wordIndex] = (uint8_t)word;
#else
    rb->buffer[index] = data;
#endif
}

static uint8_t RingBuf_LoadByte(const ringbuf_t *rb, uint16_t index)
{
#if defined(__TMS320C2000__)
    uint16_t word = rb->buffer[index >> 1U];

    return (uint8_t)(((index & 1U) == 0U) ?
        (word & 0x00FFU) : ((word >> 8U) & 0x00FFU));
#else
    return rb->buffer[index];
#endif
}
static uint16_t RingBuf_NextIndex(uint16_t index, uint16_t capacity)
{
    index++;
    if(index >= capacity)
    {
        index = 0U;
    }

    return index;
}

void RingBuf_Init(ringbuf_t *rb, uint8_t *buffer, uint16_t capacity)
{
    if(rb == (ringbuf_t *)0)
    {
        return;
    }

    rb->buffer = buffer;
    rb->capacity = (buffer != (uint8_t *)0) ? capacity : 0U;
    rb->head = 0U;
    rb->tail = 0U;
    rb->used = 0U;
}

void RingBuf_Reset(ringbuf_t *rb)
{
    if(rb == (ringbuf_t *)0)
    {
        return;
    }

    rb->head = 0U;
    rb->tail = 0U;
    rb->used = 0U;
}

uint16_t RingBuf_Used(const ringbuf_t *rb)
{
    if(rb == (const ringbuf_t *)0)
    {
        return 0U;
    }

    return rb->used;
}

uint16_t RingBuf_Free(const ringbuf_t *rb)
{
    if((rb == (const ringbuf_t *)0) || (rb->capacity < rb->used))
    {
        return 0U;
    }

    return (uint16_t)(rb->capacity - rb->used);
}

bool RingBuf_Write(ringbuf_t *rb, const uint8_t *data, uint16_t length)
{
    uint16_t index;

    if(length == 0U)
    {
        return true;
    }
    if((rb == (ringbuf_t *)0) ||
       (rb->buffer == (uint8_t *)0) ||
       (data == (const uint8_t *)0) ||
       (length > RingBuf_Free(rb)))
    {
        return false;
    }

    for(index = 0U; index < length; index++)
    {
        RingBuf_StoreByte(rb, rb->tail, data[index]);
        rb->tail = RingBuf_NextIndex(rb->tail, rb->capacity);
    }
    rb->used = (uint16_t)(rb->used + length);

    return true;
}

bool RingBuf_Read(ringbuf_t *rb, uint8_t *data, uint16_t length)
{
    uint16_t index;

    if(length == 0U)
    {
        return true;
    }
    if((rb == (ringbuf_t *)0) ||
       (rb->buffer == (uint8_t *)0) ||
       (data == (uint8_t *)0) ||
       (length > RingBuf_Used(rb)))
    {
        return false;
    }

    for(index = 0U; index < length; index++)
    {
        data[index] = RingBuf_LoadByte(rb, rb->head);
        rb->head = RingBuf_NextIndex(rb->head, rb->capacity);
    }
    rb->used = (uint16_t)(rb->used - length);

    return true;
}

bool RingBuf_ReadByte(ringbuf_t *rb, uint8_t *data)
{
    return RingBuf_Read(rb, data, 1U);
}
