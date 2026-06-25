#include "vofa.h"

#include <math.h>

#define VOFA_APP_TWO_PI_F             (6.28318530717958647692F)
#define VOFA_APP_OBJECT_VOLTAGE_PID   (0x0001U)
#define VOFA_APP_OBJECT_CURRENT_PID   (0x0002U)
#define VOFA_APP_OBJECT_CONTROL       (0x0003U)

/*
 * 修改记录
 * 日期         作者        原因
 * 2026-06-23   HUJIAXUAN   适配简化后的 Vofa_Init(write, read, user) 和 pending 参数流程。
 */

typedef struct
{
    vofa_parameter_descriptor_t descriptor;
    float *active_value;
    float pending_value;
    bool pending;
} vofa_app_parameter_t;

static bool g_vofaAppStreamEnabled;
static float g_vofaAppSampleRateHz;
static float g_voltageKp;
static float g_voltageKi;
static float g_voltageKd;
static float g_currentKp;
static float g_currentKi;
static float g_currentKd;
static float g_targetVoltage;
static float g_targetCurrent;
static float g_duty;
static float g_outputEnable;
static float g_debugSwitch;
static float g_waveEnable;

static vofa_app_parameter_t g_parameters[] =
{
    { { VOFA_APP_OBJECT_VOLTAGE_PID, 0x0001U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_voltageKp, 0.0F, false },
    { { VOFA_APP_OBJECT_VOLTAGE_PID, 0x0002U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_voltageKi, 0.0F, false },
    { { VOFA_APP_OBJECT_VOLTAGE_PID, 0x0003U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_voltageKd, 0.0F, false },
    { { VOFA_APP_OBJECT_CURRENT_PID, 0x0001U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_currentKp, 0.0F, false },
    { { VOFA_APP_OBJECT_CURRENT_PID, 0x0002U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_currentKi, 0.0F, false },
    { { VOFA_APP_OBJECT_CURRENT_PID, 0x0003U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_currentKd, 0.0F, false },
    { { VOFA_APP_OBJECT_CONTROL, 0x0001U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_targetVoltage, 0.0F, false },
    { { VOFA_APP_OBJECT_CONTROL, 0x0002U, 0.0F, 1000.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_targetCurrent, 0.0F, false },
    { { VOFA_APP_OBJECT_CONTROL, 0x0003U, 0.0F, 1.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_duty, 0.0F, false },
    { { VOFA_APP_OBJECT_CONTROL, 0x0004U, 0.0F, 1.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_outputEnable, 0.0F, false },
    { { VOFA_APP_OBJECT_CONTROL, 0x0005U, 0.0F, 1.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_debugSwitch, 0.0F, false },
    { { VOFA_APP_OBJECT_CONTROL, 0x0007U, 0.0F, 1.0F, VOFA_PARAMETER_FLAG_READABLE | VOFA_PARAMETER_FLAG_WRITABLE | VOFA_PARAMETER_FLAG_CLAMP }, &g_waveEnable, 0.0F, false },
};

/** @brief 查找应用参数描述项。 */
static vofa_app_parameter_t *vofa_app_find_parameter(uint16_t object_id, uint16_t parameter_id)
{
    size_t i;

    for (i = 0U; i < (sizeof(g_parameters) / sizeof(g_parameters[0])); ++i)
    {
        if ((g_parameters[i].descriptor.object_id == object_id) &&
            (g_parameters[i].descriptor.parameter_id == parameter_id))
        {
            return &g_parameters[i];
        }
    }

    return NULL;
}

/** @brief 按描述表限制参数范围。 */
static float vofa_app_limit_parameter(float value, const vofa_parameter_descriptor_t *descriptor)
{
    float out = value;

    if ((descriptor->flags & VOFA_PARAMETER_FLAG_CLAMP) != 0U)
    {
        if (out < descriptor->minimum)
        {
            out = descriptor->minimum;
        }
        else if (out > descriptor->maximum)
        {
            out = descriptor->maximum;
        }
        else
        {
            /* 已在范围内。 */
        }
    }

    return out;
}

void Vofa_AppInit(void)
{
    size_t i;

    g_vofaAppStreamEnabled = false;
    g_vofaAppSampleRateHz = 0.0F;
    for (i = 0U; i < (sizeof(g_parameters) / sizeof(g_parameters[0])); ++i)
    {
        g_parameters[i].pending = false;
        g_parameters[i].pending_value = 0.0F;
    }
}

void Vofa_AppProcess(void)
{
    size_t i;

    for (i = 0U; i < (sizeof(g_parameters) / sizeof(g_parameters[0])); ++i)
    {
        if (g_parameters[i].pending)
        {
            /* 在任务或控制周期边界应用，避免通信中断直接改控制参数。 */
            *(g_parameters[i].active_value) = g_parameters[i].pending_value;
            g_parameters[i].pending = false;
        }
    }
}

vofa_result_t Vofa_AppWriteParameterPending(uint16_t object_id, uint16_t parameter_id, float value)
{
    vofa_app_parameter_t *parameter = vofa_app_find_parameter(object_id, parameter_id);

    if (parameter == NULL)
    {
        return VOFA_ERROR_NOT_FOUND;
    }

    if ((parameter->descriptor.flags & VOFA_PARAMETER_FLAG_WRITABLE) == 0U)
    {
        return VOFA_ERROR_RANGE;
    }

    if (((parameter->descriptor.flags & VOFA_PARAMETER_FLAG_CLAMP) == 0U) &&
        ((value < parameter->descriptor.minimum) || (value > parameter->descriptor.maximum)))
    {
        return VOFA_ERROR_RANGE;
    }

    parameter->pending_value = vofa_app_limit_parameter(value, &parameter->descriptor);
    parameter->pending = true;
    return VOFA_OK;
}

vofa_result_t Vofa_AppReadParameter(uint16_t object_id, uint16_t parameter_id, float *value)
{
    vofa_app_parameter_t *parameter;

    if (value == NULL)
    {
        return VOFA_ERROR_INVALID_ARGUMENT;
    }

    parameter = vofa_app_find_parameter(object_id, parameter_id);
    if (parameter == NULL)
    {
        return VOFA_ERROR_NOT_FOUND;
    }

    if ((parameter->descriptor.flags & VOFA_PARAMETER_FLAG_READABLE) == 0U)
    {
        return VOFA_ERROR_RANGE;
    }

    *value = *(parameter->active_value);
    return VOFA_OK;
}

vofa_result_t Vofa_AppSaveParameters(void)
{
    /* Flash/NVM 保存必须由产品工程接入，不能在滑块拖动过程中自动频繁擦写。 */
    return VOFA_OK;
}

vofa_result_t Vofa_AppLoadDefaults(void)
{
    g_voltageKp = 0.0F;
    g_voltageKi = 0.0F;
    g_voltageKd = 0.0F;
    g_currentKp = 0.0F;
    g_currentKi = 0.0F;
    g_currentKd = 0.0F;
    g_targetVoltage = 0.0F;
    g_targetCurrent = 0.0F;
    g_duty = 0.0F;
    g_outputEnable = 0.0F;
    g_debugSwitch = 0.0F;
    g_waveEnable = 0.0F;
    return VOFA_OK;
}

vofa_result_t Vofa_AppSetSampleRate(float sample_rate_hz)
{
    if (sample_rate_hz <= 0.0F)
    {
        return VOFA_ERROR_RANGE;
    }

    g_vofaAppSampleRateHz = sample_rate_hz;
    return VOFA_OK;
}

uint32_t Vofa_AppGetStatus(void)
{
    uint32_t status = 0UL;

    if (g_vofaAppStreamEnabled)
    {
        status |= 0x00000002UL;
    }
    if (g_vofaAppSampleRateHz > 0.0F)
    {
        status |= 0x00000004UL;
    }

    return status;
}

void Vofa_AppStartStream(void)
{
    g_vofaAppStreamEnabled = true;
}

void Vofa_AppStopStream(void)
{
    g_vofaAppStreamEnabled = false;
}

/** @brief 推进归一化相位。 */
static void vofa_app_advance_phase(float *phase, float frequency, float sample_time)
{
    if (phase == NULL)
    {
        return;
    }

    *phase += frequency * sample_time;
    *phase -= floorf(*phase);
    if (*phase < 0.0F)
    {
        *phase += 1.0F;
    }
}

float Vofa_AppWaveSine(float amplitude, float frequency, float sample_time)
{
    static float phase;

    vofa_app_advance_phase(&phase, frequency, sample_time);
    return amplitude * sinf(VOFA_APP_TWO_PI_F * phase);
}

float Vofa_AppWaveSquare(float amplitude, float frequency, float sample_time)
{
    static float phase;

    vofa_app_advance_phase(&phase, frequency, sample_time);
    return (phase < 0.5F) ? amplitude : -amplitude;
}

float Vofa_AppWaveTriangle(float amplitude, float frequency, float sample_time)
{
    static float phase;
    float normalized;

    vofa_app_advance_phase(&phase, frequency, sample_time);
    if (phase < 0.25F)
    {
        normalized = 4.0F * phase;
    }
    else if (phase < 0.75F)
    {
        normalized = 2.0F - (4.0F * phase);
    }
    else
    {
        normalized = (4.0F * phase) - 4.0F;
    }

    return amplitude * normalized;
}

float Vofa_AppWaveSawtooth(float amplitude, float frequency, float sample_time)
{
    static float phase;

    vofa_app_advance_phase(&phase, frequency, sample_time);
    return amplitude * ((2.0F * phase) - 1.0F);
}

float Vofa_AppWaveStep(float low, float high, uint32_t switch_after_samples)
{
    static uint32_t sample_count;

    if (sample_count < switch_after_samples)
    {
        ++sample_count;
        return low;
    }

    return high;
}

float Vofa_AppWaveRamp(float start, float slope_per_sample, float minimum, float maximum)
{
    static bool initialized;
    static float value;

    if (!initialized)
    {
        value = start;
        initialized = true;
    }
    else
    {
        value += slope_per_sample;
    }

    if (value > maximum)
    {
        value = minimum;
    }
    else if (value < minimum)
    {
        value = maximum;
    }
    else
    {
        /* 保持当前值。 */
    }

    return value;
}