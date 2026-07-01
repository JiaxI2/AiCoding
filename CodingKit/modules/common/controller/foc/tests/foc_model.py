import math

FOC_PI = math.pi
FOC_TWO_PI = 2.0 * math.pi
FOC_ONE_BY_SQRT3 = 1.0 / math.sqrt(3.0)
FOC_SQRT3_BY_2 = math.sqrt(3.0) / 2.0
FOC_ANGLE_MODE_SENSOR = 0
FOC_ANGLE_MODE_OPEN_LOOP = 1
FOC_ANGLE_MODE_FIXED = 2


def clamp(value, min_value, max_value):
    if min_value > max_value:
        return value
    return max(min_value, min(max_value, value))


def wrap_0_2pi(angle):
    result = math.fmod(angle, FOC_TWO_PI)
    if result < 0.0:
        result += FOC_TWO_PI
    return result


def wrap_pm_pi(angle):
    result = wrap_0_2pi(angle)
    if result > FOC_PI:
        result -= FOC_TWO_PI
    return result



def phase_sub_offset(phase, offset):
    return (phase[0] - offset[0], phase[1] - offset[1], phase[2] - offset[2])

def offset_accumulate(samples):
    offset = [0.0, 0.0, 0.0]
    count = 0
    for sample in samples:
        count += 1
        for i in range(3):
            offset[i] += (sample[i] - offset[i]) / count
    return tuple(offset), count

def clarke(phase):
    a, b, c = phase
    return (a, FOC_ONE_BY_SQRT3 * (b - c))


def park(ab, angle):
    alpha, beta = ab
    s = math.sin(angle)
    c = math.cos(angle)
    return (c * alpha + s * beta, c * beta - s * alpha)


def inv_park(dq, angle):
    d, q = dq
    s = math.sin(angle)
    c = math.cos(angle)
    return (c * d - s * q, s * d + c * q)


def inv_clarke(ab):
    alpha, beta = ab
    return (
        alpha,
        -0.5 * alpha + FOC_SQRT3_BY_2 * beta,
        -0.5 * alpha - FOC_SQRT3_BY_2 * beta,
    )


def svpwm(mod, max_mod=FOC_SQRT3_BY_2, auto_scale=True):
    alpha, beta = mod
    mag = math.sqrt(alpha * alpha + beta * beta)
    saturated = False
    valid = True
    scale = 1.0
    if max_mod <= 0.0:
        max_mod = FOC_SQRT3_BY_2
    max_mod = min(max_mod, FOC_SQRT3_BY_2)
    if mag > max_mod:
        saturated = True
        if auto_scale:
            scale = max_mod / mag
            alpha *= scale
            beta *= scale
        else:
            valid = False
    phase = inv_clarke((alpha, beta))
    pmax = max(phase)
    pmin = min(phase)
    common = -0.5 * (pmax + pmin)
    duty = tuple(clamp(0.5 + x + common, 0.0, 1.0) for x in phase)
    if any(abs(x) <= 0.0 or abs(x - 1.0) <= 0.0 for x in duty):
        saturated = True
    return {"duty": duty, "scale": scale, "saturated": saturated, "valid": valid}


def angle_update_sensor(pole_pairs, direction, offset, mech_angle, mech_speed, comp_time=0.0):
    direction = -1 if direction < 0 else 1
    e_speed = direction * pole_pairs * mech_speed
    e_angle = direction * pole_pairs * mech_angle + offset + e_speed * comp_time
    return wrap_0_2pi(e_angle), e_speed


def angle_update_open_loop(open_loop_angle, direction, e_speed_cmd, control_freq, offset=0.0, comp_time=0.0):
    direction = -1 if direction < 0 else 1
    if control_freq <= 0.0:
        return None
    period = 1.0 / control_freq
    e_speed = direction * e_speed_cmd
    open_loop_angle = wrap_0_2pi(open_loop_angle + e_speed * period)
    e_angle = open_loop_angle + offset + e_speed * comp_time
    return wrap_0_2pi(e_angle), e_speed, open_loop_angle


def angle_update(pole_pairs, direction, offset, mech_angle, mech_speed, comp_time=0.0):
    return angle_update_sensor(pole_pairs, direction, offset, mech_angle, mech_speed, comp_time)


def angle_calibrate(pole_pairs, direction, mech_angle, align_electrical):
    direction = -1 if direction < 0 else 1
    raw = direction * pole_pairs * mech_angle
    return wrap_pm_pi(align_electrical - raw)


def foc_open_voltage(vbus, angle, vd, vq, max_voltage=0.0):
    if vbus <= 0.0:
        return None
    mag = math.sqrt(vd * vd + vq * vq)
    saturated = False
    if max_voltage > 0.0 and mag > max_voltage:
        scale = max_voltage / mag
        vd *= scale
        vq *= scale
        saturated = True
    vab = inv_park((vd, vq), angle)
    mod = (vab[0] / vbus, vab[1] / vbus)
    pwm = svpwm(mod)
    pwm["saturated"] = pwm["saturated"] or saturated
    pwm["vab"] = vab
    return pwm


def foc_voltage_mode(vbus, angle, vd, vq, max_voltage=0.0):
    return foc_open_voltage(vbus, angle, vd, vq, max_voltage)


def foc_closed_current(vbus, angle, phase_current, id_ref, iq_ref, kp, ki, int_d=0.0, int_q=0.0, max_voltage=0.0, current_offset=(0.0, 0.0, 0.0)):
    if vbus <= 0.0:
        return None
    phase_corrected = phase_sub_offset(phase_current, current_offset)
    ab = clarke(phase_corrected)
    dq = park(ab, angle)
    err_d = id_ref - dq[0]
    err_q = iq_ref - dq[1]
    vd = int_d + kp * err_d
    vq = int_q + kp * err_q
    pwm = foc_open_voltage(vbus, angle, vd, vq, max_voltage)
    pwm["phase_corrected"] = phase_corrected
    pwm["current_dq"] = dq
    pwm["current_error"] = (err_d, err_q)
    return pwm


FOC_MOTION_CONTROL_CURRENT = 0
FOC_MOTION_CONTROL_VELOCITY = 1
FOC_MOTION_CONTROL_POSITION = 2
FOC_MOTION_INPUT_INACTIVE = 0
FOC_MOTION_INPUT_PASSTHROUGH = 1
FOC_MOTION_INPUT_POS_FILTER = 2
FOC_MOTION_INPUT_VEL_RAMP = 3
FOC_MOTION_INPUT_CURRENT_RAMP = 4


def motion_lookup_anticogging(position_feedback, table, scale=0.0):
    if not table:
        return 0.0
    length = len(table)
    if scale <= 0.0:
        scale = float(length)
    index = math.floor(position_feedback * scale) % length
    return table[index]


def motion_update(state, cfg, inp):
    period = 1.0 / cfg.get("control_freq", 10000.0)
    state = dict(state)
    input_mode = cfg.get("input_mode", FOC_MOTION_INPUT_PASSTHROUGH)
    accel_ff = 0.0
    if input_mode == FOC_MOTION_INPUT_PASSTHROUGH:
        state["pos_sp"] = inp.get("pos_sp", 0.0)
        state["vel_sp"] = inp.get("vel_sp", 0.0)
        state["q_base"] = inp.get("q_ff", 0.0)
    elif input_mode == FOC_MOTION_INPUT_VEL_RAMP:
        max_step = abs(cfg.get("vel_ramp", 0.0) * period)
        old = state.get("vel_sp", 0.0)
        new = old + clamp(inp.get("vel_sp", 0.0) - old, -max_step, max_step) if max_step > 0.0 else inp.get("vel_sp", 0.0)
        state["vel_sp"] = new
        accel_ff = ((new - old) / period) * cfg.get("inertia_ff", 0.0)
        state["q_base"] = inp.get("q_ff", 0.0) + accel_ff
    elif input_mode == FOC_MOTION_INPUT_POS_FILTER:
        bw = cfg.get("filter_bw", 2.0)
        ki = 2.0 * min(bw, 0.25 * cfg.get("control_freq", 10000.0))
        kp = 0.25 * ki * ki
        accel = kp * (inp.get("pos_sp", 0.0) - state.get("pos_sp", 0.0)) + ki * (inp.get("vel_sp", 0.0) - state.get("vel_sp", 0.0))
        state["vel_sp"] = state.get("vel_sp", 0.0) + period * accel
        state["pos_sp"] = state.get("pos_sp", 0.0) + period * state["vel_sp"]
        accel_ff = accel * cfg.get("inertia_ff", 0.0)
        state["q_base"] = inp.get("q_ff", 0.0) + accel_ff
    elif input_mode == FOC_MOTION_INPUT_CURRENT_RAMP:
        max_step = abs(cfg.get("current_ramp", 0.0) * period)
        old = state.get("q_base", 0.0)
        state["q_base"] = old + clamp(inp.get("q_ff", 0.0) - old, -max_step, max_step) if max_step > 0.0 else inp.get("q_ff", 0.0)

    vel_cmd = state.get("vel_sp", 0.0)
    pos_err = 0.0
    if cfg.get("control_mode", 0) >= FOC_MOTION_CONTROL_POSITION:
        pos_err = state.get("pos_sp", 0.0) - inp.get("pos_fb", 0.0)
        vel_cmd += cfg.get("pos_gain", 0.0) * pos_err
    if cfg.get("vel_limit", 0.0) > 0.0:
        vel_cmd = clamp(vel_cmd, -cfg["vel_limit"], cfg["vel_limit"])

    q = state.get("q_base", 0.0)
    vel_err = 0.0
    if cfg.get("control_mode", 0) >= FOC_MOTION_CONTROL_VELOCITY:
        vel_err = vel_cmd - inp.get("vel_fb", 0.0)
        q += cfg.get("vel_gain", 0.0) * vel_err + state.get("vel_i", 0.0)
    anti = motion_lookup_anticogging(inp.get("pos_fb", 0.0), cfg.get("anti", []), cfg.get("anti_scale", 0.0)) if cfg.get("enable_anti", False) else 0.0
    q += anti
    limited = False
    if cfg.get("current_limit", 0.0) > 0.0:
        q_l = clamp(q, -cfg["current_limit"], cfg["current_limit"])
        limited = (q_l != q)
        q = q_l
    if cfg.get("control_mode", 0) >= FOC_MOTION_CONTROL_VELOCITY:
        if limited:
            state["vel_i"] = state.get("vel_i", 0.0) * cfg.get("decay", 0.99)
        else:
            state["vel_i"] = state.get("vel_i", 0.0) + cfg.get("vel_ki", 0.0) * vel_err * period
            if cfg.get("vel_i_limit", 0.0) > 0.0:
                state["vel_i"] = clamp(state["vel_i"], -cfg["vel_i_limit"], cfg["vel_i_limit"])
    else:
        state["vel_i"] = 0.0
    state.update({"q_out": q, "d_out": inp.get("d_ff", 0.0), "pos_err": pos_err, "vel_err": vel_err, "anti": anti, "accel_ff": accel_ff, "valid": True})
    return state


def foc_motion_current(vbus, angle, phase_current, motion_state, motion_cfg, motion_input, kp, ki, max_voltage=0.0, current_offset=(0.0, 0.0, 0.0)):
    st = motion_update(motion_state, motion_cfg, motion_input)
    pwm = foc_closed_current(vbus, angle, phase_current, st["d_out"], st["q_out"], kp, ki, max_voltage=max_voltage, current_offset=current_offset)
    pwm["motion"] = st
    return pwm
