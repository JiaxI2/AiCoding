from __future__ import annotations

import json
from pathlib import Path

from jsonschema import Draft202012Validator

from .errors import ValidationError
from .styling import (
    resolve_document_style,
    resolve_style_token,
    resolve_text_role,
)


SCHEMA_PATH = Path(__file__).resolve().parents[2] / "schemas" / "diagram.schema.json"


def _schema() -> dict:
    return json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))


def validate_diagram(data: dict) -> dict:
    validator = Draft202012Validator(_schema())
    errors = sorted(validator.iter_errors(data), key=lambda error: list(error.path))
    if errors:
        detail = [{"path": "/".join(map(str, error.path)), "message": error.message} for error in errors]
        raise ValidationError(json.dumps(detail, ensure_ascii=False))

    node_ids = [node["id"] for node in data["nodes"]]
    if len(node_ids) != len(set(node_ids)):
        raise ValidationError("Duplicate node id")
    edge_ids = [edge["id"] for edge in data["edges"]]
    if len(edge_ids) != len(set(edge_ids)):
        raise ValidationError("Duplicate edge id")
    known = set(node_ids)
    broken = [
        edge["id"]
        for edge in data["edges"]
        if edge["from"] not in known or edge["to"] not in known
    ]
    if broken:
        raise ValidationError(f"Edges reference missing nodes: {broken}")

    try:
        style_profile = resolve_document_style(data["document"])
        for node in data["nodes"]:
            node_type = node.get("type", "process")
            default_role = {
                "group": "groupTitle",
                "junction": "operator",
                "note": "note",
            }.get(node_type, "body")
            resolve_text_role(
                style_profile,
                str(node.get("fontRole", default_role)),
            )
            resolve_style_token(
                style_profile,
                "nodeStyles",
                node.get("style"),
            )
        for edge in data["edges"]:
            resolve_text_role(
                style_profile,
                str(edge.get("fontRole", "edgeLabel")),
            )
            resolve_style_token(
                style_profile,
                "edgeStyles",
                edge.get("style"),
            )
    except ValueError as exc:
        raise ValidationError(str(exc)) from exc

    return {
        "valid": True,
        "nodeCount": len(node_ids),
        "edgeCount": len(edge_ids),
        "warnings": semantic_warnings(data),
    }


def semantic_warnings(data: dict) -> list[dict]:
    warnings: list[dict] = []
    for node in data["nodes"]:
        if len(node["text"]) > 120:
            warnings.append({"code": "LONG_TEXT", "node": node["id"], "length": len(node["text"])})
    isolated = {node["id"] for node in data["nodes"]}
    for edge in data["edges"]:
        isolated.discard(edge["from"])
        isolated.discard(edge["to"])
        if edge["from"] == edge["to"] and edge.get("routing", "orthogonal") == "straight":
            warnings.append({"code": "STRAIGHT_SELF_LOOP", "edge": edge["id"]})
    for node_id in sorted(isolated):
        warnings.append({"code": "ISOLATED_NODE", "node": node_id})
    return warnings
