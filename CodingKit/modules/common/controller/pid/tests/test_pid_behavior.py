from pid_model import Config, Input, Limit, PidModel, PID_ERR_FREQ, PID_ERR_LIMIT, PID_OK, pid


def base_model(kp=1.0, ki=0.0, kd=0.0):
    return PidModel(Config(
        kp=kp,
        ki=ki,
        kd=kd,
        control_freq=1000.0,
        output_limit=Limit(True, -1.0, 1.0),
        integral_limit=Limit(True, -0.5, 0.5),
        anti_windup_gain=1.0,
    ))


def test_p_control_uses_struct_input():
    model = base_model(kp=2.0)
    model.input = Input(setpoint=0.2, feedback=0.1, feedforward=0.0)
    out = pid(model)
    assert abs(out - 0.2) < 1e-9
    assert model.state.status == PID_OK


def test_feedforward_is_struct_field():
    model = base_model(kp=1.0)
    model.input.setpoint = 0.1
    model.input.feedback = 0.0
    model.input.feedforward = 0.2
    assert abs(pid(model) - 0.3) < 1e-9


def test_pi_anti_windup_runs_when_saturated():
    model = base_model(kp=4.0, ki=10.0)
    for _ in range(100):
        model.input.setpoint = 10.0
        model.input.feedback = 0.0
        out = pid(model)
    assert out == 1.0
    assert model.state.saturated
    assert model.state.integral <= 0.5


def test_pid_anti_windup_runs_when_saturated():
    model = base_model(kp=4.0, ki=10.0, kd=0.01)
    for _ in range(100):
        model.input.setpoint = 10.0
        model.input.feedback = 0.0
        out = pid(model)
    assert out == 1.0
    assert model.state.saturated
    assert model.state.integral <= 0.5


def test_pd_has_no_integral_state():
    model = base_model(kp=1.0, ki=0.0, kd=0.1)
    model.input.setpoint = 1.0
    model.input.feedback = 0.0
    pid(model)
    model.input.feedback = 0.2
    pid(model)
    assert model.state.integral == 0.0


def test_input_limits():
    model = base_model(kp=1.0)
    model.config.setpoint_limit = Limit(True, -0.2, 0.2)
    model.config.feedback_limit = Limit(True, -0.1, 0.1)
    model.input.setpoint = 10.0
    model.input.feedback = -10.0
    pid(model)
    assert model.state.setpoint_limited == 0.2
    assert model.state.feedback_limited == -0.1


def test_setpoint_rate_limit():
    model = base_model(kp=1.0)
    model.config.setpoint_rate_enable = True
    model.config.setpoint_rate = 1.0
    model.input.setpoint = 0.0
    model.input.feedback = 0.0
    pid(model)
    model.input.setpoint = 10.0
    pid(model)
    assert abs(model.state.setpoint_limited - 0.001) < 1e-12


def test_invalid_control_freq_keeps_previous_output():
    model = base_model(kp=1.0)
    model.input.setpoint = 0.5
    pid(model)
    model.config.control_freq = 0.0
    out = pid(model)
    assert out == model.state.output
    assert model.state.status == PID_ERR_FREQ


def test_invalid_limit_sets_status():
    model = base_model(kp=1.0)
    model.config.output_limit = Limit(True, 1.0, -1.0)
    pid(model)
    assert model.state.status == PID_ERR_LIMIT
