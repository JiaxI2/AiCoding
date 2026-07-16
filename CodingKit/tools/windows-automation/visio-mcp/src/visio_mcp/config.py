from __future__ import annotations

from dataclasses import dataclass
import os
from pathlib import Path


@dataclass(frozen=True)
class Settings:
    project_root: Path
    allowed_output_roots: tuple[Path, ...]
    max_nodes: int = 1000
    max_edges: int = 3000
    default_timeout_s: int = 120

    @staticmethod
    def load() -> "Settings":
        root = Path(os.environ.get("VISIO_MCP_ROOT", Path.cwd())).resolve()
        raw = os.environ.get(
            "VISIO_MCP_OUTPUT_ROOTS",
            "dist;test-results;generated",
        )
        roots = tuple((root / item.strip()).resolve() for item in raw.split(";") if item.strip())
        return Settings(project_root=root, allowed_output_roots=roots)

    def ensure_output_allowed(self, path: Path) -> Path:
        resolved = path.resolve()
        if not any(resolved == root or root in resolved.parents for root in self.allowed_output_roots):
            raise PermissionError(f"Output path is outside allowed roots: {resolved}")
        resolved.parent.mkdir(parents=True, exist_ok=True)
        return resolved
