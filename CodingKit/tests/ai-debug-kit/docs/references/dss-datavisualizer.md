# DSS DataVisualizer Reference Notes

Upstream reference: https://github.com/DigitalAllianceStudio/DSS_DataVisualizer

Purpose for this kit:

- Treat DSS DataVisualizer as the reference direction for a future TI DSS/XDS backend.
- Keep J-Link integration probe-focused and architecture-neutral.
- Model C2000/C28x support through target profiles first, not through ARM-only assumptions.
- Preserve v0.2 safety boundaries: discovery, validation, capabilities, and read-only access only.

Observed design points from the reference:

- It targets Texas Instruments chip debug and visualization workflows.
- Its transport path is TI CCS DSS/DebugServer over XDS/JTAG, not J-Link.
- It covers TI DSP-oriented devices such as C2000/C28x and related XDS probes.
- Multicore targets should be modeled explicitly; the first backend contract should operate on one selected core at a time.

Implication:

The next transport after J-Link should be a separate `ti_dss` backend with DSS/XDS session ownership, target/core selection, expression/register read support, and explicit safety policy. It should share the same `DebugBackend` contract instead of adding C2000-specific behavior into `JLinkBackend`.