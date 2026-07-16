from pathlib import Path

import pytest

from visio_mcp.renderers.visio_com import VisioComRenderer


class FakeComError(Exception):
    def __init__(self, hresult: int):
        super().__init__(str(hresult))
        self.hresult = hresult


class FakePythonCom:
    def __init__(self):
        self.pumped = 0
        self.uninitialized = False

    def PumpWaitingMessages(self):
        self.pumped += 1

    def CoUninitialize(self):
        self.uninitialized = True


class FakeDocument:
    def __init__(self, calls, save_as_failures=0, save_error=None):
        self.calls = calls
        self.save_as_failures = save_as_failures
        self.save_error = save_error

    def SaveAs(self, target):
        self.calls.append(("vsdx", Path(target).suffix))
        if self.save_as_failures:
            self.save_as_failures -= 1
            raise FakeComError(VisioComRenderer._RPC_E_CALL_REJECTED)

    def ExportAsFixedFormat(self, *_args):
        self.calls.append(("pdf", Path(_args[1]).suffix))

    def Save(self):
        self.calls.append(("save", None))
        if self.save_error is not None:
            raise self.save_error

    def Close(self):
        self.calls.append(("close", None))


class FakePage:
    def __init__(self, calls):
        self.calls = calls

    def Export(self, target):
        self.calls.append(("image", Path(target).suffix))


class FakeApplication:
    def __init__(self, calls):
        self.calls = calls

    def Quit(self):
        self.calls.append(("quit", None))


def make_session(calls, **document_kwargs):
    pythoncom = FakePythonCom()
    return {
        "app": FakeApplication(calls),
        "doc": FakeDocument(calls, **document_kwargs),
        "page": FakePage(calls),
        "output": "diagram.vsdx",
        "visible": False,
        "pythoncom": pythoncom,
    }


def test_export_saves_vsdx_first_and_pdf_last(tmp_path):
    renderer = VisioComRenderer()
    calls = []
    renderer.sessions["session"] = make_session(calls)

    renderer.export("session", ["png", "pdf", "vsdx", "svg"], tmp_path)

    assert calls == [
        ("vsdx", ".vsdx"),
        ("image", ".png"),
        ("image", ".svg"),
        ("pdf", ".pdf"),
    ]


def test_export_retries_only_rejected_com_calls(tmp_path, monkeypatch):
    renderer = VisioComRenderer()
    calls = []
    session = make_session(calls, save_as_failures=2)
    renderer.sessions["session"] = session
    monkeypatch.setattr("visio_mcp.renderers.visio_com.time.sleep", lambda _seconds: None)

    renderer.export("session", ["vsdx"], tmp_path)

    assert session["pythoncom"].pumped == 2
    assert calls == [("vsdx", ".vsdx"), ("vsdx", ".vsdx"), ("vsdx", ".vsdx")]


def test_close_releases_visio_after_save_failure():
    renderer = VisioComRenderer()
    calls = []
    failure = RuntimeError("save failed")
    session = make_session(calls, save_error=failure)
    renderer.sessions["session"] = session

    with pytest.raises(RuntimeError, match="save failed"):
        renderer.close("session")

    assert calls == [("save", None), ("close", None), ("quit", None)]
    assert session["pythoncom"].uninitialized is True
    assert "session" not in renderer.sessions
