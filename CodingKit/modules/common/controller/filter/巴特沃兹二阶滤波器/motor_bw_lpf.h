#ifndef MOTOR_BW_LPF_H_
#define MOTOR_BW_LPF_H_

/*
 * motor_bw_lpf.h
 * Pure C 2nd-order Butterworth low-pass filter for embedded motor control.
 *
 * Intended use:
 *   - current sampling feedback with light filtering only;
 *   - debug/monitor/fault signals when stronger filtering is acceptable.
 *
 * Notes:
 *   - No dynamic allocation.
 *   - One filter object stores one signal channel.
 *   - Use one object per Ia/Ib/Ic or Id/Iq channel.
 *   - Coefficients are generated from sample frequency and cutoff frequency.
 */

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum
{
    MOTOR_BW_LPF_OK = 0,
    MOTOR_BW_LPF_ERR_NULL = -1,
    MOTOR_BW_LPF_ERR_PARAM = -2
} motor_bw_lpf_status_t;

typedef struct
{
    float b0;
    float b1;
    float b2;
    float a1;
    float a2;

    /* Transposed Direct Form II states. */
    float z1;
    float z2;

    float sample_hz;
    float cutoff_hz;
    uint8_t configured;
} motor_bw_lpf2_t;

/* Configure a 2nd-order Butterworth LPF.
 * sample_hz: control/update frequency, for example PWM ISR frequency.
 * cutoff_hz: -3 dB cutoff frequency.
 * Constraint: 0 < cutoff_hz < 0.45 * sample_hz.
 */
motor_bw_lpf_status_t MotorBwLpf2_Init(motor_bw_lpf2_t *f,
                                       float sample_hz,
                                       float cutoff_hz,
                                       float initial_value);

/* Recalculate coefficients while keeping the existing filter states. */
motor_bw_lpf_status_t MotorBwLpf2_Config(motor_bw_lpf2_t *f,
                                         float sample_hz,
                                         float cutoff_hz);

/* Change cutoff frequency while preserving sample frequency and states. */
motor_bw_lpf_status_t MotorBwLpf2_SetCutoff(motor_bw_lpf2_t *f,
                                            float cutoff_hz);

/* Reset filter states so output immediately starts from value. */
void MotorBwLpf2_Reset(motor_bw_lpf2_t *f, float value);

/* Process one sample. If the object is not configured, returns input directly. */
float MotorBwLpf2_Update(motor_bw_lpf2_t *f, float input);

/* Convenience helpers for dq or alpha/beta current pairs. */
void MotorBwLpf2_UpdatePair(motor_bw_lpf2_t *fa,
                            motor_bw_lpf2_t *fb,
                            float in_a,
                            float in_b,
                            float *out_a,
                            float *out_b);

/* Convenience helpers for Ia/Ib/Ic current triples. */
void MotorBwLpf2_Update3(motor_bw_lpf2_t *fa,
                         motor_bw_lpf2_t *fb,
                         motor_bw_lpf2_t *fc,
                         float in_a,
                         float in_b,
                         float in_c,
                         float *out_a,
                         float *out_b,
                         float *out_c);

#ifdef __cplusplus
}
#endif

#endif /* MOTOR_BW_LPF_H_ */
