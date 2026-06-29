from __future__ import annotations

import argparse
from pathlib import Path

from . import __version__
from .ti_dss import (
    dss_capabilities,
    dss_doctor,
    dss_profile_template,
    execute_dss_script,
    generate_dss_script,
    validate_dss_profile,
)
from .jlink_guard import (
    jlink_capabilities,
    jlink_invasive_operation,
    jlink_profile_template,
    validate_jlink_profile,
)

from .core import (
    copy_examples,
    doctor,
    envelope,
    export_context,
    generate_report,
    load_json,
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


    p_dss = sub.add_parser("dss")
    dss_sub = p_dss.add_subparsers(dest="dss_command")
    p_dss_template = dss_sub.add_parser("profile-template")
    p_dss_template.add_argument("--profile", required=True)
    add_output(p_dss_template)
    p_dss_validate = dss_sub.add_parser("validate-profile")
    p_dss_validate.add_argument("--profile", required=True)
    add_output(p_dss_validate)
    p_dss_doctor = dss_sub.add_parser("doctor")
    p_dss_doctor.add_argument("--profile")
    add_output(p_dss_doctor)
    p_dss_cap = dss_sub.add_parser("capabilities")
    p_dss_cap.add_argument("--profile")
    add_output(p_dss_cap)
    p_dss_read_expr = dss_sub.add_parser("read-expression")
    p_dss_read_expr.add_argument("--profile", required=True)
    p_dss_read_expr.add_argument("--expression", required=True)
    p_dss_read_expr.add_argument("--script-out", default=".ai-debug-repair/dss/read_expression.js")
    p_dss_read_expr.add_argument("--execute", action="store_true")
    p_dss_read_expr.add_argument("--timeout", type=int)
    add_output(p_dss_read_expr)
    p_dss_read_reg = dss_sub.add_parser("read-register")
    p_dss_read_reg.add_argument("--profile", required=True)
    p_dss_read_reg.add_argument("--register", required=True)
    p_dss_read_reg.add_argument("--script-out", default=".ai-debug-repair/dss/read_register.js")
    p_dss_read_reg.add_argument("--execute", action="store_true")
    p_dss_read_reg.add_argument("--timeout", type=int)
    add_output(p_dss_read_reg)

    p_jlink = sub.add_parser("jlink")
    jlink_sub = p_jlink.add_subparsers(dest="jlink_command")
    p_jlink_template = jlink_sub.add_parser("profile-template")
    p_jlink_template.add_argument("--profile", required=True)
    add_output(p_jlink_template)
    p_jlink_validate = jlink_sub.add_parser("validate-profile")
    p_jlink_validate.add_argument("--profile", required=True)
    add_output(p_jlink_validate)
    p_jlink_cap = jlink_sub.add_parser("capabilities")
    p_jlink_cap.add_argument("--profile")
    add_output(p_jlink_cap)
    for _name in ["reset", "halt", "flash", "write-memory"]:
        _p = jlink_sub.add_parser(_name)
        _p.add_argument("--profile", required=True)
        _p.add_argument("--approve", action="store_true")
        add_output(_p)

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

        if args.command == "dss" and args.dss_command == "profile-template":
            return print_result(dss_profile_template(Path(args.profile).resolve()), args.output)

        if args.command == "dss" and args.dss_command == "validate-profile":
            return print_result(validate_dss_profile(Path(args.profile).resolve()), args.output)

        if args.command == "dss" and args.dss_command == "doctor":
            profile = Path(args.profile).resolve() if args.profile else None
            return print_result(dss_doctor(profile), args.output)

        if args.command == "dss" and args.dss_command == "capabilities":
            if args.profile:
                profile_data = load_json(Path(args.profile).resolve())
                return print_result(dss_capabilities(profile_data), args.output)
            return print_result(dss_capabilities(), args.output)

        if args.command == "dss" and args.dss_command == "read-expression":
            script_path = Path(args.script_out).resolve()
            gen = generate_dss_script(Path(args.profile).resolve(), args.expression, None, script_path)
            if not gen["ok"] or not args.execute:
                return print_result(gen, args.output)
            return print_result(execute_dss_script(Path(args.profile).resolve(), script_path, args.timeout), args.output)

        if args.command == "dss" and args.dss_command == "read-register":
            script_path = Path(args.script_out).resolve()
            gen = generate_dss_script(Path(args.profile).resolve(), None, args.register, script_path)
            if not gen["ok"] or not args.execute:
                return print_result(gen, args.output)
            return print_result(execute_dss_script(Path(args.profile).resolve(), script_path, args.timeout), args.output)

        if args.command == "jlink" and args.jlink_command == "profile-template":
            return print_result(jlink_profile_template(Path(args.profile).resolve()), args.output)

        if args.command == "jlink" and args.jlink_command == "validate-profile":
            return print_result(validate_jlink_profile(Path(args.profile).resolve()), args.output)

        if args.command == "jlink" and args.jlink_command == "capabilities":
            if args.profile:
                profile_data = load_json(Path(args.profile).resolve())
                return print_result(jlink_capabilities(profile_data), args.output)
            return print_result(jlink_capabilities(), args.output)

        if args.command == "jlink" and args.jlink_command in {"reset", "halt", "flash", "write-memory"}:
            op = args.jlink_command.replace("-", "_")
            return print_result(jlink_invasive_operation(Path(args.profile).resolve(), op, args.approve), args.output)

        if args.command == "report" and args.report_command == "generate":
            return print_result(generate_report(Path(args.workspace).resolve()), args.output)
        parser.print_help()
        return 2
    except Exception as exc:
        return print_result(envelope(False, "INTERNAL_ERROR", str(exc)), getattr(args, "output", "text"))


if __name__ == "__main__":
    raise SystemExit(main())
