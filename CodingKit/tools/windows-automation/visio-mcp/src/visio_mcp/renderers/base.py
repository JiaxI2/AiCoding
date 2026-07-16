from __future__ import annotations

from abc import ABC, abstractmethod
from pathlib import Path

from ..model import DiagramPlan


class Renderer(ABC):
    @abstractmethod
    def doctor(self) -> dict: ...

    @abstractmethod
    def render(self, plan: DiagramPlan, output: Path, visible: bool = False) -> dict: ...

    @abstractmethod
    def export(self, session_id: str, formats: list[str], output_dir: Path) -> dict: ...

    @abstractmethod
    def snapshot(self, session_id: str, output: Path) -> dict: ...

    @abstractmethod
    def inspect(self, session_id: str) -> dict: ...

    @abstractmethod
    def edit(self, session_id: str, operations: list[dict]) -> dict: ...

    @abstractmethod
    def close(self, session_id: str, save: bool = True) -> dict: ...
