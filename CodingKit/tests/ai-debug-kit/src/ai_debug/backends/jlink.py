from __future__ import annotations

import importlib
import os
from dataclasses import dataclass, field
from typing import Any, Callable

from ai_debug.backends.base import MemoryBlock
from ai_debug.core.address import TargetAddress
from ai_debug.core.capability import Capabilities
from ai_debug.core.result import OperationResult
from ai_debug.probes.jlink import FakePylinkModule, JLinkProbe


@dataclass(frozen=True)
class MemoryRange:
    start: int
    length: int
    space: str = "data"

    @property
    def end(self) -> int:
        return self.start + self.length

    def contains(self, address: TargetAddress, octet_length: int) -> bool:
        if address.space != self.space:
            return False
        if address.value < self.start:
            return False
        return address.value + octet_length <= self.end


@dataclass(frozen=True)
class TargetProfile:
    backend: str
    device: str
    interface: str
    speed_khz: int
    address_unit_bits: int
    endianness: str
    architecture: str = "generic"
    core: str = "default"
    allowed_memory_ranges: list[MemoryRange] = field(default_factory=list)
    serial_number: int | None = None

    @classmethod
    def fake_default(cls) -> "TargetProfile":
        return cls(
            backend="jlink",
            device="FAKE-JLINK-TARGET",
            interface="swd",
            speed_khz=4000,
            address_unit_bits=8,
            endianness="little",
            architecture="generic",
            core="default",
            allowed_memory_ranges=[
                MemoryRange(start=0x20000000, length=0x10000, space="data"),
                MemoryRange(start=0x08000000, length=0x100000, space="program"),
            ],
        )

    @classmethod
    def generic_default(cls) -> "TargetProfile":
        return cls(
            backend="jlink",
            device=os.environ.get("AI_DEBUG_JLINK_DEVICE", ""),
            interface=os.environ.get("AI_DEBUG_JLINK_INTERFACE", "swd"),
            speed_khz=int(os.environ.get("AI_DEBUG_JLINK_SPEED_KHZ", "4000")),
            address_unit_bits=8,
            endianness="little",
            architecture="generic",
            core="default",
            allowed_memory_ranges=[MemoryRange(start=0x20000000, length=0x10000, space="data")],
        )

    @classmethod
    def c2000_c28x_default(cls, *, device: str, core: str) -> "TargetProfile":
        return cls(
            backend="jlink",
            device=device,
            interface="jtag",
            speed_khz=1000,
            address_unit_bits=16,
            endianness="little",
            architecture="c28x",
            core=core,
            allowed_memory_ranges=[
                MemoryRange(start=0x00008000, length=0x8000, space="data"),
                MemoryRange(start=0x00080000, length=0x80000, space="program"),
            ],
        )


class JLinkBackend:
    def __init__(
        self,
        *,
        probe_module: Any | None = None,
        profile: TargetProfile | None = None,
        missing_dependency: str = "",
    ) -> None:
        self._probe_module = probe_module
        if profile is not None:
            self._profile = profile
        elif os.environ.get("AI_DEBUG_JLINK_FAKE") == "1":
            self._profile = TargetProfile.fake_default()
        else:
            self._profile = TargetProfile.generic_default()
        self._missing_dependency = missing_dependency

    @classmethod
    def from_optional_dependency(
        cls,
        *,
        profile: TargetProfile | None = None,
        importer: Callable[[str], Any] = importlib.import_module,
    ) -> "JLinkBackend":
        if os.environ.get("AI_DEBUG_JLINK_FAKE") == "1":
            return cls(probe_module=FakePylinkModule, profile=profile or TargetProfile.fake_default())
        try:
            pylink = importer("pylink")
        except ModuleNotFoundError as exc:
            return cls(probe_module=None, profile=profile, missing_dependency=str(exc))
        return cls(probe_module=pylink, profile=profile)

    def capabilities(self) -> Capabilities:
        return Capabilities(
            artifact_load=False,
            memory_read=True,
            memory_write=False,
            variable_read=False,
            telemetry_capture=False,
            fault_snapshot=False,
            flash=False,
        )

    def discover(self) -> OperationResult:
        if self._probe_module is None:
            return self._dependency_missing()
        try:
            probe = JLinkProbe(self._probe_module)
            devices = [{"backend": "jlink", **item} for item in probe.discover()]
            return OperationResult.ok_result({"devices": devices})
        except Exception as exc:
            return OperationResult.fail("IO_ERROR", str(exc))

    def validate(self) -> OperationResult:
        profile_error = self._profile_error()
        if profile_error is not None:
            return profile_error
        if self._probe_module is None:
            return self._dependency_missing()
        link = self._open_connected_link()
        if isinstance(link, OperationResult):
            return link
        try:
            target_id = link.core_id()
            return OperationResult.ok_result(
                {
                    "connected": True,
                    "target_identity": {
                        "backend": "jlink",
                        "device": self._profile.device,
                        "architecture": self._profile.architecture,
                        "core": self._profile.core,
                        "target_id": f"0x{target_id:08X}",
                    },
                }
            )
        finally:
            link.close()

    def read_memory(self, address: TargetAddress, octet_length: int) -> MemoryBlock:
        result = self.try_read_memory(address, octet_length)
        if not result.ok:
            raise ValueError(result.message)
        return result.data["block"]

    def try_read_memory(self, address: TargetAddress, octet_length: int) -> OperationResult:
        profile_error = self._profile_error()
        if profile_error is not None:
            return profile_error
        if self._probe_module is None:
            return self._dependency_missing()
        if not self._is_allowed_read(address, octet_length):
            return OperationResult.fail("POLICY_DENIED", "memory read is outside allowed target profile ranges")
        link = self._open_connected_link()
        if isinstance(link, OperationResult):
            return link
        try:
            raw = bytes(link.memory_read8(address.value, octet_length))
            return OperationResult.ok_result({"block": MemoryBlock(address=address, data=raw)})
        except Exception as exc:
            return OperationResult.fail("IO_ERROR", str(exc))
        finally:
            link.close()

    def try_read_register(self, name: str) -> OperationResult:
        profile_error = self._profile_error()
        if profile_error is not None:
            return profile_error
        if self._probe_module is None:
            return self._dependency_missing()
        link = self._open_connected_link()
        if isinstance(link, OperationResult):
            return link
        try:
            value = int(link.register_read(name.upper()))
            return OperationResult.ok_result({"name": name.upper(), "value": f"0x{value:08X}"})
        except Exception as exc:
            return OperationResult.fail("IO_ERROR", str(exc))
        finally:
            link.close()

    def _open_connected_link(self) -> Any | OperationResult:
        try:
            probe = JLinkProbe(self._probe_module)
            return probe.open_connected(
                serial_number=self._profile.serial_number,
                interface=self._profile.interface,
                speed_khz=self._profile.speed_khz,
                device=self._profile.device,
            )
        except Exception as exc:
            return OperationResult.fail("IO_ERROR", str(exc))

    def _is_allowed_read(self, address: TargetAddress, octet_length: int) -> bool:
        if address.address_unit_bits != self._profile.address_unit_bits:
            return False
        if octet_length <= 0:
            return False
        return any(memory_range.contains(address, octet_length) for memory_range in self._profile.allowed_memory_ranges)

    def _profile_error(self) -> OperationResult | None:
        if self._profile.device == "":
            return OperationResult.fail("PROFILE_INVALID", "J-Link target device is not configured")
        return None

    def _dependency_missing(self) -> OperationResult:
        message = self._missing_dependency or "pylink-square is not installed"
        return OperationResult.fail("DEPENDENCY_MISSING", message)
