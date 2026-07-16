from __future__ import annotations

from pathlib import Path
import platform
import time

from .config import Settings
from .layout import plan_diagram
from .quality import connector_quality, image_quality, structural_quality
from .renderers.mock import MockRenderer
from .renderers.visio_com import VisioComRenderer
from .repair import auto_repair
from .styling import load_style_profiles, style_profiles_path
from .validation import validate_diagram


class VisioService:
    def __init__(self, settings: Settings | None = None, renderer: str = "auto"):
        self.settings = settings or Settings.load()
        if renderer == "mock" or (renderer == "auto" and platform.system() != "Windows"):
            self.renderer = MockRenderer()
        else:
            self.renderer = VisioComRenderer()
        self.session_plans = {}

    def doctor(self) -> dict:
        try:
            registry = load_style_profiles()
            style_profiles = {
                "available": True,
                "path": str(style_profiles_path()),
                "default": registry["defaultProfile"],
                "profiles": sorted(registry["profiles"]),
            }
        except Exception as exc:
            style_profiles = {
                "available": False,
                "path": str(style_profiles_path()),
                "error": str(exc),
            }
        return {
            "service": "visio-mcp",
            "platform": platform.platform(),
            "renderer": self.renderer.doctor(),
            "styleProfiles": style_profiles,
            "allowedOutputRoots": [str(path) for path in self.settings.allowed_output_roots],
        }

    def validate(self, data: dict) -> dict:
        return validate_diagram(data)

    def plan(self, data: dict) -> dict:
        validate_diagram(data)
        plan = plan_diagram(data)
        return {"plan": plan.to_dict(), "quality": structural_quality(plan)}

    def render(self, data: dict, output: str | Path, visible: bool = False, auto_repair_enabled: bool = True) -> dict:
        validate_diagram(data)
        plan = plan_diagram(data)
        repair_report = None
        if auto_repair_enabled:
            plan, repair_report = auto_repair(plan)
        target = self.settings.ensure_output_allowed(Path(output))
        started = time.perf_counter()
        result = self.renderer.render(plan, target, visible=visible)
        result["durationMs"] = round((time.perf_counter() - started) * 1000, 2)
        result["quality"] = structural_quality(plan)
        result["repair"] = repair_report
        self.session_plans[result["sessionId"]] = plan
        result["connectorQuality"] = connector_quality(
            plan,
            self.renderer.inspect(result["sessionId"]),
        )
        return result

    def snapshot(self, session_id: str, output: str | Path) -> dict:
        target = self.settings.ensure_output_allowed(Path(output))
        result = self.renderer.snapshot(session_id, target)
        result["imageQuality"] = image_quality(target)
        return result

    def export(self, session_id: str, formats: list[str], output_dir: str | Path) -> dict:
        root = self.settings.ensure_output_allowed(Path(output_dir) / ".probe").parent
        return self.renderer.export(session_id, formats, root)

    def inspect(self, session_id: str) -> dict:
        return self.renderer.inspect(session_id)

    def edit(self, session_id: str, operations: list[dict]) -> dict:
        return self.renderer.edit(session_id, operations)

    def quality_check(self, session_id: str | None = None, image: str | None = None) -> dict:
        output = {}
        if session_id:
            plan = self.session_plans[session_id]
            output["structure"] = structural_quality(plan)
            output["connectors"] = connector_quality(plan, self.renderer.inspect(session_id))
        if image:
            output["image"] = image_quality(image)
        scores = [item["score"] for item in output.values()]
        output["score"] = round(sum(scores) / len(scores), 2) if scores else 0
        return output

    def repair(self, session_id: str, apply: bool = True) -> dict:
        plan = self.session_plans[session_id]
        repaired, report = auto_repair(plan)
        if apply:
            before = {item.id: item for item in plan.nodes}
            operations = []
            for node in repaired.nodes:
                previous = before[node.id]
                if (node.x, node.y) != (previous.x, previous.y):
                    operations.append({"op": "move", "nodeId": node.id, "x": node.x, "y": node.y})
                if (node.width, node.height) != (previous.width, previous.height):
                    operations.append(
                        {"op": "resize", "nodeId": node.id, "width": node.width, "height": node.height}
                    )
            if operations:
                self.renderer.edit(session_id, operations)
            self.session_plans[session_id] = repaired
            report["appliedOperations"] = operations
        return report

    def close(self, session_id: str, save: bool = True) -> dict:
        self.session_plans.pop(session_id, None)
        return self.renderer.close(session_id, save)

    def close_all(self) -> None:
        for session_id in list(self.session_plans):
            try:
                self.close(session_id, save=True)
            except Exception:
                pass
