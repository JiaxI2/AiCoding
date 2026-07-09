import math

FOC_PI = math.pi
FOC_TWO_PI = 2.0 * math.pi
FOC_ONE_BY_SQRT3 = 1.0 / math.sqrt(3.0)
FOC_SQRT3_BY_2 = math.sqrt(3.0) / 2.0
FOC_MODE_VF = 0
FOC_MODE_IF = 1
FOC_ANGLE_SENSOR = 0
FOC_ANGLE_OPEN_LOOP = 1


def clamp(value, min_value, max_value):
    if min_value > max_value:
        return value
    return max(min_value, min(max_value, value))


def wrap_0_2pi(angle):
    result = math.fmod(angle, FOC_TWO_PI)
    if result < 0.0:
        result += FOC_TWO_PI
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


def integrate_open_loop_angle(theta, direction, freq_hz, control_freq):
    if control_freq <= 0.0:
        return None
    direction = -1.0 if direction < 0.0 else 1.0
    omega = direction * FOC_TWO_PI * freq_hz
    theta = wrap_0_2pi(theta + (omega / control_freq))
    return theta, omega


def limit_voltage(vd, vq, max_voltage):
    saturated = False
    mag = math.sqrt(vd * vd + vq * vq)
    if max_voltage > 0.0 and mag > max_voltage:
        scale = max_voltage / mag
        vd *= scale
        vq *= scale
        saturated = True
    return vd, vq, saturated


def foc_vf(vbus, theta, vd, vq, max_voltage=0.0):
    if vbus <= 0.0:
        return None
    vd, vq, saturated = limit_voltage(vd, vq, max_voltage)
    vab = inv_park((vd, vq), theta)
    pwm = svpwm((vab[0] / vbus, vab[1] / vbus))
    pwm["saturated"] = pwm["saturated"] or saturated
    pwm["vab"] = vab
    return pwm


def foc_if(vbus, theta, phase_current, id_ref, iq_ref, kp, max_voltage=0.0, current_offset=(0.0, 0.0, 0.0)):
    if vbus <= 0.0:
        return None
    phase_corrected = phase_sub_offset(phase_current, current_offset)
    ab = clarke(phase_corrected)
    dq = park(ab, theta)
    err_d = id_ref - dq[0]
    err_q = iq_ref - dq[1]
    vd = kp * err_d
    vq = kp * err_q
    pwm = foc_vf(vbus, theta, vd, vq, max_voltage)
    pwm["phase_corrected"] = phase_corrected
    pwm["current_dq"] = dq
    pwm["current_error"] = (err_d, err_q)
    return pwm
