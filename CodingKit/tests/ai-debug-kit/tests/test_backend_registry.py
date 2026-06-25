from ai_debug.backends.registry import backend_names, create_backend
from ai_debug.core.address import TargetAddress


def test_registry_lists_simulator_and_jlink() -> None:
    assert backend_names() == ["jlink", "simulator"]


def test_simulator_backend_contract() -> None:
    backend = create_backend("simulator")

    discover = backend.discover()
    capabilities = backend.capabilities()
    valid = backend.try_read_memory(TargetAddress(space="data", value=0, address_unit_bits=8), 4)
    invalid = backend.try_read_memory(TargetAddress(space="data", value=4096, address_unit_bits=8), 4)

    assert discover.ok is True
    assert capabilities.memory_read is True
    assert valid.ok is True
    assert valid.data["block"].octet_length == 4
    assert invalid.ok is False
    assert invalid.code == "INVALID_ARGUMENT"
