package domain

import "errors"

type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func (e *Error) WithDetails(d map[string]any) *Error {
	e.Details = d
	return e
}

const (
	CodeValidation             = "VALIDATION_ERROR"
	CodeUnauthenticated        = "UNAUTHENTICATED"
	CodeTokenExpired           = "TOKEN_EXPIRED"
	CodeFeatureDisabled        = "FEATURE_DISABLED"
	CodeInsufficientScope      = "INSUFFICIENT_SCOPE"
	CodeNotFound               = "NOT_FOUND"
	CodeRevisionConflict       = "REVISION_CONFLICT"
	CodeInvalidTransition      = "INVALID_TRANSITION"
	CodeApprovalRequired       = "APPROVAL_REQUIRED"
	CodeApprovalStale          = "APPROVAL_STALE"
	CodeUnsupportedClaim       = "UNSUPPORTED_CLAIM"
	CodeIntegrationRequired    = "INTEGRATION_REQUIRED"
	CodeIdempotencyConflict    = "IDEMPOTENCY_CONFLICT"
	CodePayloadTooLarge        = "PAYLOAD_TOO_LARGE"
	CodeVerificationFailed     = "VERIFICATION_FAILED"
	CodeDocumentNotFinalized   = "DOCUMENT_NOT_FINALIZED"
	CodeRateLimited            = "RATE_LIMITED"
	CodeCreditExhausted        = "CREDIT_EXHAUSTED"
	CodeProviderError          = "PROVIDER_ERROR"
	CodeNotConfigured          = "NOT_CONFIGURED"
	CodeTemporarilyUnavailable = "TEMPORARILY_UNAVAILABLE"
	CodeInternal               = "INTERNAL"
)

func Validation(msg string) *Error {
	return &Error{Status: 400, Code: CodeValidation, Message: msg}
}

func ValidationField(field, msg string) *Error {
	return &Error{Status: 400, Code: CodeValidation, Message: msg, Details: map[string]any{"field": field}}
}

func Unauthenticated(msg string) *Error {
	return &Error{Status: 401, Code: CodeUnauthenticated, Message: msg}
}

func TokenExpired() *Error {
	return &Error{Status: 401, Code: CodeTokenExpired, Message: "access token expired"}
}

func FeatureDisabled(msg string) *Error {
	return &Error{Status: 403, Code: CodeFeatureDisabled, Message: msg}
}

func InsufficientScope(msg string) *Error {
	return &Error{Status: 403, Code: CodeInsufficientScope, Message: msg}
}

func NotFound() *Error {
	return &Error{Status: 404, Code: CodeNotFound, Message: "resource not found"}
}

func RevisionConflict(current int64) *Error {
	return &Error{Status: 409, Code: CodeRevisionConflict, Message: "resource was modified", Details: map[string]any{"currentRevision": current}}
}

func InvalidTransition(msg string) *Error {
	return &Error{Status: 409, Code: CodeInvalidTransition, Message: msg}
}

func ApprovalRequired(msg string) *Error {
	return &Error{Status: 409, Code: CodeApprovalRequired, Message: msg}
}

func ApprovalStale(msg string) *Error {
	return &Error{Status: 409, Code: CodeApprovalStale, Message: msg}
}

func UnsupportedClaim(msg string) *Error {
	return &Error{Status: 409, Code: CodeUnsupportedClaim, Message: msg}
}

func IntegrationRequired(msg string) *Error {
	return &Error{Status: 409, Code: CodeIntegrationRequired, Message: msg}
}

func IdempotencyConflict() *Error {
	return &Error{Status: 409, Code: CodeIdempotencyConflict, Message: "idempotency key was used with a different request"}
}

func PayloadTooLarge() *Error {
	return &Error{Status: 413, Code: CodePayloadTooLarge, Message: "request body too large"}
}

func VerificationFailed(msg string) *Error {
	return &Error{Status: 422, Code: CodeVerificationFailed, Message: msg}
}

func DocumentNotFinalized(msg string) *Error {
	return &Error{Status: 422, Code: CodeDocumentNotFinalized, Message: msg}
}

func ProviderError(msg string) *Error {
	return &Error{Status: 502, Code: CodeProviderError, Message: msg}
}

func NotConfigured(msg string) *Error {
	return &Error{Status: 503, Code: CodeNotConfigured, Message: msg}
}

func TemporarilyUnavailable(msg string) *Error {
	return &Error{Status: 503, Code: CodeTemporarilyUnavailable, Message: msg}
}

func Internal() *Error {
	return &Error{Status: 500, Code: CodeInternal, Message: "internal error"}
}

var ErrNotFound = errors.New("not found")

type Conflict struct {
	Current int64
}

func (c *Conflict) Error() string { return "revision conflict" }

func AsConflict(err error) (*Conflict, bool) {
	var c *Conflict
	if errors.As(err, &c) {
		return c, true
	}
	return nil, false
}
