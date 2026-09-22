package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// The machine-readable half of the envelope. Clients branch on these, never on
// the message text. Named ErrCode because ErrorResponse.Code is the status.
const (
	ErrCodeNotFound            = "not_found"
	ErrCodeMethodNotAllowed    = "method_not_allowed"
	ErrCodeDatabaseUnavailable = "database_unavailable"
	ErrCodeInternalError       = "internal_error"
	ErrCodeInvalidBillID       = "invalid_bill_id"
	ErrCodeBillNotFound        = "bill_not_found"
	ErrCodeInvalidRequestBody  = "invalid_request_body"
	ErrCodeSharesMustSumTo100  = "shares_must_sum_to_100"
	ErrCodeDuplicatePerson     = "duplicate_person"
	ErrCodeUnknownPerson       = "unknown_person"
	ErrCodeInvalidPercentage   = "invalid_percentage"
	ErrCodeBillModified        = "bill_modified"
	ErrCodeInvalidName         = "invalid_name"
)

// ErrorResponse is the one shape every rejection takes.
type ErrorResponse struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
}

// AbortWithError writes the error envelope and stops the handler chain.
func AbortWithError(c *gin.Context, status int, errorCode, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Code:      status,
		ErrorCode: errorCode,
		Message:   message,
	})
}

// AbortWithDetailedError adds the field or id the client needs to act on.
func AbortWithDetailedError(c *gin.Context, status int, errorCode, message, detail string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Code:      status,
		ErrorCode: errorCode,
		Message:   message,
		Detail:    detail,
	})
}

// AbortWithInternalError is where anything unmapped lands. The cause is logged
// and not returned: driver messages name hosts, ports and SQL.
func AbortWithInternalError(c *gin.Context, logger *slog.Logger, err error) {
	logger.ErrorContext(c.Request.Context(), "unhandled error serving request",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.Any("error", err),
	)
	AbortWithError(c, http.StatusInternalServerError, ErrCodeInternalError, "something went wrong")
}
