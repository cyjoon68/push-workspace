from __future__ import annotations
VALIDATION = "VALIDATION_ERROR"
UNAUTHENTICATED = "UNAUTHENTICATED"
TOKEN_EXPIRED = "TOKEN_EXPIRED"
FEATURE_DISABLED = "FEATURE_DISABLED"
INSUFFICIENT_SCOPE = "INSUFFICIENT_SCOPE"
NOT_FOUND = "NOT_FOUND"
REVISION_CONFLICT = "REVISION_CONFLICT"
INVALID_TRANSITION = "INVALID_TRANSITION"
APPROVAL_REQUIRED = "APPROVAL_REQUIRED"
APPROVAL_STALE = "APPROVAL_STALE"
UNSUPPORTED_CLAIM = "UNSUPPORTED_CLAIM"
INTEGRATION_REQUIRED = "INTEGRATION_REQUIRED"
IDEMPOTENCY_CONFLICT = "IDEMPOTENCY_CONFLICT"
PAYLOAD_TOO_LARGE = "PAYLOAD_TOO_LARGE"
VERIFICATION_FAILED = "VERIFICATION_FAILED"
DOCUMENT_NOT_FINALIZED = "DOCUMENT_NOT_FINALIZED"
RATE_LIMITED = "RATE_LIMITED"
CREDIT_EXHAUSTED = "CREDIT_EXHAUSTED"
PROVIDER_ERROR = "PROVIDER_ERROR"
NOT_CONFIGURED = "NOT_CONFIGURED"
TEMPORARILY_UNAVAILABLE = "TEMPORARILY_UNAVAILABLE"
INTERNAL = "INTERNAL"


class DomainError(Exception):
    def __init__(self, status: int, code: str, message: str, details: dict | None = None):
        super().__init__(f"{code}: {message}")
        self.status = status
        self.code = code
        self.message = message
        self.details = details


def validation(msg: str) -> DomainError:
    return DomainError(400, VALIDATION, msg)


def validation_field(field: str, msg: str) -> DomainError:
    return DomainError(400, VALIDATION, msg, {"field": field})


def unauthenticated(msg: str) -> DomainError:
    return DomainError(401, UNAUTHENTICATED, msg)


def token_expired() -> DomainError:
    return DomainError(401, TOKEN_EXPIRED, "access token expired")


def feature_disabled(msg: str) -> DomainError:
    return DomainError(403, FEATURE_DISABLED, msg)


def insufficient_scope(msg: str) -> DomainError:
    return DomainError(403, INSUFFICIENT_SCOPE, msg)


def not_found() -> DomainError:
    return DomainError(404, NOT_FOUND, "resource not found")


def revision_conflict(current: int) -> DomainError:
    return DomainError(409, REVISION_CONFLICT, "resource was modified", {"currentRevision": current})


def invalid_transition(msg: str) -> DomainError:
    return DomainError(409, INVALID_TRANSITION, msg)


def approval_required(msg: str) -> DomainError:
    return DomainError(409, APPROVAL_REQUIRED, msg)


def approval_stale(msg: str) -> DomainError:
    return DomainError(409, APPROVAL_STALE, msg)


def unsupported_claim(msg: str) -> DomainError:
    return DomainError(409, UNSUPPORTED_CLAIM, msg)


def integration_required(msg: str) -> DomainError:
    return DomainError(409, INTEGRATION_REQUIRED, msg)


def idempotency_conflict() -> DomainError:
    return DomainError(409, IDEMPOTENCY_CONFLICT, "idempotency key was used with a different request")


def payload_too_large() -> DomainError:
    return DomainError(413, PAYLOAD_TOO_LARGE, "request body too large")


def verification_failed(msg: str) -> DomainError:
    return DomainError(422, VERIFICATION_FAILED, msg)


def document_not_finalized(msg: str) -> DomainError:
    return DomainError(422, DOCUMENT_NOT_FINALIZED, msg)


def provider_error(msg: str) -> DomainError:
    return DomainError(502, PROVIDER_ERROR, msg)


def not_configured(msg: str) -> DomainError:
    return DomainError(503, NOT_CONFIGURED, msg)


def temporarily_unavailable(msg: str) -> DomainError:
    return DomainError(503, TEMPORARILY_UNAVAILABLE, msg)


def internal() -> DomainError:
    return DomainError(500, INTERNAL, "internal error")


def map_revision_err(err: Exception) -> Exception:
    from app.db import ConflictError, NotFoundError
    if isinstance(err, NotFoundError):
        return not_found()
    if isinstance(err, ConflictError):
        return revision_conflict(err.current)
    return err
