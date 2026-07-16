from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from mcp.server.fastmcp import FastMCP

from .service import VisioService
from .styling import read_style_profiles_text


PROTOCOL_VERSION = "2025-11-25"


def _annotations(
    *,
    title: str,
    read_only: bool,
    destructive: bool,
    idempotent: bool,
    open_world: bool,
) -> dict[str, Any]:
    return {
        "title": title,
        "readOnlyHint": read_only,
        "destructiveHint": destructive,
        "idempotentHint": idempotent,
        "openWorldHint": open_world,
    }


def create_server(service: VisioService | None = None) -> FastMCP:
    active = service or VisioService()
    server = FastMCP(
        "visio_mcp",
        instructions="Validate and plan first. Require explicit user approval before visible edits or file-producing tools.",
    )

    @server.tool(
        name="visio_doctor",
        annotations=_annotations(
            title="Check Visio MCP",
            read_only=True,
            destructive=False,
            idempotent=True,
            open_world=True,
        ),
    )
    async def visio_doctor() -> dict:
        """Check platform, renderer availability and allowed output roots."""
        return active.doctor()

    @server.tool(
        name="diagram_validate",
        annotations=_annotations(
            title="Validate Diagram IR",
            read_only=True,
            destructive=False,
            idempotent=True,
            open_world=False,
        ),
    )
    async def diagram_validate(diagram: dict[str, Any]) -> dict:
        """Validate Diagram IR schema, IDs, references and semantic warnings."""
        return active.validate(diagram)

    @server.tool(
        name="diagram_plan",
        annotations=_annotations(
            title="Plan Diagram Layout",
            read_only=True,
            destructive=False,
            idempotent=True,
            open_world=False,
        ),
    )
    async def diagram_plan(diagram: dict[str, Any]) -> dict:
        """Compute deterministic layout and structural quality without opening Visio."""
        return active.plan(diagram)

    @server.tool(
        name="diagram_render",
        annotations=_annotations(
            title="Render Diagram",
            read_only=False,
            destructive=False,
            idempotent=False,
            open_world=True,
        ),
    )
    async def diagram_render(
        diagram: dict[str, Any],
        output: str,
        visible: bool = False,
        autoRepair: bool = True,
    ) -> dict:
        """Render Diagram IR to VSDX or a mock artifact."""
        return active.render(diagram, output, visible, autoRepair)

    @server.tool(
        name="diagram_open_visible",
        annotations=_annotations(
            title="Open Diagram in Visio",
            read_only=False,
            destructive=False,
            idempotent=False,
            open_world=True,
        ),
    )
    async def diagram_open_visible(diagram: dict[str, Any], output: str) -> dict:
        """Render and open a visible Visio window for direct user validation."""
        return active.render(diagram, output, True, True)

    @server.tool(
        name="diagram_snapshot",
        annotations=_annotations(
            title="Snapshot Diagram",
            read_only=False,
            destructive=False,
            idempotent=False,
            open_world=True,
        ),
    )
    async def diagram_snapshot(sessionId: str, output: str) -> dict:
        """Export the current page to PNG and run image quality checks."""
        return active.snapshot(sessionId, output)

    @server.tool(
        name="diagram_inspect",
        annotations=_annotations(
            title="Inspect Diagram",
            read_only=True,
            destructive=False,
            idempotent=True,
            open_world=True,
        ),
    )
    async def diagram_inspect(sessionId: str) -> dict:
        """Read actual shape text, position and dimensions from the active document."""
        return active.inspect(sessionId)

    @server.tool(
        name="diagram_edit",
        annotations=_annotations(
            title="Edit Diagram",
            read_only=False,
            destructive=True,
            idempotent=False,
            open_world=True,
        ),
    )
    async def diagram_edit(sessionId: str, operations: list[dict[str, Any]]) -> dict:
        """Apply explicit move, resize, set_text or select operations."""
        return active.edit(sessionId, operations)

    @server.tool(
        name="diagram_quality_check",
        annotations=_annotations(
            title="Check Diagram Quality",
            read_only=True,
            destructive=False,
            idempotent=True,
            open_world=True,
        ),
    )
    async def diagram_quality_check(sessionId: str | None = None, image: str | None = None) -> dict:
        """Evaluate structural and rendered-image quality."""
        return active.quality_check(sessionId, image)

    @server.tool(
        name="diagram_auto_repair",
        annotations=_annotations(
            title="Repair Diagram",
            read_only=False,
            destructive=True,
            idempotent=False,
            open_world=True,
        ),
    )
    async def diagram_auto_repair(sessionId: str, apply: bool = True) -> dict:
        """Run bounded repair and optionally apply edits to the live document."""
        return active.repair(sessionId, apply)

    @server.tool(
        name="diagram_export",
        annotations=_annotations(
            title="Export Diagram",
            read_only=False,
            destructive=False,
            idempotent=False,
            open_world=True,
        ),
    )
    async def diagram_export(sessionId: str, formats: list[str], outputDir: str) -> dict:
        """Export the active session to png, svg, pdf or vsdx."""
        return active.export(sessionId, formats, outputDir)

    @server.tool(
        name="diagram_close",
        annotations=_annotations(
            title="Close Diagram",
            read_only=False,
            destructive=False,
            idempotent=False,
            open_world=True,
        ),
    )
    async def diagram_close(sessionId: str, save: bool = True) -> dict:
        """Close a live Visio session and optionally save changes."""
        return active.close(sessionId, save)

    @server.resource(
        "visio://schemas/diagram",
        name="Diagram IR schema",
        mime_type="application/schema+json",
    )
    def diagram_schema() -> str:
        return (Path(__file__).resolve().parents[2] / "schemas" / "diagram.schema.json").read_text(encoding="utf-8")

    @server.resource(
        "visio://schemas/renderer-effective-fields",
        name="Renderer-effective Diagram IR fields",
        mime_type="application/json",
    )
    def renderer_effective_fields() -> str:
        return (
            Path(__file__).resolve().parents[2]
            / "schemas"
            / "renderer-effective-fields.json"
        ).read_text(encoding="utf-8")

    @server.resource(
        "visio://schemas/style-profile",
        name="Style profile registry schema",
        mime_type="application/schema+json",
    )
    def style_profile_schema() -> str:
        return (
            Path(__file__).resolve().parents[2]
            / "schemas"
            / "style-profile.schema.json"
        ).read_text(encoding="utf-8")

    @server.resource(
        "visio://styles/profiles",
        name="Active Visio style profiles",
        mime_type="application/json",
    )
    def style_profiles() -> str:
        return read_style_profiles_text()

    return server


def run_stdio(service: VisioService | None = None) -> None:
    active = service or VisioService()
    try:
        create_server(active).run(transport="stdio")
    finally:
        active.close_all()
