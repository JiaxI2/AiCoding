from pid_model import Config, Limit, PidModel, pid


def test_first_order_plant_response():
    model = PidModel(Config(
        kp=2.0,
        ki=5.0,
        kd=0.0,
        control_freq=1000.0,
        output_limit=Limit(True, -1.0, 1.0),
        integral_limit=Limit(True, -1.0, 1.0),
        anti_windup_gain=1.0,
    ))
    plant = 0.0
    max_output = 0.0
    settled_at = None
    values = []
    for i in range(5000):
        model.input.setpoint = 1.0
        model.input.feedback = plant
        model.input.feedforward = 0.0
        u = pid(model)
        plant += 0.001 * ((2.0 * u) - plant) / 0.15
        values.append(plant)
        max_output = max(max_output, abs(u))
        if settled_at is None and i > 10 and abs(plant - 1.0) < 0.02:
            settled_at = i * 0.001
    assert abs(values[-1] - 1.0) < 1e-3
    assert max_output <= 1.0
    assert settled_at is not None and settled_at < 2.0
