package protocol

import "net/http"

type ErrorCode string

const (
	CodeUnauthorized        ErrorCode = "unauthorized"
	CodeInvalidRequest      ErrorCode = "invalid_request"
	CodeUnsupportedLanguage ErrorCode = "unsupported_language"
	CodePayloadTooLarge     ErrorCode = "payload_too_large"
	CodeRateLimited         ErrorCode = "rate_limited"
	CodeQueueFull           ErrorCode = "queue_full"
	CodeExpired             ErrorCode = "expired"
	CodeCancelled           ErrorCode = "cancelled"
	CodeWorkerUnavailable   ErrorCode = "worker_unavailable"
	CodeInternal            ErrorCode = "internal"
)

type ErrorBody struct {
	Code         ErrorCode `json:"code"`
	Message      string    `json:"message"`
	Retryable    bool      `json:"retryable"`
	RetryAfterMs int       `json:"retryAfterMs,omitempty"`
	RequestID    string    `json:"requestId,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func HTTPStatus(code ErrorCode) int {
	switch code {
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeInvalidRequest:
		return http.StatusBadRequest
	case CodeUnsupportedLanguage:
		return http.StatusUnprocessableEntity
	case CodePayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeQueueFull:
		return http.StatusServiceUnavailable
	case CodeExpired:
		return http.StatusGatewayTimeout
	case CodeCancelled:
		return 499
	case CodeWorkerUnavailable:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func IsRetryable(code ErrorCode) bool {
	switch code {
	case CodeRateLimited, CodeQueueFull, CodeWorkerUnavailable:
		return true
	default:
		return false
	}
}