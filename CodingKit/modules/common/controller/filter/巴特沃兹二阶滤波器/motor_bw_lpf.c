#include "motor_bw_lpf.h"

#include <float.h>
#include <math.h>

#ifndef MOTOR_BW_LPF_PI_F
#define MOTOR_BW_LPF_PI_F        (3.14159265358979323846f)
#endif

#ifndef MOTOR_BW_LPF_SQRT2_F
#define MOTOR_BW_LPF_SQRT2_F     (1.41421356237309504880f)
#endif

#ifndef MOTOR_BW_LPF_MAX_FC_RATIO
/* Keep away from Nyquist to avoid tan(pi * fc / fs) numerical blow-up. */
#define MOTOR_BW_LPF_MAX_FC_RATIO (0.45f)
#endif

#ifndef MOTOR_BW_LPF_TANF
#define MOTOR_BW_LPF_TANF(x)     tanf((x))
#endif

#ifdef __TI_COMPILER_VERSION__
#pragma CODE_SECTION(MotorBwLpf2_Init, ".TI.ramfunc");
#pragma CODE_SECTION(MotorBwLpf2_Config, ".TI.ramfunc");
#pragma CODE_SECTION(MotorBwLpf2_SetCutoff, ".TI.ramfunc");
#pragma CODE_SECTION(MotorBwLpf2_Reset, ".TI.ramfunc");
#pragma CODE_SECTION(MotorBwLpf2_Update, ".TI.ramfunc");
#pragma CODE_SECTION(MotorBwLpf2_UpdatePair, ".TI.ramfunc");
#pragma CODE_SECTION(MotorBwLpf2_Update3, ".TI.ramfunc");
#endif

static int MotorBwLpf2_IsFinite(float x)
{
    return ((x == x) && (x <= FLT_MAX) && (x >= -FLT_MAX));
}

static int MotorBwLpf2_IsValidConfig(float sample_hz, float cutoff_hz)
{
    if((!MotorBwLpf2_IsFinite(sample_hz)) || (!MotorBwLpf2_IsFinite(cutoff_hz)))
    {
        return 0;
    }

    if((sample_hz <= 0.0f) || (cutoff_hz <= 0.0f))
    {
        return 0;
    }

    if(cutoff_hz >= (MOTOR_BW_LPF_MAX_FC_RATIO * sample_hz))
    {
        return 0;
    }

    return 1;
}

motor_bw_lpf_status_t MotorBwLpf2_Config(motor_bw_lpf2_t *f,
                                         float sample_hz,
                                         float cutoff_hz)
{
    float k;
    float k2;
    float norm;

    if(f == (motor_bw_lpf2_t *)0)
    {
        return MOTOR_BW_LPF_ERR_NULL;
    }

    if(!MotorBwLpf2_IsValidConfig(sample_hz, cutoff_hz))
    {
        f->configured = 0U;
        return MOTOR_BW_LPF_ERR_PARAM;
    }

    /*
     * Bilinear-transform design of analog Butterworth prototype:
     * H(s) = wc^2 / (s^2 + sqrt(2) * wc * s + wc^2)
     * with frequency prewarping via k = tan(pi * fc / fs).
     * Denominator form used by Update(): 1 + a1*z^-1 + a2*z^-2.
     */
    k = MOTOR_BW_LPF_TANF((MOTOR_BW_LPF_PI_F * cutoff_hz) / sample_hz);
    if(!MotorBwLpf2_IsFinite(k) || (k <= 0.0f))
    {
        f->configured = 0U;
        return MOTOR_BW_LPF_ERR_PARAM;
    }

    k2 = k * k;
    norm = 1.0f / (1.0f + (MOTOR_BW_LPF_SQRT2_F * k) + k2);

    f->b0 = k2 * norm;
    f->b1 = 2.0f * f->b0;
    f->b2 = f->b0;
    f->a1 = 2.0f * (k2 - 1.0f) * norm;
    f->a2 = (1.0f - (MOTOR_BW_LPF_SQRT2_F * k) + k2) * norm;

    f->sample_hz = sample_hz;
    f->cutoff_hz = cutoff_hz;
    f->configured = 1U;

    return MOTOR_BW_LPF_OK;
}

motor_bw_lpf_status_t MotorBwLpf2_Init(motor_bw_lpf2_t *f,
                                       float sample_hz,
                                       float cutoff_hz,
                                       float initial_value)
{
    motor_bw_lpf_status_t status;

    if(f == (motor_bw_lpf2_t *)0)
    {
        return MOTOR_BW_LPF_ERR_NULL;
    }

    f->b0 = 1.0f;
    f->b1 = 0.0f;
    f->b2 = 0.0f;
    f->a1 = 0.0f;
    f->a2 = 0.0f;
    f->z1 = 0.0f;
    f->z2 = 0.0f;
    f->sample_hz = 0.0f;
    f->cutoff_hz = 0.0f;
    f->configured = 0U;

    status = MotorBwLpf2_Config(f, sample_hz, cutoff_hz);
    MotorBwLpf2_Reset(f, initial_value);

    return status;
}

motor_bw_lpf_status_t MotorBwLpf2_SetCutoff(motor_bw_lpf2_t *f,
                                            float cutoff_hz)
{
    if(f == (motor_bw_lpf2_t *)0)
    {
        return MOTOR_BW_LPF_ERR_NULL;
    }

    return MotorBwLpf2_Config(f, f->sample_hz, cutoff_hz);
}

void MotorBwLpf2_Reset(motor_bw_lpf2_t *f, float value)
{
    if(f == (motor_bw_lpf2_t *)0)
    {
        return;
    }

    if((f->configured != 0U) && MotorBwLpf2_IsFinite(value))
    {
        /* Make y=value for constant input=value immediately after reset. */
        f->z1 = value * (1.0f - f->b0);
        f->z2 = value * (f->b2 - f->a2);
    }
    else
    {
        f->z1 = 0.0f;
        f->z2 = 0.0f;
    }
}

float MotorBwLpf2_Update(motor_bw_lpf2_t *f, float input)
{
    float y;
    float z1_new;
    float z2_new;

    if((f == (motor_bw_lpf2_t *)0) || (f->configured == 0U))
    {
        return input;
    }

    y = (f->b0 * input) + f->z1;
    z1_new = (f->b1 * input) + f->z2 - (f->a1 * y);
    z2_new = (f->b2 * input) - (f->a2 * y);

    f->z1 = z1_new;
    f->z2 = z2_new;

    return y;
}

void MotorBwLpf2_UpdatePair(motor_bw_lpf2_t *fa,
                            motor_bw_lpf2_t *fb,
                            float in_a,
                            float in_b,
                            float *out_a,
                            float *out_b)
{
    float ya;
    float yb;

    ya = MotorBwLpf2_Update(fa, in_a);
    yb = MotorBwLpf2_Update(fb, in_b);

    if(out_a != (float *)0)
    {
        *out_a = ya;
    }

    if(out_b != (float *)0)
    {
        *out_b = yb;
    }
}

void MotorBwLpf2_Update3(motor_bw_lpf2_t *fa,
                         motor_bw_lpf2_t *fb,
                         motor_bw_lpf2_t *fc,
                         float in_a,
                         float in_b,
                         float in_c,
                         float *out_a,
                         float *out_b,
                         float *out_c)
{
    float ya;
    float yb;
    float yc;

    ya = MotorBwLpf2_Update(fa, in_a);
    yb = MotorBwLpf2_Update(fb, in_b);
    yc = MotorBwLpf2_Update(fc, in_c);

    if(out_a != (float *)0)
    {
        *out_a = ya;
    }

    if(out_b != (float *)0)
    {
        *out_b = yb;
    }

    if(out_c != (float *)0)
    {
        *out_c = yc;
    }
}
