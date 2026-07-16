class VisioMcpError(Exception):
    code = "VISIO_MCP_ERROR"


class ValidationError(VisioMcpError):
    code = "VALIDATION_ERROR"


class PlatformError(VisioMcpError):
    code = "PLATFORM_ERROR"


class SecurityError(VisioMcpError):
    code = "SECURITY_ERROR"


class SessionError(VisioMcpError):
    code = "SESSION_ERROR"
