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


def test_open_loop_angle_accumulates_multiple_steps():
    theta = 0.0
    omega = 0.0
    for _ in range(5):
        theta, omega = integrate_open_loop_angle(theta, 1.0, 100.0 / FOC_TWO_PI, 1000.0)
    assert near(theta, 0.5)
    assert near(omega, 100.0)


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


def test_vf_output_valid():
    out = foc_vf(24.0, 0.0, 0.0, 3.0, 12.0)
    assert out is not None
    assert out["valid"] is True
    assert all(0.0 <= x <= 1.0 for x in out["duty"])


def test_if_output_valid():
    out = foc_if(24.0, 0.0, (0.0, 0.0, 0.0), 0.0, 1.0, 2.0, max_voltage=12.0)
    assert out is not None
    assert out["valid"] is True
    assert out["current_error"][1] == 1.0


def test_if_subtracts_offset_before_clarke():
    out = foc_if(24.0, 0.0, (1.10, -0.55, -0.55), 1.0, 0.0, 2.0, max_voltage=12.0, current_offset=(0.10, -0.05, -0.05))
    assert out is not None
    assert near(out["phase_corrected"][0], 1.0)
    assert near(out["phase_corrected"][1], -0.5)
    assert near(out["phase_corrected"][2], -0.5)
    assert near(out["current_dq"][0], 1.0)


def test_voltage_limit():
    out = foc_vf(24.0, 0.5, 20.0, 0.0, 12.0)
    assert out is not None
    assert out["saturated"] is True


def run_all():
    tests = [
        test_clarke_balanced,
        test_park_roundtrip,
        test_inverse_clarke_sum_zero,
        test_svpwm_zero_vector,
        test_svpwm_auto_scale,
        test_open_loop_angle_accumulates_multiple_steps,
        test_current_offset_accumulate_average,
        test_vf_output_valid,
        test_if_output_valid,
        test_if_subtracts_offset_before_clarke,
        test_voltage_limit,
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
