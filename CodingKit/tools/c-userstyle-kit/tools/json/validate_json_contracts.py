#!/usr/bin/env python3
"""Validate kit JSON documents against the checked-in JSON Schema subset in use."""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[2]


class ValidationError(ValueError):
    """Report one schema validation failure with a JSON path."""


def no_duplicate_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise ValidationError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"), object_pairs_hook=no_duplicate_object)


def resolve_pointer(document: Any, pointer: str) -> Any:
    if not pointer.startswith("#/"):
        raise ValidationError(f"only local schema references are supported: {pointer}")
    current = document
    for raw_part in pointer[2:].split("/"):
        part = raw_part.replace("~1", "/").replace("~0", "~")
        current = current[part]
    return current


def matches_type(instance: Any, expected: str) -> bool:
    if expected == "object":
        return isinstance(instance, dict)
    if expected == "array":
        return isinstance(instance, list)
    if expected == "string":
        return isinstance(instance, str)
    if expected == "boolean":
        return isinstance(instance, bool)
    if expected == "integer":
        return isinstance(instance, int) and not isinstance(instance, bool)
    raise ValidationError(f"unsupported schema type: {expected}")


def validate(instance: Any, schema: Any, root_schema: dict[str, Any], path: str = "$") -> None:
    if schema is False:
        raise ValidationError(f"{path}: additional item is not allowed")
    if schema is True:
        return
    if "$ref" in schema:
        validate(instance, resolve_pointer(root_schema, schema["$ref"]), root_schema, path)
        return
    if "oneOf" in schema:
        matches = 0
        for option in schema["oneOf"]:
            try:
                validate(instance, option, root_schema, path)
            except ValidationError:
                continue
            matches += 1
        if matches != 1:
            raise ValidationError(f"{path}: expected exactly one oneOf schema match, got {matches}")
        return
    if "type" in schema and not matches_type(instance, schema["type"]):
        raise ValidationError(f"{path}: expected {schema['type']}, got {type(instance).__name__}")
    if "const" in schema and instance != schema["const"]:
        raise ValidationError(f"{path}: expected constant {schema['const']!r}")
    if "enum" in schema and instance not in schema["enum"]:
        raise ValidationError(f"{path}: value {instance!r} is outside enum")

    if isinstance(instance, str):
        if len(instance) < schema.get("minLength", 0):
            raise ValidationError(f"{path}: string is shorter than minLength")
        if "pattern" in schema and re.search(schema["pattern"], instance) is None:
            raise ValidationError(f"{path}: string does not match {schema['pattern']!r}")

    if isinstance(instance, int) and not isinstance(instance, bool):
        if "minimum" in schema and instance < schema["minimum"]:
            raise ValidationError(f"{path}: value is below minimum")
        if "maximum" in schema and instance > schema["maximum"]:
            raise ValidationError(f"{path}: value is above maximum")

    if isinstance(instance, dict):
        if len(instance) < schema.get("minProperties", 0):
            raise ValidationError(f"{path}: object has fewer properties than minProperties")
        for required in schema.get("required", []):
            if required not in instance:
                raise ValidationError(f"{path}: missing required property {required!r}")
        properties = schema.get("properties", {})
        extra = sorted(set(instance) - set(properties))
        additional = schema.get("additionalProperties", True)
        if additional is False:
            if extra:
                raise ValidationError(f"{path}: additional properties are not allowed: {extra}")
        elif isinstance(additional, dict):
            for key in extra:
                validate(instance[key], additional, root_schema, f"{path}.{key}")
        for key, child_schema in properties.items():
            if key in instance:
                validate(instance[key], child_schema, root_schema, f"{path}.{key}")

    if isinstance(instance, list):
        if len(instance) < schema.get("minItems", 0):
            raise ValidationError(f"{path}: array is shorter than minItems")
        if "maxItems" in schema and len(instance) > schema["maxItems"]:
            raise ValidationError(f"{path}: array is longer than maxItems")
        if schema.get("uniqueItems"):
            normalized = [json.dumps(item, ensure_ascii=False, sort_keys=True) for item in instance]
            if len(normalized) != len(set(normalized)):
                raise ValidationError(f"{path}: array items are not unique")
        prefix_items = schema.get("prefixItems", [])
        for index, child_schema in enumerate(prefix_items):
            if index < len(instance):
                validate(instance[index], child_schema, root_schema, f"{path}[{index}]")
        if "items" in schema:
            start = len(prefix_items) if prefix_items else 0
            for index in range(start, len(instance)):
                validate(instance[index], schema["items"], root_schema, f"{path}[{index}]")


def validate_pair(document_path: Path, schema_path: Path) -> Any:
    document = load_json(document_path)
    schema = load_json(schema_path)
    if not isinstance(schema, dict):
        raise ValidationError(f"{schema_path}: schema root must be an object")
    validate(document, schema, schema)
    return document


def main() -> int:
    config_path = ROOT / "examples" / "c-kit.json"
    embedded_config_path = ROOT / "internal" / "cuserstyle" / "templates" / "c-kit.json"
    config_schema = ROOT / "config" / "skills" / "c99-standard-c" / "c-kit.schema.json"
    snippets_path = ROOT / "examples" / "c-snippets.json"
    embedded_snippets_path = ROOT / "internal" / "cuserstyle" / "templates" / "c-snippets.json"
    snippets_schema = ROOT / "config" / "skills" / "c99-standard-c" / "c-snippets.schema.json"
    catalog_path = ROOT / "config" / "rules" / "huawei-c-language-programming-standard.rules.json"
    catalog_schema = (
        ROOT / "config" / "skills" / "c99-standard-c" / "huawei-c-rule-catalog.schema.json"
    )
    verify_target_path = ROOT / "examples" / "verify-target.json"
    verify_target_schema = (
        ROOT / "config" / "skills" / "c99-standard-c" / "verify-target.schema.json"
    )

    config = validate_pair(config_path, config_schema)
    embedded_config = validate_pair(embedded_config_path, config_schema)
    snippets = validate_pair(snippets_path, snippets_schema)
    embedded_snippets = validate_pair(embedded_snippets_path, snippets_schema)
    catalog = validate_pair(catalog_path, catalog_schema)
    validate_pair(verify_target_path, verify_target_schema)

    if embedded_config != config:
        raise ValidationError("embedded init config and examples/c-kit.json are not identical")
    if embedded_snippets != snippets:
        raise ValidationError("embedded init snippets and examples/c-snippets.json are not identical")

    required_gcc_flags = {
        "-std=c99",
        "-pedantic-errors",
        "-Werror",
        "-Wconversion",
        "-Wsign-conversion",
        "-Wshadow",
        "-Wmissing-prototypes",
        "-Wstrict-prototypes",
        "-Wvla",
        "-Wformat=2",
    }
    for compiler in ("gcc", "clang"):
        actual = set(config["gates"][compiler]["flags"])
        missing = sorted(required_gcc_flags - actual)
        if missing:
            raise ValidationError(f"$.gates.{compiler}.flags: missing strict flags {missing}")

    if config["reference"]["expectedClauses"] != catalog["summary"]["actualClauses"]:
        raise ValidationError("config and catalog clause counts disagree")
    if config["reference"]["sha256"] != catalog["source"]["sha256"]:
        raise ValidationError("config and catalog PDF hashes disagree")
    if Path(config["reference"]["ruleCatalog"]).as_posix() != catalog_path.relative_to(ROOT).as_posix():
        raise ValidationError("config ruleCatalog path does not point to the validated catalog")

    print(
        "JSON schema validation passed: c-kit, snippets, verify target, "
        "and 139-clause rule catalog"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
