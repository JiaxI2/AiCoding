# Compile Smoke Test

CMake configure/build passed on Linux host with GCC. This is a syntax/link-level library smoke test only; it is not hardware validation.

```text
cmake -S . -B build
cmake --build build
BUILD_OK
```
