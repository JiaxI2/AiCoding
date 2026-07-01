from foc_model import *


def near(a, b, tol=1e-6):
    return abs(a - b) <= tol


def test_clarke_balanced():
    alpha, beta = clarke((1.0, -0.5, -0.5))
    assert near(alpha, 1.0)
    assert near(beta, 0.0)


def test_park_roundtrip():
    ab = (0.3, -0.7)
    angle = 1.234
    dq = park(ab, angle)
    ab2 = inv_park(dq, angle)
    assert near(ab[0], ab2[0])
    assert near(ab[1], ab2[1])


def test_inverse_clarke_sum_zero():
    phase = inv_clarke((0.25, 0.4))
    assert near(sum(phase), 0.0)


def test_svpwm_zero_vector():
    out = svpwm((0.0, 0.0))
    assert out["valid"] is True
    assert all(near(x, 0.5) for x in out["duty"])


def test_svpwm_auto_scale():
    out = svpwm((2.0, 0.0))
    assert out["valid"] is True
    assert out["saturated"] is True
    assert out["scale"] < 1.0
    assert all(0.0 <= x <= 1.0 for x in out["duty"])


def test_angle_update_sensor():
    angle, speed = angle_update_sensor(4, 1, 0.0, math.pi / 8.0, 10.0)
    assert near(angle, math.pi / 2.0)
    assert near(speed, 40.0)


def test_angle_update_open_loop():
    out = angle_update_open_loop(0.0, 1, 100.0, 1000.0)
    assert out is not None
    angle, speed, stored = out
    assert near(angle, 0.1)
    assert near(speed, 100.0)
    assert near(stored, 0.1)


def test_angle_calibrate():
    offset = angle_calibrate(4, 1, math.pi / 8.0, 0.0)
    angle, _ = angle_update_sensor(4, 1, offset, math.pi / 8.0, 0.0)
    assert near(angle, 0.0)


def test_open_voltage_output_valid():
    out = foc_open_voltage(24.0, 0.0, 0.0, 3.0, 12.0)
    assert out is not None
    assert out["valid"] is True
    assert all(0.0 <= x <= 1.0 for x in out["duty"])


def test_closed_current_output_valid():
    out = foc_closed_current(24.0, 0.0, (0.0, 0.0, 0.0), 0.0, 1.0, 2.0, 800.0, max_voltage=12.0)
    assert out is not None
    assert out["valid"] is True
    assert out["current_error"][1] == 1.0


def test_voltage_limit():
    out = foc_open_voltage(24.0, 0.5, 20.0, 0.0, 12.0)
    assert out is not None
    assert out["saturated"] is True



def test_open_loop_angle_accumulates_multiple_steps():
    stored = 0.0
    for _ in range(5):
        angle, speed, stored = angle_update_open_loop(stored, 1, 100.0, 1000.0)
    assert near(stored, 0.5)
    assert near(angle, 0.5)
    assert near(speed, 100.0)


def test_current_offset_accumulate_average():
    offset, count = offset_accumulate([
        (0.10, -0.20, 0.05),
        (0.12, -0.18, 0.07),
        (0.08, -0.22, 0.03),
    ])
    assert count == 3
    assert near(offset[0], 0.10)
    assert near(offset[1], -0.20)
    assert near(offset[2], 0.05)


def test_closed_current_subtracts_offset_before_clarke():
    out = foc_closed_current(24.0, 0.0, (1.10, -0.55, -0.55), 1.0, 0.0, 2.0, 800.0, max_voltage=12.0, current_offset=(0.10, -0.05, -0.05))
    assert out is not None
    assert near(out["phase_corrected"][0], 1.0)
    assert near(out["phase_corrected"][1], -0.5)
    assert near(out["phase_corrected"][2], -0.5)
    assert near(out["current_dq"][0], 1.0)



def test_motion_position_velocity_current_chain():
    state = {"pos_sp": 0.0, "vel_sp": 0.0, "q_base": 0.0, "vel_i": 0.0}
    cfg = {
        "control_freq": 1000.0,
        "control_mode": FOC_MOTION_CONTROL_POSITION,
        "input_mode": FOC_MOTION_INPUT_PASSTHROUGH,
        "pos_gain": 10.0,
        "vel_gain": 2.0,
        "vel_ki": 100.0,
        "vel_i_limit": 5.0,
        "vel_limit": 100.0,
        "current_limit": 20.0,
    }
    inp = {"pos_sp": 1.0, "vel_sp": 0.0, "q_ff": 0.5, "pos_fb": 0.8, "vel_fb": 1.0}
    out = motion_update(state, cfg, inp)
    assert out["valid"] is True
    assert near(out["pos_err"], 0.2)
    assert near(out["vel_err"], 1.0)
    assert near(out["q_out"], 2.5)


def test_motion_input_position_filter_moves_smoothly():
    state = {"pos_sp": 0.0, "vel_sp": 0.0, "q_base": 0.0, "vel_i": 0.0}
    cfg = {
        "control_freq": 1000.0,
        "control_mode": FOC_MOTION_CONTROL_POSITION,
        "input_mode": FOC_MOTION_INPUT_POS_FILTER,
        "filter_bw": 2.0,
        "inertia_ff": 0.1,
    }
    inp = {"pos_sp": 1.0, "vel_sp": 0.0, "q_ff": 0.0, "pos_fb": 0.0, "vel_fb": 0.0}
    out = motion_update(state, cfg, inp)
    assert out["valid"] is True
    assert out["pos_sp"] > 0.0
    assert out["vel_sp"] > 0.0
    assert out["accel_ff"] > 0.0


def test_motion_anticogging_feedforward():
    state = {"pos_sp": 0.0, "vel_sp": 0.0, "q_base": 0.0, "vel_i": 0.0}
    cfg = {
        "control_freq": 1000.0,
        "control_mode": FOC_MOTION_CONTROL_CURRENT,
        "input_mode": FOC_MOTION_INPUT_PASSTHROUGH,
        "current_limit": 10.0,
        "enable_anti": True,
        "anti": [0.0, 0.1, -0.2, 0.3],
        "anti_scale": 4.0,
    }
    inp = {"q_ff": 1.0, "pos_fb": 0.75, "vel_fb": 0.0}
    out = motion_update(state, cfg, inp)
    assert near(out["anti"], 0.3)
    assert near(out["q_out"], 1.3)


def test_foc_motion_current_output_valid():
    motion_state = {"pos_sp": 0.0, "vel_sp": 0.0, "q_base": 0.0, "vel_i": 0.0}
    motion_cfg = {
        "control_freq": 1000.0,
        "control_mode": FOC_MOTION_CONTROL_VELOCITY,
        "input_mode": FOC_MOTION_INPUT_PASSTHROUGH,
        "vel_gain": 2.0,
        "vel_ki": 50.0,
        "current_limit": 8.0,
    }
    motion_input = {"vel_sp": 2.0, "q_ff": 0.0, "vel_fb": 1.5, "pos_fb": 0.0}
    out = foc_motion_current(24.0, 0.0, (0.0, 0.0, 0.0), motion_state, motion_cfg, motion_input, 2.0, 800.0, max_voltage=12.0)
    assert out is not None
    assert out["valid"] is True
    assert near(out["motion"]["q_out"], 1.0)


def run_all():
    tests = [
        test_clarke_balanced,
        test_park_roundtrip,
        test_inverse_clarke_sum_zero,
        test_svpwm_zero_vector,
        test_svpwm_auto_scale,
        test_angle_update_sensor,
        test_angle_update_open_loop,
        test_open_loop_angle_accumulates_multiple_steps,
        test_angle_calibrate,
        test_current_offset_accumulate_average,
        test_open_voltage_output_valid,
        test_closed_current_output_valid,
        test_closed_current_subtracts_offset_before_clarke,
        test_voltage_limit,
        test_motion_position_velocity_current_chain,
        test_motion_input_position_filter_moves_smoothly,
        test_motion_anticogging_feedforward,
        test_foc_motion_current_output_valid,
    ]
    passed = 0
    failed = []
    for test in tests:
        try:
            test()
            passed += 1
        except Exception as exc:
            failed.append((test.__name__, str(exc)))
    return passed, failed


if __name__ == "__main__":
    passed, failed = run_all()
    print(f"Passed: {passed}")
    print(f"Failed: {len(failed)}")
    for name, reason in failed:
        print(f"- {name}: {reason}")
    raise SystemExit(1 if failed else 0)
