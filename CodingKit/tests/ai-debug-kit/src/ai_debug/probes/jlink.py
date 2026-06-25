from __future__ import annotations

from typing import Any


class FakeJLink:
    def __init__(self) -> None:
        self.serial_no = None

    def connected_emulators(self) -> list[dict[str, int]]:
        return [{"SerialNumber": 12345678}]

    def open(self, serial_no: int | None = None) -> None:
        self.serial_no = serial_no

    def set_tif(self, _interface: Any) -> None:
        return None

    def set_speed(self, _speed: int) -> None:
        return None

    def connect(self, _device: str, verbose: bool = False) -> None:
        return None

    def core_id(self) -> int:
        return 0x0BB11477

    def memory_read8(self, address: int, length: int) -> list[int]:
        return [(address + offset) & 0xFF for offset in range(length)]

    def register_read(self, name: str) -> int:
        return 0x12345678 if name.upper() in {"R0", "ACC"} else 0

    def close(self) -> None:
        return None


class FakePylinkModule:
    class enums:
        class JLinkInterfaces:
            SWD = 1
            JTAG = 0

    JLink = FakeJLink


class JLinkProbe:
    def __init__(self, pylink_module: Any) -> None:
        self._pylink = pylink_module

    def discover(self) -> list[dict[str, int]]:
        link = self._pylink.JLink()
        return [{"serial_number": int(item["SerialNumber"])} for item in link.connected_emulators()]

    def open_connected(self, *, serial_number: int | None, interface: str, speed_khz: int, device: str):
        link = self._pylink.JLink()
        link.open(serial_no=serial_number)
        link.set_tif(self._interface_value(interface))
        link.set_speed(speed_khz)
        link.connect(device, verbose=False)
        return link

    def _interface_value(self, interface: str) -> Any:
        values = self._pylink.enums.JLinkInterfaces
        if interface.lower() == "jtag":
            return values.JTAG
        return values.SWD
