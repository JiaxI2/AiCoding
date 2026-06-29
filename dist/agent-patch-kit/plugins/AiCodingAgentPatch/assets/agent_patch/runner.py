from __future__ import annotations

import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Sequence


@dataclass
class CmdResult:
    code: int
    stdout: str
    stderr: str
    cmd: list[str]

    @property
    def ok(self) -> bool:
        return self.code == 0

    def short_cmd(self) -> str:
        return " ".join(self.cmd)


def run(cmd: Sequence[str], cwd: str | Path | None = None, check: bool = False) -> CmdResult:
    proc = subprocess.run(
        list(cmd),
        cwd=str(cwd) if cwd else None,
        text=True,
        capture_output=True,
        encoding="utf-8",
        errors="replace",
    )
    res = CmdResult(proc.returncode, proc.stdout, proc.stderr, list(cmd))
    if check and proc.returncode != 0:
        raise RuntimeError(f"command failed ({proc.returncode}): {' '.join(cmd)}\n{proc.stderr}")
    return res
