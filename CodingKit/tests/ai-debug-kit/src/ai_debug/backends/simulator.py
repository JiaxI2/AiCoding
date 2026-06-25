from __future__ import annotations

from ai_debug.backends.base import MemoryBlock
from ai_debug.core.address import TargetAddress
from ai_debug.core.capability import Capabilities
from ai_debug.core.policy import Approval, Policy, RiskLevel
from ai_debug.core.result import OperationResult


class SimulatorBackend:
    def __init__(self, size: int = 1024) -> None:
        if size <= 0:
            raise ValueError("size must be positive")
        self._memory = bytearray((index % 256 for index in range(size)))

    def discover(self) -> OperationResult:
        return OperationResult.ok_result({"devices": [{"backend": "simulator", "serial_number": "sim-001"}]})

    def validate(self) -> OperationResult:
        return OperationResult.ok_result(
            {"target_identity": {"backend": "simulator", "core_id": "simulator"}, "connected": True}
        )

    def capabilities(self) -> Capabilities:
        return Capabilities(
            artifact_load=True,
            memory_read=True,
            memory_write=True,
            variable_read=True,
            telemetry_capture=True,
            fault_snapshot=True,
            flash=False,
        )

    def read_memory(self, address: TargetAddress, octet_length: int) -> MemoryBlock:
        result = self.try_read_memory(address, octet_length)
        if not result.ok:
            raise ValueError(result.message)
        return result.data["block"]

    def try_read_memory(self, address: TargetAddress, octet_length: int) -> OperationResult:
        if not self._is_valid_range(address, octet_length):
            return OperationResult.fail("INVALID_ARGUMENT", "memory range is outside simulator memory")
        start = address.value
        end = start + octet_length
        return OperationResult.ok_result({"block": MemoryBlock(address=address, data=bytes(self._memory[start:end]))})

    def try_write_memory(
        self,
        address: TargetAddress,
        data: bytes,
        *,
        policy: Policy,
        approval: Approval,
    ) -> OperationResult:
        denied = policy.check(RiskLevel.L3, approval)
        if denied is not None:
            return denied
        if not self._is_valid_range(address, len(data)):
            return OperationResult.fail("INVALID_ARGUMENT", "memory range is outside simulator memory")
        start = address.value
        self._memory[start:start + len(data)] = data
        return OperationResult.ok_result({"octet_length": len(data)}, side_effects=["simulator_memory_modified"])

    def try_read_register(self, name: str) -> OperationResult:
        value = 0 if name.upper() == "R0" else 0
        return OperationResult.ok_result({"name": name.upper(), "value": f"0x{value:08X}"})

    def _is_valid_range(self, address: TargetAddress, octet_length: int) -> bool:
        if address.address_unit_bits != 8:
            return False
        if address.value < 0 or octet_length < 0:
            return False
        return address.value + octet_length <= len(self._memory)
