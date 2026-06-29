from __future__ import annotations

import argparse
from pathlib import Path

from . import __version__
from .core import (
    copy_examples,
    doctor,
    envelope,
    export_context,
    generate_report,
    loop_status,
    print_result,
    record_attempt,
    run_command_profile,
    validate_profile_file,
)


def add_output(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--output", choices=["text", "json"], default="text")


def main() -> int:
    parser = argparse.ArgumentParser(prog="airepair")
    sub = parser.add_subparsers(dest="command")

    p_version = sub.add_parser("version")
    add_output(p_version)

    p_doctor = sub.add_parser("doctor")
    add_output(p_doctor)

    p_init = sub.add_parser("init")
    p_init.add_argument("--workspace", default=".")
    add_output(p_init)

    p_profile = sub.add_parser("profile")
    profile_sub = p_profile.add_subparsers(dest="profile_command")
    p_profile_validate = profile_sub.add_parser("validate")
    p_profile_validate.add_argument("--profile", required=True)
    add_output(p_profile_validate)

    p_build = sub.add_parser("build")
    build_sub = p_build.add_subparsers(dest="build_command")
    p_build_run = build_sub.add_parser("run")
    p_build_run.add_argument("--profile", required=True)
    p_build_run.add_argument("--workspace", default=".")
    p_build_run.add_argument("--timeout", type=int)
    add_output(p_build_run)

    p_test = sub.add_parser("test")
    test_sub = p_test.add_subparsers(dest="test_command")
    p_test_run = test_sub.add_parser("run")
    p_test_run.add_argument("--profile", required=True)
    p_test_run.add_argument("--workspace", default=".")
    p_test_run.add_argument("--timeout", type=int)
    add_output(p_test_run)

    p_loop = sub.add_parser("loop")
    loop_sub = p_loop.add_subparsers(dest="loop_command")

    p_loop_status = loop_sub.add_parser("status")
    p_loop_status.add_argument("--profile", required=True)
    p_loop_status.add_argument("--workspace", default=".")
    add_output(p_loop_status)

    p_loop_export = loop_sub.add_parser("export-context")
    p_loop_export.add_argument("--profile", required=True)
    p_loop_export.add_argument("--workspace", default=".")
    add_output(p_loop_export)

    p_loop_record = loop_sub.add_parser("record-attempt")
    p_loop_record.add_argument("--profile", required=True)
    p_loop_record.add_argument("--workspace", default=".")
    p_loop_record.add_argument("--result", choices=["pass", "fail", "inconclusive"], required=True)
    p_loop_record.add_argument("--notes", default="")
    add_output(p_loop_record)

    p_report = sub.add_parser("report")
    report_sub = p_report.add_subparsers(dest="report_command")
    p_report_generate = report_sub.add_parser("generate")
    p_report_generate.add_argument("--workspace", default=".")
    add_output(p_report_generate)

    args = parser.parse_args()

    try:
        if args.command == "version":
            return print_result(envelope(True, "OK", "Version", {"version": __version__}), args.output)
        if args.command == "doctor":
            return print_result(envelope(True, "OK", "Doctor completed", doctor()), args.output)
        if args.command == "init":
            workspace = Path(args.workspace).resolve()
            copied = copy_examples(workspace)
            return print_result(envelope(True, "OK", "Profiles initialized", {"workspace": str(workspace), "copied": copied}), args.output)
        if args.command == "profile" and args.profile_command == "validate":
            return print_result(validate_profile_file(Path(args.profile).resolve()), args.output)
        if args.command == "build" and args.build_command == "run":
            return print_result(run_command_profile(Path(args.profile).resolve(), Path(args.workspace).resolve(), "build", args.timeout), args.output)
        if args.command == "test" and args.test_command == "run":
            return print_result(run_command_profile(Path(args.profile).resolve(), Path(args.workspace).resolve(), "test", args.timeout), args.output)
        if args.command == "loop" and args.loop_command == "status":
            return print_result(loop_status(Path(args.profile).resolve(), Path(args.workspace).resolve()), args.output)
        if args.command == "loop" and args.loop_command == "export-context":
            return print_result(export_context(Path(args.profile).resolve(), Path(args.workspace).resolve()), args.output)
        if args.command == "loop" and args.loop_command == "record-attempt":
            return print_result(record_attempt(Path(args.profile).resolve(), Path(args.workspace).resolve(), args.result, args.notes), args.output)
        if args.command == "report" and args.report_command == "generate":
            return print_result(generate_report(Path(args.workspace).resolve()), args.output)
        parser.print_help()
        return 2
    except Exception as exc:
        return print_result(envelope(False, "INTERNAL_ERROR", str(exc)), getattr(args, "output", "text"))


if __name__ == "__main__":
    raise SystemExit(main())
