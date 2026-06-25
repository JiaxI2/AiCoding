from ai_debug.backends.simulator import SimulatorBackend
from ai_debug.core.address import TargetAddress
from ai_debug.core.policy import Approval, Policy, RiskLevel


def test_simulator_declares_core_capabilities() -> None:
    backend = SimulatorBackend()

    capabilities = backend.capabilities()

    assert capabilities.artifact_load is True
    assert capabilities.memory_read is True
    assert capabilities.memory_write is True
    assert capabilities.variable_read is True
    assert capabilities.telemetry_capture is True
    assert capabilities.flash is False


def test_simulator_read_memory_returns_requested_octets() -> None:
    backend = SimulatorBackend(size=64)

    block = backend.read_memory(TargetAddress(space="data", value=4, address_unit_bits=8), 8)

    assert block.address.value == 4
    assert block.octet_length == 8
    assert block.data == bytes(range(4, 12))


def test_simulator_rejects_invalid_memory_range() -> None:
    backend = SimulatorBackend(size=16)

    result = backend.try_read_memory(TargetAddress(space="data", value=12, address_unit_bits=8), 8)

    assert result.ok is False
    assert result.code == "INVALID_ARGUMENT"


def test_write_requires_policy_approval_for_l3_operation() -> None:
    backend = SimulatorBackend(size=16)
    policy = Policy(read_only=True)

    denied = backend.try_write_memory(
        TargetAddress(space="data", value=0, address_unit_bits=8),
        b"\xAA\xBB",
        policy=policy,
        approval=Approval.none(),
    )

    assert denied.ok is False
    assert denied.code == "POLICY_DENIED"

    approved = backend.try_write_memory(
        TargetAddress(space="data", value=0, address_unit_bits=8),
        b"\xAA\xBB",
        policy=Policy(read_only=False),
        approval=Approval(granted_levels={RiskLevel.L3}),
    )

    assert approved.ok is True
    assert backend.read_memory(TargetAddress(space="data", value=0, address_unit_bits=8), 2).data == b"\xAA\xBB"
