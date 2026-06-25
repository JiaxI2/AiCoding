from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
HEADER = ROOT / "src" / "ai_debug" / "target_shim" / "include" / "ai_debug_target_shim.h"
SOURCE = ROOT / "src" / "ai_debug" / "target_shim" / "src" / "ai_debug_target_shim.c"


def test_target_shim_exposes_minimal_api() -> None:
    header = HEADER.read_text(encoding="utf-8")

    assert "AiDebug_Init" in header
    assert "AiDebug_PushSampleU32" in header
    assert "AiDebug_PublishSnapshot" in header
    assert "AiDebug_Service" in header
    assert "AI_DEBUG_ENABLE" in header


def test_target_shim_has_no_blocking_or_dynamic_runtime_patterns() -> None:
    combined = HEADER.read_text(encoding="utf-8") + "\n" + SOURCE.read_text(encoding="utf-8")

    banned = [
        "malloc",
        "free",
        "printf",
        "sprintf",
        "fprintf",
        "while (1)",
        "for (;;)",
    ]
    for token in banned:
        assert token not in combined


def test_target_shim_uses_drop_newest_overflow_counter() -> None:
    source = SOURCE.read_text(encoding="utf-8")

    assert "dropped_count" in source
    assert "return AI_DEBUG_ERR_FULL" in source
    assert "write_index" in source
    assert "read_index" in source
