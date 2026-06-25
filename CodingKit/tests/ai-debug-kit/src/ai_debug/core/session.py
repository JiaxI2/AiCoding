from __future__ import annotations

import json
import uuid
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any


def utc_now() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


@dataclass
class DebugSession:
    workspace: Path
    session_id: str
    created_at: str
    actions: list[dict[str, Any]] = field(default_factory=list)
    observations: list[dict[str, Any]] = field(default_factory=list)

    @classmethod
    def create(cls, workspace: Path) -> "DebugSession":
        return cls(
            workspace=workspace.resolve(),
            session_id="dbg-" + uuid.uuid4().hex[:12],
            created_at=utc_now(),
        )

    @property
    def bundle_dir(self) -> Path:
        return self.workspace / ".ai-debug" / "sessions" / self.session_id

    def record_action(
        self,
        *,
        operation: str,
        result_code: str,
        risk_level: str,
        approved: bool,
        details: dict[str, Any],
    ) -> None:
        self.actions.append(
            {
                "action_id": "act-" + f"{len(self.actions) + 1:03d}",
                "operation": operation,
                "requested_by": "codex",
                "risk_level": risk_level,
                "approved": approved,
                "started_at": utc_now(),
                "completed_at": utc_now(),
                "result_code": result_code,
                "side_effects": [],
                "details": details,
            }
        )

    def record_observation(self, *, kind: str, data: dict[str, Any]) -> None:
        self.observations.append(
            {
                "observation_id": "obs-" + f"{len(self.observations) + 1:03d}",
                "kind": kind,
                "recorded_at": utc_now(),
                "data": data,
            }
        )

    def export(self) -> Path:
        self.bundle_dir.mkdir(parents=True, exist_ok=True)
        manifest = {
            "schema_version": "1.0",
            "session_id": self.session_id,
            "created_at": self.created_at,
            "state": "COMPLETED",
        }
        (self.bundle_dir / "manifest.json").write_text(
            json.dumps(manifest, indent=2, ensure_ascii=False) + "\n",
            encoding="utf-8",
        )
        self._write_jsonl(self.bundle_dir / "actions.jsonl", self.actions)
        self._write_jsonl(self.bundle_dir / "observations.jsonl", self.observations)
        return self.bundle_dir

    @staticmethod
    def _write_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
        path.write_text(
            "".join(json.dumps(row, ensure_ascii=False) + "\n" for row in rows),
            encoding="utf-8",
        )
