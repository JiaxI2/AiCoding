import asyncio
import json
import os
from pathlib import Path
import subprocess
import sys

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

from visio_mcp.protocol import PROTOCOL_VERSION


async def protocol_scenario():
    environment = os.environ.copy()
    environment["PYTHONPATH"] = str((Path.cwd() / "src").resolve())
    parameters = StdioServerParameters(
        command=sys.executable,
        args=["-m", "visio_mcp", "--renderer", "mock", "server"],
        cwd=str(Path.cwd()),
        env=environment,
    )
    async with stdio_client(parameters) as (read, write):
        async with ClientSession(read, write) as session:
            initialized = await session.initialize()
            tools = await session.list_tools()
            resources = await session.list_resources()
            prompts = await session.list_prompts()
            if initialized.capabilities.logging is not None:
                await session.set_logging_level("info")
            invalid = await session.call_tool("diagram_validate", {"diagram": {}})
            await session.send_ping()
            return initialized, tools, resources, prompts, invalid


def test_official_client_lifecycle_and_annotations():
    initialized, tools, resources, prompts, invalid = asyncio.run(protocol_scenario())
    assert initialized.protocolVersion == PROTOCOL_VERSION
    assert len(tools.tools) == 12
    assert all(tool.annotations is not None for tool in tools.tools)
    assert {
        str(resource.uri)
        for resource in resources.resources
    } == {
        "visio://schemas/diagram",
        "visio://schemas/renderer-effective-fields",
        "visio://schemas/style-profile",
        "visio://styles/profiles",
    }
    assert len(prompts.prompts) == 0
    assert initialized.capabilities.logging is None
    assert invalid.isError is True


def test_invalid_json_does_not_break_following_ping():
    process = subprocess.Popen(
        [sys.executable, "-m", "visio_mcp", "--renderer", "mock", "server"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    )
    payload = (
        "{invalid-json}\n"
        + json.dumps(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": PROTOCOL_VERSION,
                    "capabilities": {},
                    "clientInfo": {"name": "raw-probe", "version": "1"},
                },
            }
        )
        + "\n"
    )
    stdout, _ = process.communicate(payload, timeout=20)
    assert process.returncode == 0
    assert any(json.loads(line).get("id") == 1 for line in stdout.splitlines() if line.startswith("{"))
