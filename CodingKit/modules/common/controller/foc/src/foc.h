#ifndef FOC_H
#define FOC_H

/**
 * @file foc.h
 * @brief Flat VF/IF FOC controller template.
 * @author HU JIAXUAN
 *
 * This module owns only the reusable FOC math chain:
 * phase current correction -> Clarke -> Park -> VF/IF voltage generation ->
 * inverse Park -> SVPWM duty generation.
 *
 * It does not bind ADC, PWM, encoder, Hall, observer, state-machine, fault,
 * communication, or hardware driver objects.
 */

#include <stdbool.h>
#include <stdint.h>

#include "foc_math.h"
#include "foc_svpwm.h"
#include "pid.h"

#ifdef __cplusplus
extern "C" {
#endif

#ifndef FOC_DEFAULT_CONTROL_FREQ_HZ
#define FOC_DEFAULT_CONTROL_FREQ_HZ (10000.0f)
#endif

typedef enum {
    FOC_MODE_VF = 0,
    FOC_MODE_IF = 1
} FocMode;

typedef enum {
    FOC_ANGLE_SENSOR = 0,
    FOC_ANGLE_OPEN_LOOP = 1
} FocAngleMode;

typedef enum {
    FOC_CONTROL_MODE_OPEN_VOLTAGE = 0,
    FOC_CONTROL_MODE_CLOSED_CURRENT = 1,
    FOC_CONTROL_MODE_MOTION_CURRENT = 2
} FocControlMode;

typedef struct {
    FocMode mode;
    FocAngleMode angle_mode;

    float control_freq;

    float vbus;
    float ia;
    float ib;
    float ic;

    float theta_e;
    float omega_e;
    float open_loop_freq_hz;
    float dir;

    float cmd_pos;
    float pos;
    float cmd_vel;
    float vel;

    float cmd_id;
    float cmd_iq;

    float cmd_vd;
    float cmd_vq;

    float real_id;
    float real_iq;
    float real_ialpha;
    float real_ibeta;

    float out_vd;
    float out_vq;
    float out_valpha;
    float out_vbeta;

    float duty_a;
    float duty_b;
    float duty_c;

    float max_voltage;
    float modulation_limit;

    float vf_gain_v_per_hz;
    float vf_boost_v;
    float vf_min_v;
    float vf_max_v;

    bool enable_pos_loop;
    bool enable_vel_loop;
    bool enable_id_loop;
    bool enable_iq_loop;

    Pid pid_pos;
    Pid pid_vel;
    Pid pid_id;
    Pid pid_iq;

    float ia_offset;
    float ib_offset;
    float ic_offset;
    uint32_t offset_sample_count;
    bool current_offset_valid;

    bool saturated;
    bool valid;
} Foc;

/**
 * @brief Initialize a flat FOC controller.
 * @param[out] controller FOC controller object.
 * @return true when initialized; false when controller is NULL.
 */
bool foc_init(Foc *controller);

/**
 * @brief Clear the accumulated phase current offset.
 * @param[in,out] controller FOC controller object.
 * @return true when cleared; false when controller is NULL.
 */
bool foc_current_offset_clear(Foc *controller);

/**
 * @brief Accumulate one phase current sample into the running offset estimate.
 * @param[in,out] controller FOC controller object.
 * @param[in] sample Phase current sample in amperes.
 * @return true when accumulated; false when controller is NULL.
 */
bool foc_current_offset_accumulate(Foc *controller, FocPhase sample);

/**
 * @brief Set the phase current offset directly.
 * @param[in,out] controller FOC controller object.
 * @param[in] offset Phase current offset in amperes.
 * @return true when set; false when controller is NULL.
 */
bool foc_current_offset_set(Foc *controller, FocPhase offset);

/**
 * @brief Map a legacy control mode onto the flat VF/IF model.
 * @param[in,out] controller FOC controller object.
 * @param[in] mode Legacy control mode.
 * @return true when mapped; false when the input is invalid.
 */
bool foc_set_legacy_control_mode(Foc *controller, FocControlMode mode);

/**
 * @brief Execute one flat FOC update.
 * @param[in,out] controller FOC controller object.
 * @return true when duty outputs are valid; false when inputs are invalid.
 */
bool foc_run(Foc *controller);

/**
 * @brief Legacy entry point retained as a wrapper around foc_run().
 * @param[in,out] controller FOC controller object.
 * @return true when duty outputs are valid; false when inputs are invalid.
 */
bool foc(Foc *controller);

#ifdef __cplusplus
}
#endif

#endif
