from __future__ import annotations

import argparse
import json
import sys
from dataclasses import asdict
from pathlib import Path
from typing import Any

from ai_debug import __version__
from ai_debug.app.services import doctor_summary, run_smoke_test
from ai_debug.backends.registry import backend_names, create_backend
from ai_debug.core.address import TargetAddress
from ai_debug.core.result import Envelope, OperationResult


def print_envelope(envelope: Envelope, output: str) -> None:
    data = asdict(envelope)
    if output == "json":
        print(json.dumps(data, ensure_ascii=False))
        return
    if envelope.ok:
        print(f"[OK] {envelope.message}")
    else:
        print(f"[FAIL] {envelope.message}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="ai-debug")
    sub = parser.add_subparsers(dest="command")

    version = sub.add_parser("version", help="Show kit version")
    version.add_argument("--output", choices=["text", "json"], default="text")

    doctor = sub.add_parser("doctor", help="Validate local CLI and simulator backend")
    doctor.add_argument("--output", choices=["text", "json"], default="text")

    smoke = sub.add_parser("smoke-test", help="Run simulator deployment smoke test")
    smoke.add_argument("--workspace", default=".")
    smoke.add_argument("--output", choices=["text", "json"], default="text")

    backend = sub.add_parser("backend", help="Backend discovery and validation")
    backend_sub = backend.add_subparsers(dest="backend_command")
    backend_list = backend_sub.add_parser("list", help="List registered backends")
    backend_list.add_argument("--output", choices=["text", "json"], default="text")
    for command_name in ("discover", "validate", "capabilities"):
        command = backend_sub.add_parser(command_name, help=f"Run backend {command_name}")
        command.add_argument("--backend", default="simulator")
        command.add_argument("--output", choices=["text", "json"], default="text")

    memory = sub.add_parser("memory", help="Memory operations")
    memory_sub = memory.add_subparsers(dest="memory_command")
    memory_read = memory_sub.add_parser("read", help="Read target memory")
    memory_read.add_argument("address")
    memory_read.add_argument("length", type=int)
    memory_read.add_argument("--backend", default="simulator")
    memory_read.add_argument("--space", default="data")
    memory_read.add_argument("--output", choices=["text", "json"], default="text")

    register = sub.add_parser("register", help="Register operations")
    register_sub = register.add_subparsers(dest="register_command")
    register_read = register_sub.add_parser("read", help="Read target register")
    register_read.add_argument("name")
    register_read.add_argument("--backend", default="simulator")
    register_read.add_argument("--output", choices=["text", "json"], default="text")

    return parser


def ok(data: dict[str, Any], message: str = "Operation completed") -> Envelope:
    return Envelope(ok=True, code="OK", message=message, data=data)


def fail(code: str, message: str, data: dict[str, Any] | None = None) -> Envelope:
    return Envelope(ok=False, code=code, message=message, data=data or {})


def result_envelope(result: OperationResult, message: str = "Operation completed") -> Envelope:
    data = _jsonable_data(result.data)
    return Envelope(
        ok=result.ok,
        code=result.code,
        message=message if result.ok else result.message,
        data=data,
        warnings=result.warnings,
        side_effects=result.side_effects,
    )


def _jsonable_data(data: dict[str, Any]) -> dict[str, Any]:
    converted = dict(data)
    block = converted.pop("block", None)
    if block is not None:
        converted.update(
            {
                "address": block.address.value,
                "space": block.address.space,
                "address_unit_bits": block.address.address_unit_bits,
                "octet_length": block.octet_length,
                "raw_octets": block.data.hex(),
            }
        )
    return converted


def _parse_address(value: str) -> int:
    return int(value, 0)


def _load_backend(name: str):
    try:
        return create_backend(name)
    except ValueError as exc:
        return None, fail("INVALID_ARGUMENT", str(exc))


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.command is None:
        parser.print_help()
        return 0

    if args.command == "version":
        print_envelope(
            ok({"name": "ai-debug-kit", "version": __version__}, "ai-debug-kit version"),
            args.output,
        )
        return 0

    if args.command == "doctor":
        print_envelope(ok(doctor_summary(), "doctor completed"), args.output)
        return 0

    if args.command == "smoke-test":
        print_envelope(ok(run_smoke_test(Path(args.workspace)), "smoke-test completed"), args.output)
        return 0

    if args.command == "backend":
        if args.backend_command == "list":
            print_envelope(ok({"backends": backend_names()}, "backend list completed"), args.output)
            return 0
        backend_or_none = _load_backend(args.backend)
        if isinstance(backend_or_none, tuple):
            _, envelope = backend_or_none
            print_envelope(envelope, args.output)
            return 2
        backend = backend_or_none
        if args.backend_command == "discover":
            envelope = result_envelope(backend.discover(), "backend discover completed")
        elif args.backend_command == "validate":
            envelope = result_envelope(backend.validate(), "backend validate completed")
        elif args.backend_command == "capabilities":
            envelope = ok(asdict(backend.capabilities()), "backend capabilities completed")
        else:
            envelope = fail("INVALID_ARGUMENT", "backend subcommand is required")
        print_envelope(envelope, args.output)
        return 0 if envelope.ok else 3

    if args.command == "memory":
        if args.memory_command != "read":
            print_envelope(fail("INVALID_ARGUMENT", "memory subcommand is required"), "json")
            return 2
        backend_or_none = _load_backend(args.backend)
        if isinstance(backend_or_none, tuple):
            _, envelope = backend_or_none
            print_envelope(envelope, args.output)
            return 2
        address = TargetAddress(space=args.space, value=_parse_address(args.address), address_unit_bits=8)
        envelope = result_envelope(
            backend_or_none.try_read_memory(address, args.length),
            "memory read completed",
        )
        print_envelope(envelope, args.output)
        return 0 if envelope.ok else 3

    if args.command == "register":
        if args.register_command != "read":
            print_envelope(fail("INVALID_ARGUMENT", "register subcommand is required"), "json")
            return 2
        backend_or_none = _load_backend(args.backend)
        if isinstance(backend_or_none, tuple):
            _, envelope = backend_or_none
            print_envelope(envelope, args.output)
            return 2
        envelope = result_envelope(
            backend_or_none.try_read_register(args.name),
            "register read completed",
        )
        print_envelope(envelope, args.output)
        return 0 if envelope.ok else 3

    print(f"unknown command: {args.command}", file=sys.stderr)
    return 2
