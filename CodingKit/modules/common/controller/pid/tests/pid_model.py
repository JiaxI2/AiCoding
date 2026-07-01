from __future__ import annotations

from dataclasses import dataclass, field

PID_OK = 0
PID_ERR_NULL = -1
PID_ERR_FREQ = -2
PID_ERR_LIMIT = -3
PID_EPSILON = 1.0e-12


@dataclass
class Limit:
    enable: bool = False
    min: float = 0.0
    max: float = 0.0


@dataclass
class Input:
    setpoint: float = 0.0
    feedback: float = 0.0
    feedforward: float = 0.0


@dataclass
class Config:
    kp: float = 0.0
    ki: float = 0.0
    kd: float = 0.0
    control_freq: float = 1000.0
    setpoint_limit: Limit = field(default_factory=Limit)
    feedback_limit: Limit = field(default_factory=Limit)
    error_limit: Limit = field(default_factory=Limit)
    integral_limit: Limit = field(default_factory=Limit)
    output_limit: Limit = field(default_factory=Limit)
    setpoint_rate_enable: bool = False
    setpoint_rate: float = 0.0
    derivative_filter_coef: float = 0.0
    anti_windup_gain: float = 0.0
    deadband: float = 0.0


@dataclass
class State:
    setpoint: float = 0.0
    setpoint_limited: float = 0.0
    feedback: float = 0.0
    feedback_limited: float = 0.0
    error: float = 0.0
    previous_error: float = 0.0
    derivative: float = 0.0
    derivative_filtered: float = 0.0
    proportional: float = 0.0
    integral: float = 0.0
    derivative_term: float = 0.0
    feedforward: float = 0.0
    raw_output: float = 0.0
    output: float = 0.0
    saturated: bool = False
    initialized: bool = False
    status: int = PID_OK


@dataclass
class PidModel:
    config: Config
    input: Input = field(default_factory=Input)
    state: State = field(default_factory=State)


def limit_valid(limit: Limit) -> bool:
    return (not limit.enable) or (limit.min <= limit.max)


def clamp(value: float, limit: Limit) -> float:
    if not limit.enable:
        return value
    if value > limit.max:
        return limit.max
    if value < limit.min:
        return limit.min
    return value


def clean_filter_coef(filter_coef: float) -> float:
    return min(max(filter_coef, 0.0), 1.0)


def check_config(config: Config) -> int:
    if config.control_freq <= PID_EPSILON:
        return PID_ERR_FREQ
    if not all(limit_valid(x) for x in (
        config.setpoint_limit,
        config.feedback_limit,
        config.error_limit,
        config.integral_limit,
        config.output_limit,
    )):
        return PID_ERR_LIMIT
    if config.setpoint_rate_enable and config.setpoint_rate < 0.0:
        return PID_ERR_LIMIT
    if config.anti_windup_gain < 0.0:
        return PID_ERR_LIMIT
    return PID_OK


def pid(model: PidModel) -> float:
    cfg = model.config
    inp = model.input
    st = model.state
    status = check_config(cfg)
    if status != PID_OK:
        st.status = status
        return st.output

    period = 1.0 / cfg.control_freq

    setpoint_limited = clamp(inp.setpoint, cfg.setpoint_limit)
    if cfg.setpoint_rate_enable and st.initialized:
        max_step = cfg.setpoint_rate * period
        delta = setpoint_limited - st.setpoint_limited
        if delta > max_step:
            setpoint_limited = st.setpoint_limited + max_step
        elif delta < -max_step:
            setpoint_limited = st.setpoint_limited - max_step

    feedback_limited = clamp(inp.feedback, cfg.feedback_limit)
    error = clamp(setpoint_limited - feedback_limited, cfg.error_limit)
    derivative = 0.0 if not st.initialized else (error - st.error) / period
    filter_coef = clean_filter_coef(cfg.derivative_filter_coef)
    st.derivative_filtered = filter_coef * st.derivative_filtered + (1.0 - filter_coef) * derivative
    st.proportional = cfg.kp * error
    st.derivative_term = cfg.kd * st.derivative_filtered
    st.feedforward = inp.feedforward
    raw_output = st.proportional + st.integral + st.derivative_term + st.feedforward
    if cfg.deadband > 0.0 and abs(raw_output) < cfg.deadband:
        raw_output = 0.0
    limited_output = clamp(raw_output, cfg.output_limit)

    if cfg.ki != 0.0:
        aw_error = limited_output - raw_output
        st.integral += cfg.ki * (error + cfg.anti_windup_gain * aw_error) * period
        st.integral = clamp(st.integral, cfg.integral_limit)
    else:
        st.integral = 0.0

    st.setpoint = inp.setpoint
    st.setpoint_limited = setpoint_limited
    st.feedback = inp.feedback
    st.feedback_limited = feedback_limited
    st.previous_error = st.error
    st.error = error
    st.derivative = derivative
    st.raw_output = raw_output
    st.output = limited_output
    st.saturated = limited_output != raw_output
    st.initialized = True
    st.status = PID_OK
    return limited_output
