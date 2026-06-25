from ai_debug.backends.jlink import JLinkBackend, TargetProfile
from ai_debug.core.address import TargetAddress
from ai_debug.probes.jlink import FakePylinkModule


def test_jlink_fake_can_discover_validate_read_memory_and_register() -> None:
    backend = JLinkBackend(probe_module=FakePylinkModule, profile=TargetProfile.fake_default())

    discover = backend.discover()
    validate = backend.validate()
    block = backend.read_memory(TargetAddress(space="data", value=0x20000000, address_unit_bits=8), 4)
    reg = backend.try_read_register("R0")

    assert discover.ok is True
    assert discover.data["devices"][0]["serial_number"] == 12345678
    assert validate.ok is True
    assert validate.data["target_identity"]["target_id"] == "0x0BB11477"
    assert validate.data["target_identity"]["architecture"] == "generic"
    assert block.data == b"\x00\x01\x02\x03"
    assert reg.ok is True
    assert reg.data["value"] == "0x12345678"


def test_jlink_missing_dependency_returns_dependency_missing() -> None:
    def missing_importer(_name: str):
        raise ModuleNotFoundError("pylink")

    backend = JLinkBackend.from_optional_dependency(importer=missing_importer)

    result = backend.discover()

    assert result.ok is False
    assert result.code == "DEPENDENCY_MISSING"


def test_jlink_memory_read_rejects_range_outside_profile() -> None:
    backend = JLinkBackend(probe_module=FakePylinkModule, profile=TargetProfile.fake_default())

    result = backend.try_read_memory(TargetAddress(space="data", value=0x10000000, address_unit_bits=8), 4)

    assert result.ok is False
    assert result.code == "POLICY_DENIED"


def test_c2000_profile_is_not_arm_specific_and_uses_word_address_units() -> None:
    profile = TargetProfile.c2000_c28x_default(device="TMS320F28379D", core="cpu1")
    backend = JLinkBackend(probe_module=FakePylinkModule, profile=profile)

    validate = backend.validate()
    read = backend.try_read_memory(TargetAddress(space="data", value=0x00008000, address_unit_bits=16), 2)

    assert profile.architecture == "c28x"
    assert profile.core == "cpu1"
    assert profile.address_unit_bits == 16
    assert validate.ok is True
    assert validate.data["target_identity"]["architecture"] == "c28x"
    assert validate.data["target_identity"]["core"] == "cpu1"
    assert read.ok is True
