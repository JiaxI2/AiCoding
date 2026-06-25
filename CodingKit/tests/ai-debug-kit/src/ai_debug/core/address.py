from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class TargetAddress:
    space: str
    value: int
    address_unit_bits: int
